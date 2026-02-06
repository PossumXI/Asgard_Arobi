/**
 * ASGARD Government Client - Hunoid Control
 * Humanoid robotics management interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Bot,
  Battery,
  MapPin,
  Activity,
  Play,
  Pause,
  RotateCcw,
  Users,
} from 'lucide-react';
import { toast } from '@/components/ui/Toaster';

interface HunoidUnit {
  id: string;
  name: string;
  status: 'idle' | 'active' | 'mission' | 'maintenance' | 'offline';
  battery: number;
  position: { lat: number; lng: number };
  currentMission?: string;
  lastUpdate: string;
}

export function HunoidControl() {
  const [selectedUnit, setSelectedUnit] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data: units, isLoading } = useQuery<HunoidUnit[]>({
    queryKey: ['hunoid-units'],
    queryFn: async () => {
      const response = await fetch('/api/hunoids');
      return response.json();
    },
    refetchInterval: 5000,
  });

  const deployMutation = useMutation({
    mutationFn: async (unitId: string) => {
      const response = await fetch(`/api/hunoids/${unitId}/deploy`, { method: 'POST' });
      if (!response.ok) throw new Error('Failed to deploy');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hunoid-units'] });
      toast({ type: 'success', title: 'Unit deployed successfully' });
    },
    onError: () => toast({ type: 'error', title: 'Failed to deploy unit' }),
  });

  const recallMutation = useMutation({
    mutationFn: async (unitId: string) => {
      const response = await fetch(`/api/hunoids/${unitId}/recall`, { method: 'POST' });
      if (!response.ok) throw new Error('Failed to recall');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hunoid-units'] });
      toast({ type: 'warning', title: 'Unit recalled' });
    },
    onError: () => toast({ type: 'error', title: 'Failed to recall unit' }),
  });

  const getStatusColor = (status: HunoidUnit['status']) => {
    const colors = {
      idle: 'bg-slate-500',
      active: 'bg-emerald-500',
      mission: 'bg-blue-500',
      maintenance: 'bg-amber-500',
      offline: 'bg-red-500',
    };
    return colors[status];
  };

  const activeUnits = units?.filter((u) => u.status === 'active' || u.status === 'mission').length ?? 0;
  const totalUnits = units?.length ?? 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Hunoid Control</h1>
          <p className="text-slate-400">Autonomous humanoid robotics management</p>
        </div>
        <div className="flex items-center gap-2 px-4 py-2 bg-blue-500/20 border border-blue-500/50 rounded-lg">
          <Users className="w-4 h-4 text-blue-400" />
          <span className="text-blue-400">{activeUnits}/{totalUnits} Active</span>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Total Units', value: totalUnits, icon: Bot, color: 'from-blue-500 to-cyan-600' },
          { label: 'Active', value: activeUnits, icon: Activity, color: 'from-emerald-500 to-teal-600' },
          { label: 'On Mission', value: units?.filter((u) => u.status === 'mission').length ?? 0, icon: MapPin, color: 'from-violet-500 to-purple-600' },
          { label: 'Avg Battery', value: `${Math.round((units?.reduce((sum, u) => sum + u.battery, 0) ?? 0) / (totalUnits || 1))}%`, icon: Battery, color: 'from-amber-500 to-orange-600' },
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

      {/* Unit Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Unit List */}
        <div className="lg:col-span-1 bg-slate-800/50 border border-slate-700 rounded-xl p-4">
          <h2 className="text-lg font-semibold text-white mb-4">Units</h2>
          {isLoading ? (
            <div className="text-center py-8 text-slate-500">Loading units...</div>
          ) : (
            <div className="space-y-3">
              {units?.map((unit) => (
                <motion.button
                  key={unit.id}
                  onClick={() => setSelectedUnit(unit.id)}
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                  className={`w-full text-left p-4 rounded-lg border transition-all ${
                    selectedUnit === unit.id
                      ? 'bg-blue-500/10 border-blue-500/50'
                      : 'bg-slate-900/50 border-slate-700 hover:border-slate-600'
                  }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Bot className="w-4 h-4 text-slate-400" />
                      <span className="font-medium text-white">{unit.name}</span>
                    </div>
                    <div className={`w-2 h-2 rounded-full ${getStatusColor(unit.status)}`} />
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-slate-500 capitalize">{unit.status}</span>
                    <span className="text-slate-400">{unit.battery}%</span>
                  </div>
                </motion.button>
              ))}
            </div>
          )}
        </div>

        {/* Unit Details */}
        <div className="lg:col-span-2 bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          {selectedUnit ? (
            (() => {
              const unit = units?.find((u) => u.id === selectedUnit);
              if (!unit) return null;

              return (
                <div className="space-y-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <h2 className="text-xl font-bold text-white">{unit.name}</h2>
                      <div className="flex items-center gap-2 mt-1">
                        <div className={`w-2 h-2 rounded-full ${getStatusColor(unit.status)}`} />
                        <span className="text-slate-400 capitalize">{unit.status}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {unit.status === 'idle' ? (
                        <button
                          onClick={() => deployMutation.mutate(unit.id)}
                          className="flex items-center gap-2 px-4 py-2 bg-emerald-500/20 text-emerald-400 rounded-lg hover:bg-emerald-500/30 transition-colors"
                        >
                          <Play className="w-4 h-4" />
                          Deploy
                        </button>
                      ) : unit.status === 'active' || unit.status === 'mission' ? (
                        <button
                          onClick={() => recallMutation.mutate(unit.id)}
                          className="flex items-center gap-2 px-4 py-2 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30 transition-colors"
                        >
                          <RotateCcw className="w-4 h-4" />
                          Recall
                        </button>
                      ) : null}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="p-4 bg-slate-900/50 rounded-lg">
                      <p className="text-xs text-slate-500 mb-1">Battery Level</p>
                      <div className="flex items-center gap-2">
                        <Battery className="w-5 h-5 text-emerald-400" />
                        <span className="text-xl font-bold text-white">{unit.battery}%</span>
                      </div>
                    </div>
                    <div className="p-4 bg-slate-900/50 rounded-lg">
                      <p className="text-xs text-slate-500 mb-1">Position</p>
                      <div className="flex items-center gap-2">
                        <MapPin className="w-5 h-5 text-blue-400" />
                        <span className="text-sm text-white font-mono">
                          {unit.position.lat.toFixed(4)}, {unit.position.lng.toFixed(4)}
                        </span>
                      </div>
                    </div>
                  </div>

                  {unit.currentMission && (
                    <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-lg">
                      <p className="text-xs text-blue-400 mb-1">Current Mission</p>
                      <p className="text-white">{unit.currentMission}</p>
                    </div>
                  )}

                  <div className="text-xs text-slate-500">
                    Last updated: {new Date(unit.lastUpdate).toLocaleString()}
                  </div>
                </div>
              );
            })()
          ) : (
            <div className="flex items-center justify-center h-64 text-slate-500">
              <div className="text-center">
                <Bot className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Select a unit to view details</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
