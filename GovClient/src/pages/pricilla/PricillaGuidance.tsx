/**
 * ASGARD Government Client - Pricilla Guidance
 * Precision guidance system interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Crosshair,
  Navigation,
  Target,
  MapPin,
  Activity,
  Zap,
} from 'lucide-react';

interface GuidanceMission {
  id: string;
  name: string;
  type: 'hunoid' | 'uav' | 'missile' | 'spacecraft';
  status: 'planning' | 'active' | 'terminal' | 'completed' | 'aborted';
  target: { lat: number; lng: number; alt: number };
  currentPosition: { lat: number; lng: number; alt: number };
  progress: number;
  eta: string;
  hitProbability: number;
}

export function PricillaGuidance() {
  const { data: missions } = useQuery<GuidanceMission[]>({
    queryKey: ['guidance-missions'],
    queryFn: async () => {
      const response = await fetch('/api/pricilla/api/v1/missions');
      return response.json();
    },
    refetchInterval: 2000,
  });

  const activeMissions = missions?.filter((m) => m.status === 'active' || m.status === 'terminal') ?? [];

  const getStatusColor = (status: GuidanceMission['status']) => {
    const colors = {
      planning: 'bg-slate-500/20 text-slate-400 border-slate-500/50',
      active: 'bg-blue-500/20 text-blue-400 border-blue-500/50',
      terminal: 'bg-amber-500/20 text-amber-400 border-amber-500/50',
      completed: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/50',
      aborted: 'bg-red-500/20 text-red-400 border-red-500/50',
    };
    return colors[status];
  };

  const getTypeIcon = (type: GuidanceMission['type']) => {
    const icons = {
      hunoid: '🤖',
      uav: '✈️',
      missile: '🚀',
      spacecraft: '🛸',
    };
    return icons[type];
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Pricilla Guidance</h1>
          <p className="text-slate-400">Precision navigation and targeting system</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-violet-500/20 border border-violet-500/50 rounded-lg">
          <Crosshair className="w-4 h-4 text-violet-400" />
          <span className="text-violet-400">{activeMissions.length} Active Guidance</span>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Active Missions', value: activeMissions.length, icon: Target, color: 'from-violet-500 to-purple-600' },
          { label: 'Terminal Phase', value: missions?.filter((m) => m.status === 'terminal').length ?? 0, icon: Zap, color: 'from-amber-500 to-orange-600' },
          { label: 'Completed', value: missions?.filter((m) => m.status === 'completed').length ?? 0, icon: Navigation, color: 'from-emerald-500 to-teal-600' },
          { label: 'Avg Hit Prob', value: `${Math.round((missions?.reduce((sum, m) => sum + m.hitProbability, 0) ?? 0) / (missions?.length || 1) * 100)}%`, icon: Crosshair, color: 'from-blue-500 to-cyan-600' },
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
                <p className="text-2xl font-bold text-white">{stat.value}</p>
                <p className="text-sm text-slate-400">{stat.label}</p>
              </div>
              <div className={`w-10 h-10 rounded-lg bg-gradient-to-br ${stat.color} flex items-center justify-center`}>
                <stat.icon className="w-5 h-5 text-white" />
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {/* Active Missions */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
        <h2 className="text-lg font-semibold text-white mb-4">Guidance Missions</h2>
        <div className="space-y-4">
          {missions?.map((mission) => (
            <motion.div
              key={mission.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              className="p-4 bg-slate-900/50 rounded-lg"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <span className="text-2xl">{getTypeIcon(mission.type)}</span>
                  <div>
                    <p className="font-medium text-white">{mission.name}</p>
                    <span className={`px-2 py-0.5 text-xs rounded border ${getStatusColor(mission.status)}`}>
                      {mission.status}
                    </span>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm text-slate-400">Hit Probability</p>
                  <p className="text-lg font-bold text-white">{(mission.hitProbability * 100).toFixed(1)}%</p>
                </div>
              </div>

              {/* Progress Bar */}
              <div className="mb-3">
                <div className="flex items-center justify-between text-sm mb-1">
                  <span className="text-slate-400">Progress</span>
                  <span className="text-white">{(mission.progress * 100).toFixed(0)}%</span>
                </div>
                <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                  <motion.div
                    initial={{ width: 0 }}
                    animate={{ width: `${mission.progress * 100}%` }}
                    className={`h-full ${
                      mission.status === 'terminal' ? 'bg-amber-500' : 'bg-violet-500'
                    }`}
                  />
                </div>
              </div>

              {/* Position Info */}
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="flex items-center gap-2 text-slate-400">
                  <MapPin className="w-4 h-4" />
                  <span>
                    Current: {mission.currentPosition.lat.toFixed(4)}, {mission.currentPosition.lng.toFixed(4)}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-slate-400">
                  <Target className="w-4 h-4" />
                  <span>
                    Target: {mission.target.lat.toFixed(4)}, {mission.target.lng.toFixed(4)}
                  </span>
                </div>
              </div>

              {mission.eta && (
                <div className="mt-2 text-sm text-slate-500">
                  ETA: {mission.eta}
                </div>
              )}
            </motion.div>
          )) || (
            <div className="text-center py-8 text-slate-500">
              No guidance missions
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
