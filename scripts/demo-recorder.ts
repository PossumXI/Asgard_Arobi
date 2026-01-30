/**
 * ASGARD Demo Recorder
 * 
 * Comprehensive Playwright script to record full demos of all ASGARD systems.
 * Records videos, takes screenshots, and generates a manifest of all recordings.
 * 
 * Usage:
 *   npx ts-node demo-recorder.ts [options]
 * 
 * Options:
 *   --all           Record all demos (default)
 *   --website-only  Record only website demo
 *   --valkyrie-only Record only Valkyrie demo
 *   --pricilla-only Record only Pricilla demo
 *   --giru-only     Record only Giru JARVIS demo
 *   --headless      Run in headless mode (no visible browser)
 *   --slow          Add delays between actions for better visibility
 */

import { chromium, Browser, BrowserContext, Page } from 'playwright';
import * as fs from 'fs';
import * as path from 'path';

// ============================================================================
// Configuration
// ============================================================================

interface DemoConfig {
  outputDir: string;
  screenshotsDir: string;
  videosDir: string;
  manifestPath: string;
  viewport: { width: number; height: number };
  slowMo: number;
  headless: boolean;
  timeout: number;
}

interface RecordingEntry {
  id: string;
  name: string;
  description: string;
  type: 'video' | 'screenshot';
  path: string;
  timestamp: string;
  duration?: number;
  system: string;
  url?: string;
}

interface DemoManifest {
  version: string;
  generatedAt: string;
  totalDuration: number;
  recordings: RecordingEntry[];
  systems: {
    name: string;
    status: 'recorded' | 'skipped' | 'failed';
    error?: string;
  }[];
}

const DEFAULT_CONFIG: DemoConfig = {
  outputDir: path.join(__dirname, 'demo-output'),
  screenshotsDir: path.join(__dirname, 'demo-output', 'screenshots'),
  videosDir: path.join(__dirname, 'demo-output', 'videos'),
  manifestPath: path.join(__dirname, 'demo-output', 'manifest.json'),
  viewport: { width: 1920, height: 1080 },
  slowMo: 100,
  headless: false,
  timeout: 30000,
};

// Service URLs
const URLS = {
  website: 'http://localhost:3000',
  valkyrie: 'http://localhost:8093',
  pricilla: 'http://localhost:8092',
  giru: 'http://localhost:5000', // Giru JARVIS Flask server
};

// ============================================================================
// Utility Functions
// ============================================================================

function parseArgs(): { 
  websiteOnly: boolean; 
  valkyrieOnly: boolean; 
  pricillaOnly: boolean; 
  giruOnly: boolean;
  headless: boolean;
  slow: boolean;
  all: boolean;
} {
  const args = process.argv.slice(2);
  return {
    websiteOnly: args.includes('--website-only'),
    valkyrieOnly: args.includes('--valkyrie-only'),
    pricillaOnly: args.includes('--pricilla-only'),
    giruOnly: args.includes('--giru-only'),
    headless: args.includes('--headless'),
    slow: args.includes('--slow'),
    all: args.includes('--all') || args.length === 0,
  };
}

function ensureDirectories(config: DemoConfig): void {
  [config.outputDir, config.screenshotsDir, config.videosDir].forEach(dir => {
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
      console.log(`📁 Created directory: ${dir}`);
    }
  });
}

function timestamp(): string {
  return new Date().toISOString().replace(/[:.]/g, '-');
}

async function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function checkServiceHealth(url: string): Promise<boolean> {
  try {
    const response = await fetch(url, { method: 'GET', signal: AbortSignal.timeout(5000) });
    return response.ok || response.status < 500;
  } catch {
    return false;
  }
}

class DemoRecorder {
  private config: DemoConfig;
  private manifest: DemoManifest;
  private browser: Browser | null = null;
  private startTime: number = 0;
  private slowMode: boolean;

  constructor(config: DemoConfig, slowMode: boolean = false) {
    this.config = config;
    this.slowMode = slowMode;
    this.manifest = {
      version: '1.0.0',
      generatedAt: new Date().toISOString(),
      totalDuration: 0,
      recordings: [],
      systems: [],
    };
  }

  async initialize(): Promise<void> {
    ensureDirectories(this.config);
    this.startTime = Date.now();

    console.log('\n🎬 ASGARD Demo Recorder');
    console.log('=' .repeat(50));
    console.log(`📅 Started at: ${new Date().toLocaleString()}`);
    console.log(`📁 Output directory: ${this.config.outputDir}`);
    console.log(`🖥️  Viewport: ${this.config.viewport.width}x${this.config.viewport.height}`);
    console.log(`🐢 Slow mode: ${this.slowMode ? 'enabled' : 'disabled'}`);
    console.log('=' .repeat(50) + '\n');

    this.browser = await chromium.launch({
      headless: this.config.headless,
      slowMo: this.slowMode ? 500 : this.config.slowMo,
    });
  }

  async createContext(videoName: string): Promise<BrowserContext> {
    if (!this.browser) throw new Error('Browser not initialized');

    const videoPath = path.join(this.config.videosDir, videoName);
    
    return await this.browser.newContext({
      viewport: this.config.viewport,
      recordVideo: {
        dir: videoPath,
        size: this.config.viewport,
      },
      ignoreHTTPSErrors: true,
    });
  }

  async takeScreenshot(page: Page, name: string, description: string, system: string): Promise<void> {
    const filename = `${timestamp()}-${name}.png`;
    const filepath = path.join(this.config.screenshotsDir, filename);
    
    await page.screenshot({ 
      path: filepath, 
      fullPage: false,
      animations: 'disabled',
    });

    this.addRecording({
      id: `screenshot-${name}-${timestamp()}`,
      name,
      description,
      type: 'screenshot',
      path: `screenshots/${filename}`,
      timestamp: new Date().toISOString(),
      system,
      url: page.url(),
    });

    console.log(`   📸 Screenshot: ${name}`);
  }

  addRecording(entry: RecordingEntry): void {
    this.manifest.recordings.push(entry);
  }

  addSystemStatus(name: string, status: 'recorded' | 'skipped' | 'failed', error?: string): void {
    this.manifest.systems.push({ name, status, error });
  }

  // ==========================================================================
  // Website Demo (localhost:3000)
  // ==========================================================================

  async recordWebsiteDemo(): Promise<void> {
    console.log('\n🌐 Recording Website Demo (localhost:3000)');
    console.log('-'.repeat(40));

    const isHealthy = await checkServiceHealth(URLS.website);
    if (!isHealthy) {
      console.log('   ⚠️  Website not available, skipping...');
      this.addSystemStatus('Website', 'skipped', 'Service not available');
      return;
    }

    const context = await this.createContext('website');
    const page = await context.newPage();
    const demoStart = Date.now();

    try {
      // Landing Page
      console.log('   📍 Landing page...');
      await page.goto(URLS.website, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'website-landing', 'ASGARD Landing Page', 'Website');

      // Scroll to showcase sections
      await page.evaluate(() => window.scrollTo({ top: 500, behavior: 'smooth' }));
      await sleep(1000);
      await this.takeScreenshot(page, 'website-features', 'Features Section', 'Website');

      // Navigate to Valkyrie page
      console.log('   📍 Valkyrie page...');
      try {
        await page.click('text=Valkyrie', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-valkyrie', 'Valkyrie Product Page', 'Website');
      } catch {
        console.log('      ⚠️  Valkyrie link not found, trying direct navigation...');
        await page.goto(`${URLS.website}/valkyrie`, { waitUntil: 'networkidle' });
        await this.takeScreenshot(page, 'website-valkyrie', 'Valkyrie Product Page', 'Website');
      }

      // Navigate to Pricilla page
      console.log('   📍 Pricilla page...');
      try {
        await page.click('text=Pricilla', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-pricilla', 'Pricilla Product Page', 'Website');
      } catch {
        console.log('      ⚠️  Pricilla link not found, trying direct navigation...');
        await page.goto(`${URLS.website}/pricilla`, { waitUntil: 'networkidle' });
        await this.takeScreenshot(page, 'website-pricilla', 'Pricilla Product Page', 'Website');
      }

      // Navigate to Giru page
      console.log('   📍 Giru page...');
      try {
        await page.click('text=Giru', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-giru', 'Giru Product Page', 'Website');
      } catch {
        console.log('      ⚠️  Giru link not found, trying direct navigation...');
        await page.goto(`${URLS.website}/giru`, { waitUntil: 'networkidle' });
        await this.takeScreenshot(page, 'website-giru', 'Giru Product Page', 'Website');
      }

      // Navigate to Contact page
      console.log('   📍 Contact page...');
      try {
        await page.click('text=Contact', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-contact', 'Contact Page', 'Website');
      } catch {
        console.log('      ⚠️  Contact link not found, trying direct navigation...');
        await page.goto(`${URLS.website}/contact`, { waitUntil: 'networkidle' });
        await this.takeScreenshot(page, 'website-contact', 'Contact Page', 'Website');
      }

      // Navigate to Pricing page
      console.log('   📍 Pricing page...');
      try {
        await page.click('text=Pricing', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-pricing', 'Pricing Page', 'Website');
      } catch {
        console.log('      ⚠️  Pricing link not found, trying direct navigation...');
        await page.goto(`${URLS.website}/pricing`, { waitUntil: 'networkidle' });
        await this.takeScreenshot(page, 'website-pricing', 'Pricing Page', 'Website');
      }

      // Sign-up flow
      console.log('   📍 Sign-up flow...');
      try {
        await page.click('text=Sign Up', { timeout: 5000 });
        await page.waitForLoadState('networkidle');
        await sleep(this.slowMode ? 2000 : 1000);
        await this.takeScreenshot(page, 'website-signup', 'Sign Up Page', 'Website');
        
        // Fill demo data (but don't submit)
        const emailInput = page.locator('input[type="email"], input[name="email"]').first();
        if (await emailInput.isVisible()) {
          await emailInput.fill('demo@asgard.systems');
          await sleep(500);
        }
        
        const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
        if (await passwordInput.isVisible()) {
          await passwordInput.fill('DemoPassword123!');
          await sleep(500);
        }
        
        await this.takeScreenshot(page, 'website-signup-filled', 'Sign Up Form Filled', 'Website');
      } catch {
        console.log('      ⚠️  Sign up flow not available');
        await page.goto(`${URLS.website}/signup`, { waitUntil: 'networkidle' }).catch(() => {});
        await this.takeScreenshot(page, 'website-signup', 'Sign Up Page', 'Website');
      }

      const duration = (Date.now() - demoStart) / 1000;
      console.log(`   ✅ Website demo completed (${duration.toFixed(1)}s)`);
      this.addSystemStatus('Website', 'recorded');

      // Add video recording entry
      this.addRecording({
        id: `video-website-${timestamp()}`,
        name: 'website-demo',
        description: 'Full website navigation demo',
        type: 'video',
        path: 'videos/website/',
        timestamp: new Date().toISOString(),
        duration,
        system: 'Website',
        url: URLS.website,
      });

    } catch (error) {
      console.error(`   ❌ Website demo failed: ${error}`);
      this.addSystemStatus('Website', 'failed', String(error));
    } finally {
      await page.close();
      await context.close();
    }
  }

  // ==========================================================================
  // Valkyrie Demo (localhost:8093)
  // ==========================================================================

  async recordValkyrieDemo(): Promise<void> {
    console.log('\n⚔️  Recording Valkyrie Demo (localhost:8093)');
    console.log('-'.repeat(40));

    const isHealthy = await checkServiceHealth(`${URLS.valkyrie}/health`);
    if (!isHealthy) {
      console.log('   ⚠️  Valkyrie not available, skipping...');
      this.addSystemStatus('Valkyrie', 'skipped', 'Service not available');
      return;
    }

    const context = await this.createContext('valkyrie');
    const page = await context.newPage();
    const demoStart = Date.now();

    try {
      // Health check endpoint
      console.log('   📍 Health check endpoint...');
      await page.goto(`${URLS.valkyrie}/health`, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'valkyrie-health', 'Valkyrie Health Check', 'Valkyrie');

      // Status endpoint
      console.log('   📍 Status endpoint...');
      await page.goto(`${URLS.valkyrie}/status`, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'valkyrie-status', 'Valkyrie Status', 'Valkyrie');

      // State endpoint
      console.log('   📍 State endpoint...');
      await page.goto(`${URLS.valkyrie}/state`, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'valkyrie-state', 'Valkyrie State', 'Valkyrie');

      // API endpoints
      console.log('   📍 API endpoints...');
      await page.goto(`${URLS.valkyrie}/api/v1/sensors`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'valkyrie-sensors', 'Valkyrie Sensors API', 'Valkyrie');

      await page.goto(`${URLS.valkyrie}/api/v1/navigation`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'valkyrie-navigation', 'Valkyrie Navigation API', 'Valkyrie');

      const duration = (Date.now() - demoStart) / 1000;
      console.log(`   ✅ Valkyrie demo completed (${duration.toFixed(1)}s)`);
      this.addSystemStatus('Valkyrie', 'recorded');

      this.addRecording({
        id: `video-valkyrie-${timestamp()}`,
        name: 'valkyrie-demo',
        description: 'Valkyrie autonomous vehicle system demo',
        type: 'video',
        path: 'videos/valkyrie/',
        timestamp: new Date().toISOString(),
        duration,
        system: 'Valkyrie',
        url: URLS.valkyrie,
      });

    } catch (error) {
      console.error(`   ❌ Valkyrie demo failed: ${error}`);
      this.addSystemStatus('Valkyrie', 'failed', String(error));
    } finally {
      await page.close();
      await context.close();
    }
  }

  // ==========================================================================
  // Pricilla Demo (localhost:8092)
  // ==========================================================================

  async recordPricillaDemo(): Promise<void> {
    console.log('\n🎯 Recording Pricilla Demo (localhost:8092)');
    console.log('-'.repeat(40));

    const isHealthy = await checkServiceHealth(`${URLS.pricilla}/health`);
    if (!isHealthy) {
      console.log('   ⚠️  Pricilla not available, skipping...');
      this.addSystemStatus('Pricilla', 'skipped', 'Service not available');
      return;
    }

    const context = await this.createContext('pricilla');
    const page = await context.newPage();
    const demoStart = Date.now();

    try {
      // Health check
      console.log('   📍 Health check...');
      await page.goto(`${URLS.pricilla}/health`, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'pricilla-health', 'Pricilla Health Check', 'Pricilla');

      // Status endpoint
      console.log('   📍 Status endpoint...');
      await page.goto(`${URLS.pricilla}/status`, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'pricilla-status', 'Pricilla Status', 'Pricilla');

      // Targeting API
      console.log('   📍 Targeting API...');
      await page.goto(`${URLS.pricilla}/api/v1/targeting`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'pricilla-targeting', 'Pricilla Targeting API', 'Pricilla');

      // Metrics endpoint
      console.log('   📍 Metrics endpoint...');
      await page.goto(`${URLS.pricilla}/metrics`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'pricilla-metrics', 'Pricilla Metrics', 'Pricilla');

      const duration = (Date.now() - demoStart) / 1000;
      console.log(`   ✅ Pricilla demo completed (${duration.toFixed(1)}s)`);
      this.addSystemStatus('Pricilla', 'recorded');

      this.addRecording({
        id: `video-pricilla-${timestamp()}`,
        name: 'pricilla-demo',
        description: 'Pricilla ad targeting system demo',
        type: 'video',
        path: 'videos/pricilla/',
        timestamp: new Date().toISOString(),
        duration,
        system: 'Pricilla',
        url: URLS.pricilla,
      });

    } catch (error) {
      console.error(`   ❌ Pricilla demo failed: ${error}`);
      this.addSystemStatus('Pricilla', 'failed', String(error));
    } finally {
      await page.close();
      await context.close();
    }
  }

  // ==========================================================================
  // Giru JARVIS Demo (localhost:5000 or Electron)
  // ==========================================================================

  async recordGiruDemo(): Promise<void> {
    console.log('\n🤖 Recording Giru JARVIS Demo (localhost:5000)');
    console.log('-'.repeat(40));

    const isHealthy = await checkServiceHealth(URLS.giru);
    if (!isHealthy) {
      console.log('   ⚠️  Giru JARVIS not available, skipping...');
      this.addSystemStatus('Giru JARVIS', 'skipped', 'Service not available');
      return;
    }

    const context = await this.createContext('giru');
    const page = await context.newPage();
    const demoStart = Date.now();

    try {
      // Main JARVIS UI
      console.log('   📍 JARVIS main UI...');
      await page.goto(URLS.giru, { waitUntil: 'networkidle', timeout: this.config.timeout });
      await sleep(this.slowMode ? 3000 : 1500);
      await this.takeScreenshot(page, 'giru-jarvis-main', 'Giru JARVIS Main Interface', 'Giru');

      // Monitor dashboard
      console.log('   📍 Monitor dashboard...');
      await page.goto(`${URLS.giru}/monitor`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(this.slowMode ? 2000 : 1000);
      await this.takeScreenshot(page, 'giru-monitor', 'Giru Monitor Dashboard', 'Giru');

      // Status page
      console.log('   📍 Status page...');
      await page.goto(`${URLS.giru}/status`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'giru-status', 'Giru Status Page', 'Giru');

      // API endpoints
      console.log('   📍 API endpoints...');
      await page.goto(`${URLS.giru}/api/health`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'giru-api-health', 'Giru API Health', 'Giru');

      // Voice assistant interface (if available)
      console.log('   📍 Voice assistant...');
      await page.goto(`${URLS.giru}/assistant`, { waitUntil: 'networkidle' }).catch(() => {});
      await sleep(500);
      await this.takeScreenshot(page, 'giru-assistant', 'Giru Voice Assistant', 'Giru');

      const duration = (Date.now() - demoStart) / 1000;
      console.log(`   ✅ Giru JARVIS demo completed (${duration.toFixed(1)}s)`);
      this.addSystemStatus('Giru JARVIS', 'recorded');

      this.addRecording({
        id: `video-giru-${timestamp()}`,
        name: 'giru-jarvis-demo',
        description: 'Giru JARVIS AI assistant demo',
        type: 'video',
        path: 'videos/giru/',
        timestamp: new Date().toISOString(),
        duration,
        system: 'Giru',
        url: URLS.giru,
      });

    } catch (error) {
      console.error(`   ❌ Giru JARVIS demo failed: ${error}`);
      this.addSystemStatus('Giru JARVIS', 'failed', String(error));
    } finally {
      await page.close();
      await context.close();
    }
  }

  // ==========================================================================
  // Finalization
  // ==========================================================================

  async finalize(): Promise<void> {
    this.manifest.totalDuration = (Date.now() - this.startTime) / 1000;

    // Write manifest
    fs.writeFileSync(
      this.config.manifestPath,
      JSON.stringify(this.manifest, null, 2),
      'utf-8'
    );

    if (this.browser) {
      await this.browser.close();
    }

    console.log('\n' + '=' .repeat(50));
    console.log('🎬 DEMO RECORDING COMPLETE');
    console.log('=' .repeat(50));
    console.log(`📊 Total duration: ${this.manifest.totalDuration.toFixed(1)} seconds`);
    console.log(`📸 Screenshots: ${this.manifest.recordings.filter(r => r.type === 'screenshot').length}`);
    console.log(`🎥 Videos: ${this.manifest.recordings.filter(r => r.type === 'video').length}`);
    console.log(`📁 Output: ${this.config.outputDir}`);
    console.log(`📋 Manifest: ${this.config.manifestPath}`);
    console.log('\nSystem Status:');
    this.manifest.systems.forEach(sys => {
      const icon = sys.status === 'recorded' ? '✅' : sys.status === 'skipped' ? '⚠️' : '❌';
      console.log(`   ${icon} ${sys.name}: ${sys.status}${sys.error ? ` (${sys.error})` : ''}`);
    });
    console.log('=' .repeat(50) + '\n');
  }
}

// ============================================================================
// Main Execution
// ============================================================================

async function main(): Promise<void> {
  const args = parseArgs();
  
  const config: DemoConfig = {
    ...DEFAULT_CONFIG,
    headless: args.headless,
  };

  const recorder = new DemoRecorder(config, args.slow);
  
  try {
    await recorder.initialize();

    // Determine which demos to record
    const shouldRecordAll = args.all && !args.websiteOnly && !args.valkyrieOnly && !args.pricillaOnly && !args.giruOnly;

    if (shouldRecordAll || args.websiteOnly) {
      await recorder.recordWebsiteDemo();
    }

    if (shouldRecordAll || args.valkyrieOnly) {
      await recorder.recordValkyrieDemo();
    }

    if (shouldRecordAll || args.pricillaOnly) {
      await recorder.recordPricillaDemo();
    }

    if (shouldRecordAll || args.giruOnly) {
      await recorder.recordGiruDemo();
    }

    await recorder.finalize();

  } catch (error) {
    console.error('\n❌ Fatal error:', error);
    process.exit(1);
  }
}

// Run if executed directly
main().catch(console.error);

export { DemoRecorder, DemoConfig, DemoManifest, RecordingEntry };
