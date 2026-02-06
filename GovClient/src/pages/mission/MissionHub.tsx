/**
 * ASGARD Government Client - Mission Hub
 * Mission planning and monitoring interface
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { motion } from 'framer-motion';
import {
  Target,
  Plus,
  Map,
  List,
  Clock,
  Users,
  Crosshair,
} from 'lucide-react';

type ViewMode = 'map' | 'list' | 'timeline';

export function MissionHub() {
  const [viewMode, setViewMode] = useState<ViewMode>('list');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Mission Hub</h1>
          <p className="text-slate-400">Plan, deploy, and monitor missions</p>
        </div>
        <div className="flex items-center gap-4">
          {/* View Toggle */}
          <div className="flex bg-slate-800 rounded-lg p-1">
            <button
              onClick={() => setViewMode('list')}
              className={`p-2 rounded-md transition-colors ${
                viewMode === 'list' ? 'bg-amber-500 text-white' : 'text-slate-400 hover:text-white'
              }`}
            >
              <List className="w-4 h-4" />
            </button>
            <button
              onClick={() => setViewMode('map')}
              className={`p-2 rounded-md transition-colors ${
                viewMode === 'map' ? 'bg-amber-500 text-white' : 'text-slate-400 hover:text-white'
              }`}
            >
              <Map className="w-4 h-4" />
            </button>
            <button
              onClick={() => setViewMode('timeline')}
              className={`p-2 rounded-md transition-colors ${
                viewMode === 'timeline' ? 'bg-amber-500 text-white' : 'text-slate-400 hover:text-white'
              }`}
            >
              <Clock className="w-4 h-4" />
            </button>
          </div>

          <button className="flex items-center gap-2 px-4 py-2 bg-amber-500 text-white rounded-lg hover:bg-amber-600 transition-colors">
            <Plus className="w-4 h-4" />
            New Mission
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Active Missions', value: 5, icon: Target, color: 'from-emerald-500 to-teal-600' },
          { label: 'Pending Approval', value: 3, icon: Clock, color: 'from-amber-500 to-orange-600' },
          { label: 'Assets Deployed', value: 12, icon: Users, color: 'from-blue-500 to-cyan-600' },
          { label: 'Targets Tracked', value: 28, icon: Crosshair, color: 'from-violet-500 to-purple-600' },
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

      {/* Content Area */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6 min-h-[500px]">
        {viewMode === 'list' && (
          <div className="text-center py-20 text-slate-500">
            <Target className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>Mission list view</p>
            <p className="text-sm">Select or create a mission to begin</p>
          </div>
        )}
        {viewMode === 'map' && (
          <div className="text-center py-20 text-slate-500">
            <Map className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>Map view</p>
            <p className="text-sm">Geographic mission overview</p>
          </div>
        )}
        {viewMode === 'timeline' && (
          <div className="text-center py-20 text-slate-500">
            <Clock className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p>Timeline view</p>
            <p className="text-sm">Mission timeline and scheduling</p>
          </div>
        )}
      </div>
    </div>
  );
}
