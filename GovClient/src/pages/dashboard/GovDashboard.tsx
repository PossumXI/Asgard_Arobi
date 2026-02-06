/**
 * ASGARD Government Client - Government Dashboard
 * Main dashboard for government operations overview
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Activity,
  Satellite,
  Bot,
  Shield,
  Plane,
  AlertTriangle,
  TrendingUp,
  Globe,
} from 'lucide-react';

interface SystemStatus {
  name: string;
  status: 'online' | 'degraded' | 'offline';
  metrics: {
    uptime: string;
    latency: number;
    load: number;
  };
}

interface Alert {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  message: string;
  timestamp: string;
  source: string;
}

interface DashboardStats {
  activeMissions: number;
  systemsOnline: number;
  totalSystems: number;
  alertsCount: number;
  activeSatellites: number;
  activeHunoids: number;
}

const fetchDashboardData = async () => {
  const response = await fetch('/api/dashboard/gov/stats');
  if (!response.ok) throw new Error('Failed to fetch dashboard data');
  return response.json();
};

const fetchSystemStatus = async (): Promise<SystemStatus[]> => {
  const response = await fetch('/api/systems/status');
  if (!response.ok) throw new Error('Failed to fetch system status');
  return response.json();
};

const fetchAlerts = async (): Promise<Alert[]> => {
  const response = await fetch('/api/alerts?limit=5&severity=critical,warning');
  if (!response.ok) throw new Error('Failed to fetch alerts');
  return response.json();
};

export function GovDashboard() {
  const { data: stats } = useQuery<DashboardStats>({
    queryKey: ['dashboard-stats'],
    queryFn: fetchDashboardData,
    refetchInterval: 30000,
  });

  const { data: systems } = useQuery<SystemStatus[]>({
    queryKey: ['system-status'],
    queryFn: fetchSystemStatus,
    refetchInterval: 10000,
  });

  const { data: alerts } = useQuery<Alert[]>({
    queryKey: ['alerts'],
    queryFn: fetchAlerts,
    refetchInterval: 15000,
  });

  const statCards = [
    {
      label: 'Active Missions',
      value: stats?.activeMissions ?? 0,
      icon: Activity,
      color: 'from-emerald-500 to-teal-600',
    },
    {
      label: 'Systems Online',
      value: `${stats?.systemsOnline ?? 0}/${stats?.totalSystems ?? 0}`,
      icon: Globe,
      color: 'from-blue-500 to-cyan-600',
    },
    {
      label: 'Active Satellites',
      value: stats?.activeSatellites ?? 0,
      icon: Satellite,
      color: 'from-violet-500 to-purple-600',
    },
    {
      label: 'Active Hunoids',
      value: stats?.activeHunoids ?? 0,
      icon: Bot,
      color: 'from-amber-500 to-orange-600',
    },
  ];

  const systemIcons: Record<string, typeof Shield> = {
    Nysus: Globe,
    Silenus: Satellite,
    Hunoid: Bot,
    Giru: Shield,
    Valkyrie: Plane,
    Pricilla: TrendingUp,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Operations Dashboard</h1>
          <p className="text-slate-400">Real-time overview of all ASGARD systems</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-emerald-500/20 border border-emerald-500/50 rounded-lg">
          <div className="w-2 h-2 bg-emerald-500 rounded-full status-online" />
          <span className="text-emerald-400 text-sm font-medium">All Systems Operational</span>
        </div>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="bg-slate-800/50 border border-slate-700 rounded-xl p-6"
          >
            <div className="flex items-center justify-between mb-4">
              <div className={`w-12 h-12 rounded-lg bg-gradient-to-br ${stat.color} flex items-center justify-center`}>
                <stat.icon className="w-6 h-6 text-white" />
              </div>
            </div>
            <p className="text-3xl font-bold text-white mb-1">{stat.value}</p>
            <p className="text-sm text-slate-400">{stat.label}</p>
          </motion.div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* System Status */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4">System Status</h2>
          <div className="space-y-3">
            {systems?.map((system) => {
              const Icon = systemIcons[system.name] || Shield;
              const statusColors = {
                online: 'bg-emerald-500',
                degraded: 'bg-amber-500',
                offline: 'bg-red-500',
              };

              return (
                <div
                  key={system.name}
                  className="flex items-center justify-between p-3 bg-slate-900/50 rounded-lg"
                >
                  <div className="flex items-center gap-3">
                    <Icon className="w-5 h-5 text-slate-400" />
                    <span className="text-white font-medium">{system.name}</span>
                  </div>
                  <div className="flex items-center gap-4">
                    <span className="text-sm text-slate-400">
                      {system.metrics.latency}ms
                    </span>
                    <span className="text-sm text-slate-400">
                      {system.metrics.load}% load
                    </span>
                    <div
                      className={`w-2 h-2 rounded-full ${statusColors[system.status]}`}
                    />
                  </div>
                </div>
              );
            }) || (
              <div className="text-center py-8 text-slate-500">
                Loading system status...
              </div>
            )}
          </div>
        </div>

        {/* Recent Alerts */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Recent Alerts</h2>
          <div className="space-y-3">
            {alerts?.map((alert) => {
              const severityColors = {
                critical: 'bg-red-500/20 border-red-500/50 text-red-400',
                warning: 'bg-amber-500/20 border-amber-500/50 text-amber-400',
                info: 'bg-blue-500/20 border-blue-500/50 text-blue-400',
              };

              return (
                <div
                  key={alert.id}
                  className={`flex items-start gap-3 p-3 rounded-lg border ${severityColors[alert.severity]}`}
                >
                  <AlertTriangle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{alert.message}</p>
                    <p className="text-xs opacity-70 mt-1">
                      {alert.source} • {new Date(alert.timestamp).toLocaleTimeString()}
                    </p>
                  </div>
                </div>
              );
            }) || (
              <div className="text-center py-8 text-slate-500">
                No recent alerts
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
