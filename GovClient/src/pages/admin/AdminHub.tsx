/**
 * ASGARD Government Client - Admin Hub
 * Administrative controls and system management
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Settings,
  Users,
  Shield,
  Database,
  Activity,
  Key,
  RefreshCw,
} from 'lucide-react';

interface SystemHealth {
  service: string;
  status: 'healthy' | 'degraded' | 'down';
  uptime: string;
  version: string;
}

export function AdminHub() {
  const [activeTab, setActiveTab] = useState<'users' | 'systems' | 'security' | 'logs'>('systems');

  const { data: health } = useQuery<SystemHealth[]>({
    queryKey: ['system-health'],
    queryFn: async () => {
      const response = await fetch('/api/admin/health');
      return response.json();
    },
    refetchInterval: 30000,
  });

  const tabs = [
    { id: 'systems', label: 'Systems', icon: Database },
    { id: 'users', label: 'Users', icon: Users },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'logs', label: 'Logs', icon: Activity },
  ] as const;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Admin Hub</h1>
          <p className="text-slate-400">System administration and configuration</p>
        </div>
        <button className="flex items-center gap-2 px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors">
          <RefreshCw className="w-4 h-4" />
          Refresh All
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-slate-700 pb-2">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              activeTab === tab.id
                ? 'bg-amber-500 text-white'
                : 'text-slate-400 hover:bg-slate-800 hover:text-white'
            }`}
          >
            <tab.icon className="w-4 h-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
        {activeTab === 'systems' && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-white mb-4">System Health</h2>
            <div className="grid gap-4">
              {health?.map((service) => (
                <motion.div
                  key={service.service}
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="flex items-center justify-between p-4 bg-slate-900/50 rounded-lg"
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`w-3 h-3 rounded-full ${
                        service.status === 'healthy'
                          ? 'bg-emerald-500'
                          : service.status === 'degraded'
                          ? 'bg-amber-500'
                          : 'bg-red-500'
                      }`}
                    />
                    <div>
                      <p className="text-white font-medium">{service.service}</p>
                      <p className="text-sm text-slate-500">v{service.version}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm text-slate-400">Uptime</p>
                    <p className="text-white">{service.uptime}</p>
                  </div>
                </motion.div>
              )) || (
                <div className="text-center py-8 text-slate-500">
                  Loading system health...
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'users' && (
          <div className="text-center py-20 text-slate-500">
            <Users className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>User management interface</p>
          </div>
        )}

        {activeTab === 'security' && (
          <div className="space-y-4">
            <h2 className="text-lg font-semibold text-white mb-4">Security Settings</h2>
            <div className="grid gap-4">
              <div className="flex items-center justify-between p-4 bg-slate-900/50 rounded-lg">
                <div className="flex items-center gap-3">
                  <Key className="w-5 h-5 text-amber-500" />
                  <div>
                    <p className="text-white font-medium">Access Codes</p>
                    <p className="text-sm text-slate-500">Manage government access codes</p>
                  </div>
                </div>
                <button className="px-4 py-2 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30 transition-colors">
                  Manage
                </button>
              </div>
              <div className="flex items-center justify-between p-4 bg-slate-900/50 rounded-lg">
                <div className="flex items-center gap-3">
                  <Shield className="w-5 h-5 text-emerald-500" />
                  <div>
                    <p className="text-white font-medium">FIDO2 Keys</p>
                    <p className="text-sm text-slate-500">Hardware security key management</p>
                  </div>
                </div>
                <button className="px-4 py-2 bg-emerald-500/20 text-emerald-400 rounded-lg hover:bg-emerald-500/30 transition-colors">
                  Configure
                </button>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'logs' && (
          <div className="text-center py-20 text-slate-500">
            <Activity className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>System logs and audit trail</p>
          </div>
        )}
      </div>
    </div>
  );
}
