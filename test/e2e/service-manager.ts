/**
 * ASGARD Service Manager
 *
 * Manages lifecycle of all ASGARD services for integration testing.
 * Provides health checks, auto-start, and graceful shutdown.
 *
 * DO-178C DAL-B Compliant: All services are validated before test execution.
 */

import { spawn, ChildProcess, execSync } from 'child_process';
import path from 'path';
import http from 'http';
import https from 'https';

// Service configuration
export interface ServiceConfig {
  name: string;
  displayName: string;
  executable: string;
  args: string[];
  healthEndpoint: string;
  port: number;
  env?: Record<string, string>;
  startupTimeout: number; // ms
  healthCheckInterval: number; // ms
  required: boolean; // If false, test continues even if service fails
}

// Service state
export interface ServiceState {
  name: string;
  status: 'stopped' | 'starting' | 'running' | 'error';
  process?: ChildProcess;
  pid?: number;
  healthStatus?: 'healthy' | 'unhealthy' | 'unknown';
  lastHealthCheck?: Date;
  error?: string;
  startTime?: Date;
  restartCount: number;
}

// All ASGARD services configuration
const BIN_DIR = path.resolve(__dirname, '../../bin');
const ASGARD_ROOT = path.resolve(__dirname, '../../');

export const ASGARD_SERVICES: ServiceConfig[] = [
  {
    name: 'valkyrie',
    displayName: 'Valkyrie Flight System',
    executable: path.join(BIN_DIR, 'valkyrie.exe'),
    args: ['-sim', '-http-port', '8093', '-metrics-port', '9093'],
    healthEndpoint: 'http://localhost:8093/health',
    port: 8093,
    startupTimeout: 15000,
    healthCheckInterval: 2000,
    required: true,
  },
  {
    name: 'giru',
    displayName: 'Giru Security System',
    executable: path.join(BIN_DIR, 'giru.exe'),
    args: ['-api-only', '-api-addr', ':9090', '-metrics-addr', ':9091'],
    healthEndpoint: 'http://localhost:9090/health',
    port: 9090,
    env: { ASGARD_ENV: 'development' },
    startupTimeout: 10000,
    healthCheckInterval: 2000,
    required: true,
  },
  {
    name: 'hunoid',
    displayName: 'Hunoid Robotics',
    executable: path.join(BIN_DIR, 'hunoid.exe'),
    args: ['-operator-ui-addr', ':8090', '-metrics-addr', ':9092', '-operator-mode', 'auto', '-stay-alive'],
    healthEndpoint: 'http://localhost:8090/api/status',
    port: 8090,
    env: { HUNOID_BYPASS_HARDWARE: '1' },
    startupTimeout: 15000,
    healthCheckInterval: 2000,
    required: true,
  },
  {
    name: 'pricilla',
    displayName: 'Pricilla Trajectory System',
    executable: path.join(BIN_DIR, 'pricilla.exe'),
    args: ['-http-port', '8089', '-metrics-port', '9089', '-enable-nats=false'],
    healthEndpoint: 'http://localhost:8089/health',
    port: 8089,
    startupTimeout: 15000,
    healthCheckInterval: 2000,
    required: true,
  },
  {
    name: 'vault',
    displayName: 'Security Vault',
    executable: path.join(BIN_DIR, 'vault.exe'),
    args: ['-http', ':8094', '-auto-unseal'],
    healthEndpoint: 'http://localhost:8094/vault/health',
    port: 8094,
    env: { VAULT_MASTER_PASSWORD: process.env.VAULT_MASTER_PASSWORD || 'asgard-dev-vault-2026' },
    startupTimeout: 10000,
    healthCheckInterval: 2000,
    required: true,
  },
  {
    name: 'notifications',
    displayName: 'Notification Service',
    executable: path.join(BIN_DIR, 'notifications.exe'),
    args: ['-http', ':8095'],
    healthEndpoint: 'http://localhost:8095/api/notifications/status',
    port: 8095,
    env: { RESEND_API_KEY: process.env.RESEND_API_KEY || '' },
    startupTimeout: 10000,
    healthCheckInterval: 2000,
    required: false, // Optional - works without API key
  },
  {
    name: 'nysus',
    displayName: 'Nysus Command System',
    executable: path.join(BIN_DIR, 'nysus.exe'),
    args: [],
    healthEndpoint: 'http://localhost:8080/health',
    port: 8080,
    startupTimeout: 10000,
    healthCheckInterval: 2000,
    required: false,
  },
  {
    name: 'silenus',
    displayName: 'Silenus Satellite System',
    executable: path.join(BIN_DIR, 'silenus.exe'),
    args: ['-metrics-addr', ':9094', '-vision-backend', 'simple'],
    healthEndpoint: 'http://localhost:9094/healthz',
    port: 9094,
    env: { SILENUS_BYPASS_HARDWARE: '1' },
    startupTimeout: 15000,
    healthCheckInterval: 2000,
    required: false,
  },
];

/**
 * Service Manager Class
 * Manages all ASGARD services lifecycle with health monitoring
 */
export class ServiceManager {
  private services: Map<string, ServiceState> = new Map();
  private logs: Map<string, string[]> = new Map();
  private healthCheckIntervals: Map<string, NodeJS.Timeout> = new Map();

  constructor() {
    // Initialize service states
    for (const config of ASGARD_SERVICES) {
      this.services.set(config.name, {
        name: config.name,
        status: 'stopped',
        healthStatus: 'unknown',
        restartCount: 0,
      });
      this.logs.set(config.name, []);
    }
  }

  /**
   * Check if a port is in use
   */
  private async isPortInUse(port: number): Promise<boolean> {
    return new Promise((resolve) => {
      const server = http.createServer();
      server.once('error', (err: NodeJS.ErrnoException) => {
        if (err.code === 'EADDRINUSE') {
          resolve(true);
        } else {
          resolve(false);
        }
      });
      server.once('listening', () => {
        server.close();
        resolve(false);
      });
      server.listen(port);
    });
  }

  /**
   * Check service health via HTTP endpoint
   */
  async checkHealth(config: ServiceConfig): Promise<boolean> {
    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        resolve(false);
      }, 3000);

      const url = new URL(config.healthEndpoint);
      const client = url.protocol === 'https:' ? https : http;

      const req = client.get(config.healthEndpoint, { timeout: 2500 }, (res) => {
        clearTimeout(timeout);
        resolve(res.statusCode === 200);
      });

      req.on('error', () => {
        clearTimeout(timeout);
        resolve(false);
      });
    });
  }

  /**
   * Wait for service to become healthy
   */
  private async waitForHealth(config: ServiceConfig): Promise<boolean> {
    const startTime = Date.now();
    while (Date.now() - startTime < config.startupTimeout) {
      const healthy = await this.checkHealth(config);
      if (healthy) {
        return true;
      }
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
    return false;
  }

  /**
   * Start a single service
   */
  async startService(config: ServiceConfig): Promise<boolean> {
    const state = this.services.get(config.name);
    if (!state) return false;

    // Check if already running
    const portInUse = await this.isPortInUse(config.port);
    if (portInUse) {
      const healthy = await this.checkHealth(config);
      if (healthy) {
        console.log(`[ServiceManager] ${config.displayName} already running on port ${config.port}`);
        state.status = 'running';
        state.healthStatus = 'healthy';
        state.lastHealthCheck = new Date();
        return true;
      }
    }

    // Check if executable exists
    try {
      const fs = await import('fs');
      if (!fs.existsSync(config.executable)) {
        console.log(`[ServiceManager] Executable not found: ${config.executable}`);
        console.log(`[ServiceManager] Building ${config.name}...`);
        try {
          // Try to build the service
          execSync(`go build -o ${config.executable} ./cmd/${config.name}`, {
            cwd: ASGARD_ROOT,
            stdio: 'pipe',
          });
        } catch (buildErr) {
          console.error(`[ServiceManager] Failed to build ${config.name}:`, buildErr);
          state.status = 'error';
          state.error = `Build failed: ${buildErr}`;
          return false;
        }
      }
    } catch (err) {
      state.status = 'error';
      state.error = `Failed to check executable: ${err}`;
      return false;
    }

    console.log(`[ServiceManager] Starting ${config.displayName}...`);
    state.status = 'starting';
    state.startTime = new Date();

    // Merge environment variables
    const env = { ...process.env, ...config.env };

    // Spawn the process
    const proc = spawn(config.executable, config.args, {
      cwd: ASGARD_ROOT,
      env,
      stdio: ['pipe', 'pipe', 'pipe'],
      detached: false,
      windowsHide: true,
    });

    state.process = proc;
    state.pid = proc.pid;

    // Capture logs
    const logs = this.logs.get(config.name) || [];

    proc.stdout?.on('data', (data) => {
      const line = data.toString().trim();
      if (line) {
        logs.push(`[OUT] ${line}`);
        if (logs.length > 500) logs.shift();
      }
    });

    proc.stderr?.on('data', (data) => {
      const line = data.toString().trim();
      if (line) {
        logs.push(`[ERR] ${line}`);
        if (logs.length > 500) logs.shift();
      }
    });

    proc.on('error', (err) => {
      console.error(`[ServiceManager] ${config.name} error:`, err);
      state.status = 'error';
      state.error = err.message;
    });

    proc.on('exit', (code) => {
      console.log(`[ServiceManager] ${config.name} exited with code ${code}`);
      if (state.status !== 'stopped') {
        state.status = 'error';
        state.error = `Process exited with code ${code}`;
      }
    });

    // Wait for service to become healthy
    const healthy = await this.waitForHealth(config);
    if (healthy) {
      console.log(`[ServiceManager] ${config.displayName} is healthy`);
      state.status = 'running';
      state.healthStatus = 'healthy';
      state.lastHealthCheck = new Date();
      return true;
    } else {
      console.error(`[ServiceManager] ${config.displayName} failed to become healthy`);
      state.status = 'error';
      state.error = 'Health check timeout';
      return false;
    }
  }

  /**
   * Stop a single service
   */
  async stopService(name: string): Promise<void> {
    const state = this.services.get(name);
    if (!state || !state.process) return;

    console.log(`[ServiceManager] Stopping ${name}...`);
    state.status = 'stopped';

    // Clear health check interval
    const interval = this.healthCheckIntervals.get(name);
    if (interval) {
      clearInterval(interval);
      this.healthCheckIntervals.delete(name);
    }

    // Kill the process
    if (state.process && !state.process.killed) {
      state.process.kill('SIGTERM');

      // Force kill after 5 seconds
      await new Promise<void>((resolve) => {
        const timeout = setTimeout(() => {
          if (state.process && !state.process.killed) {
            state.process.kill('SIGKILL');
          }
          resolve();
        }, 5000);

        state.process!.on('exit', () => {
          clearTimeout(timeout);
          resolve();
        });
      });
    }

    state.process = undefined;
    state.pid = undefined;
    console.log(`[ServiceManager] ${name} stopped`);
  }

  /**
   * Start all required services
   */
  async startAll(): Promise<{ success: boolean; failed: string[] }> {
    console.log('[ServiceManager] Starting all ASGARD services...');
    const failed: string[] = [];

    for (const config of ASGARD_SERVICES) {
      const success = await this.startService(config);
      if (!success && config.required) {
        failed.push(config.name);
      }
    }

    return {
      success: failed.length === 0,
      failed,
    };
  }

  /**
   * Stop all services
   */
  async stopAll(): Promise<void> {
    console.log('[ServiceManager] Stopping all services...');
    const stopPromises = ASGARD_SERVICES.map((config) => this.stopService(config.name));
    await Promise.all(stopPromises);
    console.log('[ServiceManager] All services stopped');
  }

  /**
   * Get service status
   */
  getStatus(): Map<string, ServiceState> {
    return this.services;
  }

  /**
   * Get service logs
   */
  getLogs(name: string): string[] {
    return this.logs.get(name) || [];
  }

  /**
   * Get all services health summary
   */
  async getHealthSummary(): Promise<Record<string, { status: string; healthy: boolean }>> {
    const summary: Record<string, { status: string; healthy: boolean }> = {};

    for (const config of ASGARD_SERVICES) {
      const state = this.services.get(config.name);
      const healthy = await this.checkHealth(config);
      summary[config.name] = {
        status: state?.status || 'unknown',
        healthy,
      };
    }

    return summary;
  }
}

// Singleton instance
let serviceManager: ServiceManager | null = null;

export function getServiceManager(): ServiceManager {
  if (!serviceManager) {
    serviceManager = new ServiceManager();
  }
  return serviceManager;
}

export async function ensureServicesRunning(): Promise<boolean> {
  const manager = getServiceManager();
  const result = await manager.startAll();

  if (!result.success) {
    console.error('[ServiceManager] Failed to start required services:', result.failed);
  }

  return result.success;
}

export async function shutdownServices(): Promise<void> {
  if (serviceManager) {
    await serviceManager.stopAll();
    serviceManager = null;
  }
}
