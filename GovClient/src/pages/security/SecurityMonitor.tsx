/**
 * ASGARD Government Client - Security Monitor
 * Giru security system integration
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Shield,
  AlertTriangle,
  Eye,
  Lock,
  Unlock,
  Activity,
  Wifi,
  WifiOff,
} from 'lucide-react';

interface ThreatZone {
  id: string;
  name: string;
  level: 'low' | 'medium' | 'high' | 'critical';
  location: { lat: number; lng: number };
  radius: number;
  activeThreats: number;
}

interface SecurityEvent {
  id: string;
  type: string;
  severity: 'info' | 'warning' | 'critical';
  message: string;
  timestamp: string;
  source: string;
}

export function SecurityMonitor() {
  const { data: threatZones } = useQuery<ThreatZone[]>({
    queryKey: ['threat-zones'],
    queryFn: async () => {
      const response = await fetch('/api/security/threat-zones');
      return response.json();
    },
    refetchInterval: 10000,
  });

  const { data: events } = useQuery<SecurityEvent[]>({
    queryKey: ['security-events'],
    queryFn: async () => {
      const response = await fetch('/api/security/events?limit=10');
      return response.json();
    },
    refetchInterval: 5000,
  });

  const getThreatLevelColor = (level: ThreatZone['level']) => {
    const colors = {
      low: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/50',
      medium: 'bg-amber-500/20 text-amber-400 border-amber-500/50',
      high: 'bg-orange-500/20 text-orange-400 border-orange-500/50',
      critical: 'bg-red-500/20 text-red-400 border-red-500/50',
    };
    return colors[level];
  };

  const totalThreats = threatZones?.reduce((sum, zone) => sum + zone.activeThreats, 0) ?? 0;
  const criticalZones = threatZones?.filter((z) => z.level === 'critical').length ?? 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Security Monitor</h1>
          <p className="text-slate-400">Giru AI Defense System</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/20 border border-emerald-500/50 rounded-lg">
            <Wifi className="w-4 h-4 text-emerald-400" />
            <span className="text-sm text-emerald-400">Giru Connected</span>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Active Threats', value: totalThreats, icon: AlertTriangle, color: totalThreats > 0 ? 'text-red-400' : 'text-emerald-400' },
          { label: 'Critical Zones', value: criticalZones, icon: Shield, color: criticalZones > 0 ? 'text-red-400' : 'text-emerald-400' },
          { label: 'Monitored Areas', value: threatZones?.length ?? 0, icon: Eye, color: 'text-blue-400' },
          { label: 'Security Status', value: totalThreats === 0 ? 'SECURE' : 'ALERT', icon: totalThreats === 0 ? Lock : Unlock, color: totalThreats === 0 ? 'text-emerald-400' : 'text-amber-400' },
        ].map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className={`text-2xl font-bold ${stat.color}`}>{stat.value}</p>
                <p className="text-sm text-slate-400">{stat.label}</p>
              </div>
              <stat.icon className={`w-8 h-8 ${stat.color} opacity-50`} />
            </div>
          </motion.div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Threat Zones */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Threat Zones</h2>
          <div className="space-y-3">
            {threatZones?.map((zone) => (
              <div
                key={zone.id}
                className={`flex items-center justify-between p-4 rounded-lg border ${getThreatLevelColor(zone.level)}`}
              >
                <div>
                  <p className="font-medium">{zone.name}</p>
                  <p className="text-sm opacity-70">
                    {zone.location.lat.toFixed(4)}, {zone.location.lng.toFixed(4)}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-bold">{zone.activeThreats}</p>
                  <p className="text-xs uppercase">{zone.level}</p>
                </div>
              </div>
            )) || (
              <div className="text-center py-8 text-slate-500">
                No threat zones detected
              </div>
            )}
          </div>
        </div>

        {/* Security Events */}
        <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Recent Events</h2>
          <div className="space-y-3 max-h-96 overflow-y-auto">
            {events?.map((event) => (
              <div
                key={event.id}
                className="flex items-start gap-3 p-3 bg-slate-900/50 rounded-lg"
              >
                <Activity
                  className={`w-4 h-4 mt-0.5 ${
                    event.severity === 'critical'
                      ? 'text-red-400'
                      : event.severity === 'warning'
                      ? 'text-amber-400'
                      : 'text-blue-400'
                  }`}
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-white">{event.message}</p>
                  <p className="text-xs text-slate-500">
                    {event.source} • {new Date(event.timestamp).toLocaleTimeString()}
                  </p>
                </div>
              </div>
            )) || (
              <div className="text-center py-8 text-slate-500">
                No recent events
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
