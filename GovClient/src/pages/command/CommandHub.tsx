/**
 * ASGARD Government Client - Command Hub
 * Central command and control interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Radio,
  Satellite,
  Bot,
  Plane,
  MapPin,
  Play,
  Pause,
  Square,
  AlertTriangle,
  CheckCircle,
} from 'lucide-react';
import { toast } from '@/components/ui/Toaster';

interface Mission {
  id: string;
  name: string;
  type: 'reconnaissance' | 'security' | 'humanitarian' | 'defense';
  status: 'planning' | 'active' | 'paused' | 'completed' | 'aborted';
  priority: 'low' | 'medium' | 'high' | 'critical';
  assets: string[];
  location: { lat: number; lng: number };
  startTime?: string;
  eta?: string;
}

const fetchActiveMissions = async (): Promise<Mission[]> => {
  const response = await fetch('/api/missions?status=active,planning');
  if (!response.ok) throw new Error('Failed to fetch missions');
  return response.json();
};

export function CommandHub() {
  const [selectedMission, setSelectedMission] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data: missions, isLoading } = useQuery<Mission[]>({
    queryKey: ['active-missions'],
    queryFn: fetchActiveMissions,
    refetchInterval: 5000,
  });

  const updateMissionStatus = useMutation({
    mutationFn: async ({ id, status }: { id: string; status: string }) => {
      const response = await fetch(`/api/missions/${id}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      });
      if (!response.ok) throw new Error('Failed to update mission');
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-missions'] });
      toast({ type: 'success', title: 'Mission status updated' });
    },
    onError: () => {
      toast({ type: 'error', title: 'Failed to update mission status' });
    },
  });

  const getStatusColor = (status: Mission['status']) => {
    const colors = {
      planning: 'bg-blue-500/20 text-blue-400 border-blue-500/50',
      active: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/50',
      paused: 'bg-amber-500/20 text-amber-400 border-amber-500/50',
      completed: 'bg-slate-500/20 text-slate-400 border-slate-500/50',
      aborted: 'bg-red-500/20 text-red-400 border-red-500/50',
    };
    return colors[status];
  };

  const getPriorityColor = (priority: Mission['priority']) => {
    const colors = {
      low: 'text-slate-400',
      medium: 'text-blue-400',
      high: 'text-amber-400',
      critical: 'text-red-400',
    };
    return colors[priority];
  };

  const getTypeIcon = (type: Mission['type']) => {
    const icons = {
      reconnaissance: Satellite,
      security: AlertTriangle,
      humanitarian: Bot,
      defense: Plane,
    };
    return icons[type];
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Command Hub</h1>
          <p className="text-slate-400">Mission command and control center</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/20 border border-emerald-500/50 rounded-lg">
            <Radio className="w-4 h-4 text-emerald-400" />
            <span className="text-sm text-emerald-400">COMMS ACTIVE</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Mission List */}
        <div className="lg:col-span-1 bg-slate-800/50 border border-slate-700 rounded-xl p-4">
          <h2 className="text-lg font-semibold text-white mb-4">Active Missions</h2>

          {isLoading ? (
            <div className="text-center py-8 text-slate-500">Loading missions...</div>
          ) : (
            <div className="space-y-3">
              {missions?.map((mission) => {
                const TypeIcon = getTypeIcon(mission.type);
                return (
                  <motion.button
                    key={mission.id}
                    onClick={() => setSelectedMission(mission.id)}
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    className={`w-full text-left p-4 rounded-lg border transition-all ${
                      selectedMission === mission.id
                        ? 'bg-amber-500/10 border-amber-500/50'
                        : 'bg-slate-900/50 border-slate-700 hover:border-slate-600'
                    }`}
                  >
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <TypeIcon className="w-4 h-4 text-slate-400" />
                        <span className="font-medium text-white">{mission.name}</span>
                      </div>
                      <span className={`text-xs font-medium uppercase ${getPriorityColor(mission.priority)}`}>
                        {mission.priority}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className={`px-2 py-0.5 text-xs rounded border ${getStatusColor(mission.status)}`}>
                        {mission.status}
                      </span>
                      <span className="text-xs text-slate-500">
                        {mission.assets.length} assets
                      </span>
                    </div>
                  </motion.button>
                );
              })}
            </div>
          )}
        </div>

        {/* Mission Details */}
        <div className="lg:col-span-2 bg-slate-800/50 border border-slate-700 rounded-xl p-6">
          {selectedMission ? (
            (() => {
              const mission = missions?.find((m) => m.id === selectedMission);
              if (!mission) return null;
              const TypeIcon = getTypeIcon(mission.type);

              return (
                <div className="space-y-6">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-3 mb-2">
                        <TypeIcon className="w-6 h-6 text-amber-500" />
                        <h2 className="text-xl font-bold text-white">{mission.name}</h2>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className={`px-2 py-0.5 text-xs rounded border ${getStatusColor(mission.status)}`}>
                          {mission.status}
                        </span>
                        <span className={`text-sm ${getPriorityColor(mission.priority)}`}>
                          {mission.priority.toUpperCase()} PRIORITY
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {mission.status === 'active' ? (
                        <button
                          onClick={() => updateMissionStatus.mutate({ id: mission.id, status: 'paused' })}
                          className="p-2 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30 transition-colors"
                        >
                          <Pause className="w-5 h-5" />
                        </button>
                      ) : mission.status === 'paused' ? (
                        <button
                          onClick={() => updateMissionStatus.mutate({ id: mission.id, status: 'active' })}
                          className="p-2 bg-emerald-500/20 text-emerald-400 rounded-lg hover:bg-emerald-500/30 transition-colors"
                        >
                          <Play className="w-5 h-5" />
                        </button>
                      ) : null}
                      <button
                        onClick={() => updateMissionStatus.mutate({ id: mission.id, status: 'aborted' })}
                        className="p-2 bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors"
                      >
                        <Square className="w-5 h-5" />
                      </button>
                    </div>
                  </div>

                  {/* Location */}
                  <div className="flex items-center gap-2 text-slate-400">
                    <MapPin className="w-4 h-4" />
                    <span>{mission.location.lat.toFixed(4)}, {mission.location.lng.toFixed(4)}</span>
                  </div>

                  {/* Assets */}
                  <div>
                    <h3 className="text-sm font-medium text-slate-300 mb-3">Assigned Assets</h3>
                    <div className="grid grid-cols-2 gap-3">
                      {mission.assets.map((asset) => (
                        <div
                          key={asset}
                          className="flex items-center gap-3 p-3 bg-slate-900/50 rounded-lg"
                        >
                          <CheckCircle className="w-4 h-4 text-emerald-400" />
                          <span className="text-white">{asset}</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Timeline */}
                  {(mission.startTime || mission.eta) && (
                    <div className="grid grid-cols-2 gap-4">
                      {mission.startTime && (
                        <div>
                          <p className="text-xs text-slate-500 mb-1">Start Time</p>
                          <p className="text-white">{new Date(mission.startTime).toLocaleString()}</p>
                        </div>
                      )}
                      {mission.eta && (
                        <div>
                          <p className="text-xs text-slate-500 mb-1">ETA</p>
                          <p className="text-white">{new Date(mission.eta).toLocaleString()}</p>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })()
          ) : (
            <div className="flex items-center justify-center h-64 text-slate-500">
              Select a mission to view details
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
