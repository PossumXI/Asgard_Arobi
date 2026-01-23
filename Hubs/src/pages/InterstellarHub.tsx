import { useState, Suspense } from 'react';
import { motion } from 'framer-motion';
import { Rocket, Clock, Radio, Orbit, Info } from 'lucide-react';
import StreamCard, { Stream } from '@/components/StreamCard';
import { cn } from '@/lib/utils';

const interstellarStreams: Stream[] = [
  {
    id: 'mars-alpha',
    title: 'Mars Base Alpha - Habitat Construction',
    source: 'Mars Hunoid Squadron',
    location: 'Jezero Crater, Mars',
    type: 'interstellar',
    status: 'delayed',
    viewers: 45892,
    latency: 720000, // 12 minutes
    description: 'Primary habitat dome construction progress',
  },
  {
    id: 'mars-rover',
    title: 'Geological Survey Unit',
    source: 'Mars Rover H-7',
    location: 'Syrtis Major, Mars',
    type: 'interstellar',
    status: 'delayed',
    viewers: 23156,
    latency: 780000,
    description: 'Rock sample collection and analysis',
  },
  {
    id: 'lunar-gateway',
    title: 'Lunar Gateway Station',
    source: 'Gateway Command',
    location: 'Lunar Orbit',
    type: 'interstellar',
    status: 'delayed',
    viewers: 18432,
    latency: 2500,
    description: 'Orbital operations and docking activities',
  },
  {
    id: 'deep-space',
    title: 'Deep Space Network Relay',
    source: 'DSN-42',
    location: 'L2 Lagrange Point',
    type: 'interstellar',
    status: 'live',
    viewers: 8721,
    latency: 4000,
    description: 'Communications relay status',
  },
];

export default function InterstellarHub() {
  const [showInfo, setShowInfo] = useState(true);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 rounded-xl bg-interstellar/10">
              <Rocket className="w-6 h-6 text-interstellar" />
            </div>
            <h1 className="text-2xl font-bold text-white">Interstellar Hub</h1>
          </div>
          <p className="text-gray-400">
            Beyond Earth operations and deep space missions
          </p>
        </div>
        <div className="flex items-center gap-2 bg-interstellar/10 px-3 py-1.5 rounded-full">
          <Clock className="w-4 h-4 text-interstellar" />
          <span className="text-sm text-interstellar font-medium">
            Time-Delayed Feeds
          </span>
        </div>
      </div>

      {/* Info Banner */}
      {showInfo && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="hub-card p-4 border-interstellar/20"
        >
          <div className="flex items-start gap-4">
            <div className="p-2 rounded-lg bg-interstellar/10">
              <Info className="w-5 h-5 text-interstellar" />
            </div>
            <div className="flex-1">
              <h3 className="font-medium text-white mb-1">
                About Interstellar Feeds
              </h3>
              <p className="text-sm text-gray-400 mb-3">
                Due to the vast distances involved, interstellar feeds experience significant 
                light-speed delays. Mars feeds have approximately 4-24 minute delays depending 
                on orbital positions. What you see is "Reconstructed Reality" - a 3D 
                visualization based on telemetry and journal data sent by our units.
              </p>
              <div className="flex items-center gap-6 text-xs text-gray-500">
                <span className="flex items-center gap-1">
                  <Radio className="w-3 h-3" />
                  Mars: 4-24 min delay
                </span>
                <span className="flex items-center gap-1">
                  <Orbit className="w-3 h-3" />
                  Moon: 1.3 sec delay
                </span>
              </div>
            </div>
            <button
              onClick={() => setShowInfo(false)}
              className="text-gray-500 hover:text-white transition-colors"
            >
              ×
            </button>
          </div>
        </motion.div>
      )}

      {/* Mission Status */}
      <div className="grid md:grid-cols-4 gap-4">
        {[
          { label: 'Mars Operations', value: '2', status: 'active' },
          { label: 'Lunar Operations', value: '1', status: 'active' },
          { label: 'Deep Space Relays', value: '4', status: 'active' },
          { label: 'Signal Quality', value: '94%', status: 'good' },
        ].map((item, index) => (
          <motion.div
            key={item.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="hub-card p-4"
          >
            <div className="text-2xl font-bold text-white mb-1">{item.value}</div>
            <div className="text-sm text-gray-500">{item.label}</div>
          </motion.div>
        ))}
      </div>

      {/* Stream Grid */}
      <div>
        <h2 className="text-lg font-semibold text-white mb-4">Active Feeds</h2>
        <div className="grid md:grid-cols-2 gap-4">
          {interstellarStreams.map((stream, index) => (
            <motion.div
              key={stream.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <StreamCard stream={stream} />
            </motion.div>
          ))}
        </div>
      </div>

      {/* 3D Visualization Placeholder */}
      <div className="hub-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4">
          Solar System Overview
        </h2>
        <div className="aspect-video bg-hub-darker rounded-xl flex items-center justify-center">
          <div className="text-center">
            <Orbit className="w-16 h-16 text-gray-700 mx-auto mb-4 animate-spin-slow" />
            <p className="text-gray-500">3D Solar System Visualization</p>
            <p className="text-sm text-gray-600 mt-1">Three.js rendering coming soon</p>
          </div>
        </div>
      </div>
    </div>
  );
}
