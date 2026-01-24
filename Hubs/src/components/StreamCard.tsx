import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { 
  Play, 
  Eye, 
  Clock, 
  MapPin,
  Signal
} from 'lucide-react';
import { cn, formatLatency } from '@/lib/utils';
import type { Stream } from '@/lib/api';

interface StreamCardProps {
  stream: Stream;
  layout?: 'grid' | 'list';
}

const typeColors = {
  civilian: 'border-civilian/30 hover:border-civilian/50',
  military: 'border-military/30 hover:border-military/50',
  interstellar: 'border-interstellar/30 hover:border-interstellar/50',
};

const statusLabels: Record<Stream['status'], { label: string; class: string }> = {
  live: { label: 'LIVE', class: 'bg-green-500' },
  delayed: { label: 'DELAYED', class: 'bg-yellow-500' },
  offline: { label: 'OFFLINE', class: 'bg-red-500' },
  buffering: { label: 'BUFFERING', class: 'bg-orange-500' },
};

export default function StreamCard({ stream, layout = 'grid' }: StreamCardProps) {
  return (
    <Link to={`/stream/${stream.id}`}>
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        whileHover={{ scale: 1.02 }}
        className={cn(
          'hub-card group cursor-pointer',
          typeColors[stream.type],
          layout === 'list' && 'flex'
        )}
      >
        {/* Thumbnail */}
        <div className={cn(
          'relative overflow-hidden bg-hub-darker',
          layout === 'grid' ? 'aspect-video' : 'w-48 h-28 flex-shrink-0'
        )}>
          {stream.thumbnail ? (
            <img
              src={stream.thumbnail}
              alt={stream.title}
              className="w-full h-full object-cover"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center">
              <Signal className="w-12 h-12 text-gray-700" />
            </div>
          )}

          {/* Overlay */}
          <div className="stream-overlay" />

          {/* Status Badge */}
          <div className="absolute top-3 left-3">
            <span className={cn(
              'px-2 py-0.5 rounded text-xs font-bold text-white',
              statusLabels[stream.status].class
            )}>
              {statusLabels[stream.status].label}
            </span>
          </div>

          {/* Play Button */}
          <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
            <div className="w-14 h-14 rounded-full bg-hub-accent/90 flex items-center justify-center">
              <Play className="w-6 h-6 text-white ml-1" fill="white" />
            </div>
          </div>

          {/* Bottom Stats */}
          <div className="absolute bottom-2 left-2 right-2 flex items-center justify-between">
            <div className="flex items-center gap-3 text-xs text-white/80">
              <span className="flex items-center gap-1">
                <Eye className="w-3 h-3" />
                {stream.viewers.toLocaleString()}
              </span>
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {formatLatency(stream.latency)}
              </span>
            </div>
          </div>
        </div>

        {/* Info */}
        <div className={cn(
          'p-4',
          layout === 'list' && 'flex-1'
        )}>
          <h3 className="font-medium text-white mb-1 line-clamp-1 group-hover:text-hub-accent transition-colors">
            {stream.title}
          </h3>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <span>{stream.source}</span>
            <span>•</span>
            <span className="flex items-center gap-1">
              <MapPin className="w-3 h-3" />
              {stream.location}
            </span>
          </div>
          {stream.description && layout === 'list' && (
            <p className="text-sm text-gray-500 mt-2 line-clamp-2">
              {stream.description}
            </p>
          )}
        </div>
      </motion.div>
    </Link>
  );
}
