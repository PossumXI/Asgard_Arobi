/**
 * ASGARD Government Client - Valkyrie Control
 * Autonomous flight system control interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useQuery, useMutation } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Plane,
  Navigation,
  Gauge,
  Battery,
  Thermometer,
  AlertTriangle,
  Play,
  Square,
  Home,
} from 'lucide-react';
import { toast } from '@/components/ui/Toaster';

interface FlightState {
  armed: boolean;
  mode: string;
  position: { lat: number; lng: number; alt: number };
  velocity: { x: number; y: number; z: number };
  attitude: { roll: number; pitch: number; yaw: number };
  battery: { voltage: number; percent: number };
  health: 'healthy' | 'degraded' | 'critical';
}

export function ValkyrieControl() {
  const { data: flightState, isLoading } = useQuery<FlightState>({
    queryKey: ['valkyrie-state'],
    queryFn: async () => {
      const response = await fetch('/api/valkyrie/api/v1/state');
      return response.json();
    },
    refetchInterval: 1000,
  });

  const armMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch('/api/valkyrie/api/v1/arm', { method: 'POST' });
      if (!response.ok) throw new Error('Failed to arm');
    },
    onSuccess: () => toast({ type: 'success', title: 'Aircraft armed' }),
    onError: () => toast({ type: 'error', title: 'Failed to arm aircraft' }),
  });

  const disarmMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch('/api/valkyrie/api/v1/disarm', { method: 'POST' });
      if (!response.ok) throw new Error('Failed to disarm');
    },
    onSuccess: () => toast({ type: 'success', title: 'Aircraft disarmed' }),
    onError: () => toast({ type: 'error', title: 'Failed to disarm aircraft' }),
  });

  const rtbMutation = useMutation({
    mutationFn: async () => {
      const response = await fetch('/api/valkyrie/api/v1/emergency/rtb', { method: 'POST' });
      if (!response.ok) throw new Error('Failed to initiate RTB');
    },
    onSuccess: () => toast({ type: 'warning', title: 'Return to base initiated' }),
    onError: () => toast({ type: 'error', title: 'Failed to initiate RTB' }),
  });

  const getHealthColor = (health: FlightState['health']) => {
    return health === 'healthy' ? 'text-emerald-400' : health === 'degraded' ? 'text-amber-400' : 'text-red-400';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Valkyrie Control</h1>
          <p className="text-slate-400">Autonomous Flight System</p>
        </div>
        <div className="flex items-center gap-3">
          {flightState?.armed ? (
            <button
              onClick={() => disarmMutation.mutate()}
              className="flex items-center gap-2 px-4 py-2 bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors"
            >
              <Square className="w-4 h-4" />
              Disarm
            </button>
          ) : (
            <button
              onClick={() => armMutation.mutate()}
              className="flex items-center gap-2 px-4 py-2 bg-emerald-500/20 text-emerald-400 rounded-lg hover:bg-emerald-500/30 transition-colors"
            >
              <Play className="w-4 h-4" />
              Arm
            </button>
          )}
          <button
            onClick={() => rtbMutation.mutate()}
            className="flex items-center gap-2 px-4 py-2 bg-amber-500/20 text-amber-400 rounded-lg hover:bg-amber-500/30 transition-colors"
          >
            <Home className="w-4 h-4" />
            RTB
          </button>
        </div>
      </div>

      {isLoading ? (
        <div className="text-center py-20 text-slate-500">Loading flight data...</div>
      ) : (
        <>
          {/* Status Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
            >
              <div className="flex items-center gap-3 mb-2">
                <Plane className={`w-5 h-5 ${flightState?.armed ? 'text-emerald-400' : 'text-slate-400'}`} />
                <span className="text-sm text-slate-400">Status</span>
              </div>
              <p className="text-xl font-bold text-white">{flightState?.armed ? 'ARMED' : 'DISARMED'}</p>
              <p className="text-sm text-slate-500">{flightState?.mode}</p>
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.1 }}
              className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
            >
              <div className="flex items-center gap-3 mb-2">
                <Navigation className="w-5 h-5 text-blue-400" />
                <span className="text-sm text-slate-400">Altitude</span>
              </div>
              <p className="text-xl font-bold text-white">{flightState?.position.alt.toFixed(1)} m</p>
              <p className="text-sm text-slate-500">AGL</p>
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
              className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
            >
              <div className="flex items-center gap-3 mb-2">
                <Gauge className="w-5 h-5 text-amber-400" />
                <span className="text-sm text-slate-400">Speed</span>
              </div>
              <p className="text-xl font-bold text-white">
                {Math.sqrt(
                  (flightState?.velocity.x ?? 0) ** 2 +
                  (flightState?.velocity.y ?? 0) ** 2
                ).toFixed(1)} m/s
              </p>
              <p className="text-sm text-slate-500">Ground speed</p>
            </motion.div>

            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
              className="bg-slate-800/50 border border-slate-700 rounded-xl p-4"
            >
              <div className="flex items-center gap-3 mb-2">
                <Battery className="w-5 h-5 text-emerald-400" />
                <span className="text-sm text-slate-400">Battery</span>
              </div>
              <p className="text-xl font-bold text-white">{flightState?.battery.percent}%</p>
              <p className="text-sm text-slate-500">{flightState?.battery.voltage.toFixed(1)}V</p>
            </motion.div>
          </div>

          {/* Flight Data */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Position */}
            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white mb-4">Position</h2>
              <div className="grid grid-cols-3 gap-4">
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Latitude</p>
                  <p className="text-lg font-mono text-white">{flightState?.position.lat.toFixed(6)}</p>
                </div>
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Longitude</p>
                  <p className="text-lg font-mono text-white">{flightState?.position.lng.toFixed(6)}</p>
                </div>
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Altitude</p>
                  <p className="text-lg font-mono text-white">{flightState?.position.alt.toFixed(1)}m</p>
                </div>
              </div>
            </div>

            {/* Attitude */}
            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
              <h2 className="text-lg font-semibold text-white mb-4">Attitude</h2>
              <div className="grid grid-cols-3 gap-4">
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Roll</p>
                  <p className="text-lg font-mono text-white">
                    {((flightState?.attitude.roll ?? 0) * 180 / Math.PI).toFixed(1)}°
                  </p>
                </div>
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Pitch</p>
                  <p className="text-lg font-mono text-white">
                    {((flightState?.attitude.pitch ?? 0) * 180 / Math.PI).toFixed(1)}°
                  </p>
                </div>
                <div className="text-center p-4 bg-slate-900/50 rounded-lg">
                  <p className="text-xs text-slate-500 mb-1">Yaw</p>
                  <p className="text-lg font-mono text-white">
                    {((flightState?.attitude.yaw ?? 0) * 180 / Math.PI).toFixed(1)}°
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Health Status */}
          <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
            <h2 className="text-lg font-semibold text-white mb-4">System Health</h2>
            <div className="flex items-center gap-4">
              <div className={`flex items-center gap-2 ${getHealthColor(flightState?.health ?? 'healthy')}`}>
                {flightState?.health === 'healthy' ? (
                  <div className="w-3 h-3 bg-emerald-500 rounded-full" />
                ) : (
                  <AlertTriangle className="w-5 h-5" />
                )}
                <span className="font-medium uppercase">{flightState?.health}</span>
              </div>
              <span className="text-slate-500">|</span>
              <span className="text-slate-400">All systems operational</span>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
