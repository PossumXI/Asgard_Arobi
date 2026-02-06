import { useState } from 'react';
import { motion } from 'framer-motion';
import { Globe, Grid, List, Search, MapPin, Loader2 } from 'lucide-react';
import StreamCard from '@/components/StreamCard';
import { cn } from '@/lib/utils';
import { useStreams, useStreamUpdates } from '@/hooks/useStreams';

const regions = ['All Regions', 'Asia Pacific', 'Americas', 'Europe', 'Africa', 'Arctic'];

export default function CivilianHub() {
  const [layout, setLayout] = useState<'grid' | 'list'>('grid');
  const [selectedRegion, setSelectedRegion] = useState('All Regions');
  const [searchQuery, setSearchQuery] = useState('');
  const { data, isLoading, error } = useStreams({ type: 'civilian' });

  useStreamUpdates();

  const streams = data?.streams ?? [];

  const filteredStreams = streams.filter((stream) => {
    const matchesSearch =
      stream.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      stream.location.toLowerCase().includes(searchQuery.toLowerCase());

    if (selectedRegion === 'All Regions') {
      return matchesSearch;
    }

    return matchesSearch && stream.location.toLowerCase().includes(selectedRegion.toLowerCase());
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
            aria-label="Select region"
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
            type="button"
            aria-label="Grid layout"
            title="Grid layout"
            onClick={() => setLayout('grid')}
            className={cn(
              'p-2 rounded-lg transition-colors',
              layout === 'grid' ? 'bg-hub-accent text-white' : 'text-gray-400 hover:text-white'
            )}
          >
            <Grid className="w-4 h-4" />
          </button>
          <button
            type="button"
            aria-label="List layout"
            title="List layout"
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
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center py-12">
            <Loader2 className="w-6 h-6 animate-spin text-hub-accent" />
          </div>
        ) : error ? (
          <div className="col-span-full text-center text-red-400">
            Failed to load civilian streams.
          </div>
        ) : filteredStreams.length > 0 ? (
          filteredStreams.map((stream, index) => (
            <motion.div
              key={stream.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
            >
              <StreamCard stream={stream} layout={layout} />
            </motion.div>
          ))
        ) : (
          <div className="col-span-full text-center text-gray-500">
            No civilian streams available.
          </div>
        )}
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
