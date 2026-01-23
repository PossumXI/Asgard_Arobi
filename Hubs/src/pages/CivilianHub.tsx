import { useState } from 'react';
import { motion } from 'framer-motion';
import { Globe, Filter, Grid, List, Search, MapPin } from 'lucide-react';
import StreamCard, { Stream } from '@/components/StreamCard';
import { cn } from '@/lib/utils';

const civilianStreams: Stream[] = [
  {
    id: 'pacific-monitor-1',
    title: 'Pacific Shipping Lane Monitor',
    source: 'Silenus-47',
    location: 'Pacific Ocean',
    type: 'civilian',
    status: 'live',
    viewers: 12453,
    latency: 234,
    description: 'Real-time monitoring of major shipping routes',
  },
  {
    id: 'hunoid-aid-1',
    title: 'Medical Supply Delivery',
    source: 'Hunoid-102',
    location: 'Vietnam',
    type: 'civilian',
    status: 'live',
    viewers: 8721,
    latency: 156,
    description: 'Emergency medical supplies to rural clinic',
  },
  {
    id: 'amazon-watch',
    title: 'Amazon Forest Monitoring',
    source: 'Silenus-56',
    location: 'Brazil',
    type: 'civilian',
    status: 'live',
    viewers: 6743,
    latency: 267,
    description: 'Deforestation and fire detection',
  },
  {
    id: 'atlantic-patrol',
    title: 'Atlantic Maritime Safety',
    source: 'Silenus-23',
    location: 'Atlantic Ocean',
    type: 'civilian',
    status: 'live',
    viewers: 3421,
    latency: 189,
    description: 'Search and rescue coordination',
  },
  {
    id: 'earthquake-response',
    title: 'Earthquake Response Team',
    source: 'Hunoid Squadron',
    location: 'Indonesia',
    type: 'civilian',
    status: 'live',
    viewers: 15632,
    latency: 201,
    description: 'Search and rescue operations',
  },
  {
    id: 'arctic-survey',
    title: 'Arctic Climate Survey',
    source: 'Silenus-12',
    location: 'Arctic Circle',
    type: 'civilian',
    status: 'live',
    viewers: 2156,
    latency: 312,
    description: 'Ice coverage and wildlife monitoring',
  },
  {
    id: 'flood-relief',
    title: 'Flood Relief Operations',
    source: 'Hunoid-78',
    location: 'Bangladesh',
    type: 'civilian',
    status: 'live',
    viewers: 9234,
    latency: 178,
    description: 'Evacuation and supply distribution',
  },
  {
    id: 'wildlife-protection',
    title: 'Wildlife Migration Tracking',
    source: 'Silenus-34',
    location: 'Africa',
    type: 'civilian',
    status: 'live',
    viewers: 4521,
    latency: 289,
    description: 'Anti-poaching surveillance',
  },
];

const regions = ['All Regions', 'Asia Pacific', 'Americas', 'Europe', 'Africa', 'Arctic'];

export default function CivilianHub() {
  const [layout, setLayout] = useState<'grid' | 'list'>('grid');
  const [selectedRegion, setSelectedRegion] = useState('All Regions');
  const [searchQuery, setSearchQuery] = useState('');

  const filteredStreams = civilianStreams.filter((stream) => {
    const matchesSearch = stream.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         stream.location.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 rounded-xl bg-civilian/10">
              <Globe className="w-6 h-6 text-civilian" />
            </div>
            <h1 className="text-2xl font-bold text-white">Civilian Hub</h1>
          </div>
          <p className="text-gray-400">
            Public access to humanitarian operations and global monitoring
          </p>
        </div>
        <div className="flex items-center gap-2 bg-civilian/10 px-3 py-1.5 rounded-full">
          <span className="w-2 h-2 rounded-full bg-civilian animate-pulse" />
          <span className="text-sm text-civilian font-medium">
            {filteredStreams.length} streams live
          </span>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
          <input
            type="text"
            placeholder="Search streams..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-xl bg-hub-surface border border-hub-border text-white placeholder-gray-500 focus:outline-none focus:border-hub-accent"
          />
        </div>

        {/* Region Filter */}
        <div className="flex items-center gap-2">
          <MapPin className="w-4 h-4 text-gray-500" />
          <select
            value={selectedRegion}
            onChange={(e) => setSelectedRegion(e.target.value)}
            className="h-10 px-4 rounded-xl bg-hub-surface border border-hub-border text-white focus:outline-none focus:border-hub-accent appearance-none cursor-pointer"
          >
            {regions.map((region) => (
              <option key={region} value={region}>{region}</option>
            ))}
          </select>
        </div>

        {/* Layout Toggle */}
        <div className="flex items-center gap-1 p-1 rounded-xl bg-hub-surface border border-hub-border">
          <button
            onClick={() => setLayout('grid')}
            className={cn(
              'p-2 rounded-lg transition-colors',
              layout === 'grid' ? 'bg-hub-accent text-white' : 'text-gray-400 hover:text-white'
            )}
          >
            <Grid className="w-4 h-4" />
          </button>
          <button
            onClick={() => setLayout('list')}
            className={cn(
              'p-2 rounded-lg transition-colors',
              layout === 'list' ? 'bg-hub-accent text-white' : 'text-gray-400 hover:text-white'
            )}
          >
            <List className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Stream Grid */}
      <div className={cn(
        layout === 'grid' 
          ? 'grid md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4'
          : 'space-y-3'
      )}>
        {filteredStreams.map((stream, index) => (
          <motion.div
            key={stream.id}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.05 }}
          >
            <StreamCard stream={stream} layout={layout} />
          </motion.div>
        ))}
      </div>

      {filteredStreams.length === 0 && (
        <div className="text-center py-12">
          <Globe className="w-12 h-12 text-gray-700 mx-auto mb-4" />
          <p className="text-gray-500">No streams found matching your criteria</p>
        </div>
      )}
    </div>
  );
}
