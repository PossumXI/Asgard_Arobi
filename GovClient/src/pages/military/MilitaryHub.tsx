/**
 * ASGARD Government Client - Military Hub
 * Military operations and tactical interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { motion } from 'framer-motion';
import {
  Shield,
  Radar,
  Target,
  Crosshair,
  AlertTriangle,
} from 'lucide-react';

export function MilitaryHub() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Military Operations</h1>
          <p className="text-slate-400">Tactical command and defense systems</p>
        </div>
        <div className="flex items-center gap-2 px-3 py-1.5 bg-red-500/20 border border-red-500/50 rounded-lg">
          <AlertTriangle className="w-4 h-4 text-red-400" />
          <span className="text-sm text-red-400">DEFCON 5</span>
        </div>
      </div>

      {/* Tactical Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {[
          { label: 'Active Threats', value: 0, icon: Target, color: 'text-red-400' },
          { label: 'Radar Coverage', value: '98%', icon: Radar, color: 'text-emerald-400' },
          { label: 'Defense Status', value: 'READY', icon: Shield, color: 'text-amber-400' },
        ].map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="bg-slate-800/50 border border-slate-700 rounded-xl p-6"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className={`text-3xl font-bold ${stat.color}`}>{stat.value}</p>
                <p className="text-sm text-slate-400 mt-1">{stat.label}</p>
              </div>
              <stat.icon className={`w-8 h-8 ${stat.color} opacity-50`} />
            </div>
          </motion.div>
        ))}
      </div>

      {/* Tactical Map Placeholder */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6 min-h-[500px]">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Tactical Overview</h2>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 bg-emerald-500 rounded-full status-online" />
            <span className="text-sm text-slate-400">Live Feed</span>
          </div>
        </div>
        <div className="flex items-center justify-center h-96 text-slate-500 border border-dashed border-slate-700 rounded-lg">
          <div className="text-center">
            <Crosshair className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>Tactical map integration</p>
            <p className="text-sm">Real-time threat tracking and defense coordination</p>
          </div>
        </div>
      </div>
    </div>
  );
}
