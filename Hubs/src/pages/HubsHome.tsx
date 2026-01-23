import { motion } from 'framer-motion';
import { Globe, Shield, Rocket, TrendingUp, Activity } from 'lucide-react';
import StreamCard, { Stream } from '@/components/StreamCard';
import { cn } from '@/lib/utils';

// Sample stream data - in production this would come from the API
const featuredStreams: Stream[] = [
  {
    id: 'silenus-47',
    title: 'Silenus-47 Pacific Monitoring',
    source: 'Silenus Constellation',
    location: 'Pacific Ocean',
    type: 'civilian',
    status: 'live',
    viewers: 12453,
    latency: 234,
    description: 'Real-time monitoring of Pacific shipping lanes and weather patterns',
  },
  {
    id: 'hunoid-102',
    title: 'Hunoid-102 Aid Delivery',
    source: 'Hunoid Ground Unit',
    location: 'Southeast Asia',
    type: 'civilian',
    status: 'live',
    viewers: 8721,
    latency: 156,
    description: 'Medical supply delivery to remote village',
  },
  {
    id: 'mars-relay',
    title: 'Mars Base Alpha Construction',
    source: 'Interstellar Hub',
    location: 'Mars',
    type: 'interstellar',
    status: 'delayed',
    viewers: 45892,
    latency: 720000,
    description: 'Reconstructed reality feed from Mars operations',
  },
];

const recentStreams: Stream[] = [
  {
    id: 'atlantic-patrol',
    title: 'Atlantic Maritime Patrol',
    source: 'Silenus-23',
    location: 'Atlantic Ocean',
    type: 'civilian',
    status: 'live',
    viewers: 3421,
    latency: 189,
  },
  {
    id: 'disaster-relief',
    title: 'Earthquake Response Team',
    source: 'Hunoid Units',
    location: 'Indonesia',
    type: 'civilian',
    status: 'live',
    viewers: 15632,
    latency: 201,
  },
  {
    id: 'forest-monitor',
    title: 'Amazon Deforestation Monitor',
    source: 'Silenus-56',
    location: 'Brazil',
    type: 'civilian',
    status: 'live',
    viewers: 6743,
    latency: 267,
  },
  {
    id: 'arctic-survey',
    title: 'Arctic Ice Survey',
    source: 'Silenus-12',
    location: 'Arctic',
    type: 'civilian',
    status: 'live',
    viewers: 2156,
    latency: 312,
  },
];

const stats = [
  { label: 'Active Streams', value: '247', icon: Activity, trend: '+12' },
  { label: 'Global Viewers', value: '1.2M', icon: Globe, trend: '+5.3%' },
  { label: 'Satellites Online', value: '152', icon: TrendingUp, trend: '+3' },
];

const hubCategories = [
  { 
    id: 'civilian', 
    label: 'Civilian Hub', 
    icon: Globe, 
    color: 'bg-civilian/10 text-civilian border-civilian/20',
    streams: 156,
    description: 'Public humanitarian operations'
  },
  { 
    id: 'military', 
    label: 'Military Hub', 
    icon: Shield, 
    color: 'bg-military/10 text-military border-military/20',
    streams: 67,
    description: 'Authorized tactical feeds'
  },
  { 
    id: 'interstellar', 
    label: 'Interstellar Hub', 
    icon: Rocket, 
    color: 'bg-interstellar/10 text-interstellar border-interstellar/20',
    streams: 24,
    description: 'Beyond Earth operations'
  },
];

export default function HubsHome() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white mb-2">ASGARD Viewing Hubs</h1>
        <p className="text-gray-400">
          Real-time access to global operations and humanitarian missions
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        {stats.map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="hub-card p-5"
          >
            <div className="flex items-start justify-between mb-3">
              <div className="p-2 rounded-lg bg-hub-accent/10">
                <stat.icon className="w-5 h-5 text-hub-accent" />
              </div>
              <span className="text-xs font-medium text-green-400 bg-green-400/10 px-2 py-0.5 rounded">
                {stat.trend}
              </span>
            </div>
            <div className="text-2xl font-bold text-white mb-1">{stat.value}</div>
            <div className="text-sm text-gray-500">{stat.label}</div>
          </motion.div>
        ))}
      </div>

      {/* Hub Categories */}
      <div>
        <h2 className="text-lg font-semibold text-white mb-4">Browse by Hub</h2>
        <div className="grid md:grid-cols-3 gap-4">
          {hubCategories.map((hub, index) => (
            <motion.a
              key={hub.id}
              href={`/${hub.id}`}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
              className={cn(
                'hub-card p-5 flex items-center gap-4 border',
                hub.color
              )}
            >
              <div className={cn('p-3 rounded-xl', hub.color)}>
                <hub.icon className="w-6 h-6" />
              </div>
              <div className="flex-1">
                <h3 className="font-medium text-white">{hub.label}</h3>
                <p className="text-sm text-gray-500">{hub.description}</p>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold text-white">{hub.streams}</div>
                <div className="text-xs text-gray-500">streams</div>
              </div>
            </motion.a>
          ))}
        </div>
      </div>

      {/* Featured Streams */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Featured Streams</h2>
          <a href="#" className="text-sm text-hub-accent hover:underline">
            View all
          </a>
        </div>
        <div className="grid md:grid-cols-3 gap-4">
          {featuredStreams.map((stream) => (
            <StreamCard key={stream.id} stream={stream} />
          ))}
        </div>
      </div>

      {/* Recent Streams */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Recently Active</h2>
          <a href="#" className="text-sm text-hub-accent hover:underline">
            View all
          </a>
        </div>
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
          {recentStreams.map((stream) => (
            <StreamCard key={stream.id} stream={stream} />
          ))}
        </div>
      </div>
    </div>
  );
}
