import { useParams, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { 
  ArrowLeft, 
  Share2, 
  Heart, 
  Eye, 
  MapPin, 
  Clock,
  MessageSquare,
  Users,
  Satellite
} from 'lucide-react';
import VideoPlayer from '@/components/VideoPlayer';
import { cn, formatLatency } from '@/lib/utils';

// Mock stream data - in production this would come from API
const getStreamById = (id: string) => ({
  id,
  title: 'Pacific Maritime Monitoring',
  source: 'Silenus-47',
  location: 'Pacific Ocean, 34°N 140°W',
  type: 'civilian' as const,
  status: 'live' as const,
  viewers: 12453,
  latency: 234,
  description: `Real-time monitoring of Pacific shipping lanes for search and rescue coordination. 
  This feed covers approximately 2.5 million square kilometers of ocean surface, tracking vessel 
  movements and weather patterns to support humanitarian operations.`,
  startedAt: new Date(Date.now() - 3600000 * 4), // 4 hours ago
  resolution: '1920x1080',
  bitrate: '4.5 Mbps',
});

export default function StreamView() {
  const { streamId } = useParams();
  const stream = getStreamById(streamId || 'unknown');

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Back Navigation */}
      <Link
        to="/"
        className="inline-flex items-center gap-2 text-gray-400 hover:text-white transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        Back to Hubs
      </Link>

      {/* Video Player */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <VideoPlayer
          streamId={stream.id}
          title={stream.title}
          isLive={stream.status === 'live'}
          latency={stream.latency}
        />
      </motion.div>

      {/* Stream Info */}
      <div className="grid lg:grid-cols-3 gap-6">
        {/* Main Info */}
        <div className="lg:col-span-2 space-y-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="hub-card p-6"
          >
            <div className="flex items-start justify-between mb-4">
              <div>
                <h1 className="text-xl font-bold text-white mb-2">
                  {stream.title}
                </h1>
                <div className="flex flex-wrap items-center gap-4 text-sm text-gray-400">
                  <span className="flex items-center gap-1">
                    <Satellite className="w-4 h-4" />
                    {stream.source}
                  </span>
                  <span className="flex items-center gap-1">
                    <MapPin className="w-4 h-4" />
                    {stream.location}
                  </span>
                  <span className="flex items-center gap-1">
                    <Eye className="w-4 h-4" />
                    {stream.viewers.toLocaleString()} watching
                  </span>
                </div>
              </div>
              
              <div className="flex items-center gap-2">
                <button className="p-2 rounded-lg hover:bg-hub-surface transition-colors">
                  <Heart className="w-5 h-5 text-gray-400 hover:text-red-500" />
                </button>
                <button className="p-2 rounded-lg hover:bg-hub-surface transition-colors">
                  <Share2 className="w-5 h-5 text-gray-400" />
                </button>
              </div>
            </div>

            <p className="text-gray-400 leading-relaxed">
              {stream.description}
            </p>
          </motion.div>

          {/* Comments / Chat */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="hub-card p-6"
          >
            <div className="flex items-center gap-2 mb-4">
              <MessageSquare className="w-5 h-5 text-gray-400" />
              <h2 className="font-semibold text-white">Live Chat</h2>
              <span className="text-xs text-gray-500 bg-hub-surface px-2 py-0.5 rounded">
                1,234 messages
              </span>
            </div>

            <div className="h-64 bg-hub-surface rounded-xl flex items-center justify-center">
              <p className="text-gray-500 text-sm">Chat functionality coming soon</p>
            </div>

            <div className="mt-4 flex gap-2">
              <input
                type="text"
                placeholder="Send a message..."
                className="flex-1 h-10 px-4 rounded-xl bg-hub-surface border border-hub-border text-white placeholder-gray-500 focus:outline-none focus:border-hub-accent"
              />
              <button className="h-10 px-4 rounded-xl bg-hub-accent text-white font-medium hover:bg-hub-accent/90 transition-colors">
                Send
              </button>
            </div>
          </motion.div>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Stream Stats */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 }}
            className="hub-card p-6"
          >
            <h2 className="font-semibold text-white mb-4">Stream Details</h2>
            <div className="space-y-3">
              {[
                { label: 'Status', value: stream.status.toUpperCase(), highlight: true },
                { label: 'Latency', value: formatLatency(stream.latency) },
                { label: 'Resolution', value: stream.resolution },
                { label: 'Bitrate', value: stream.bitrate },
                { label: 'Started', value: stream.startedAt.toLocaleTimeString() },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className={cn(
                    'text-sm font-medium',
                    item.highlight ? 'text-green-400' : 'text-white'
                  )}>
                    {item.value}
                  </span>
                </div>
              ))}
            </div>
          </motion.div>

          {/* Related Streams */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="hub-card p-6"
          >
            <h2 className="font-semibold text-white mb-4">Related Streams</h2>
            <div className="space-y-3">
              {[
                { title: 'Atlantic Patrol', viewers: 3421 },
                { title: 'Indian Ocean Monitor', viewers: 2156 },
                { title: 'Arctic Survey', viewers: 1823 },
              ].map((related, index) => (
                <Link
                  key={index}
                  to={`/stream/${related.title.toLowerCase().replace(/\s+/g, '-')}`}
                  className="flex items-center gap-3 p-2 -mx-2 rounded-lg hover:bg-hub-surface transition-colors"
                >
                  <div className="w-16 h-10 bg-hub-darker rounded flex items-center justify-center">
                    <Satellite className="w-4 h-4 text-gray-600" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-white truncate">{related.title}</p>
                    <p className="text-xs text-gray-500">{related.viewers} watching</p>
                  </div>
                </Link>
              ))}
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
}
