import { useParams, Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useState } from 'react';
import { 
  ArrowLeft, 
  Share2, 
  Heart, 
  Eye, 
  MapPin, 
  MessageSquare,
  Users,
  Satellite,
  Send,
  MoreVertical,
  Loader2
} from 'lucide-react';
import VideoPlayer from '@/components/VideoPlayer';
import { cn, formatLatency, formatBitrate } from '@/lib/utils';
import { useStream, useStreams, useStreamUpdates, useWebRTCStream, useStreamChat } from '@/hooks/useStreams';
import { usePlayerStore } from '@/stores/hubStore';
import type { Stream } from '@/lib/api';

// Chat Component
function LiveChat({ streamId }: { streamId: string }) {
  const { enabled, messages, isConnected, isLoading, error, sendMessage } = useStreamChat(streamId);
  const [draft, setDraft] = useState('');

  const handleSend = async () => {
    if (!draft.trim()) return;
    try {
      await sendMessage(draft.trim());
      setDraft('');
    } catch {
      // ignore send errors to avoid blocking UI
    }
  };

  return (
    <div className="hub-card p-0 flex flex-col h-[400px]">
      <div className="flex items-center justify-between p-4 border-b border-hub-border">
        <div className="flex items-center gap-2">
          <MessageSquare className="w-5 h-5 text-hub-accent" />
          <h2 className="font-semibold text-white">Live Chat</h2>
        </div>
        <button className="p-1 hover:bg-hub-surface rounded" title="More options" aria-label="More options">
          <MoreVertical className="w-4 h-4 text-gray-500" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
        {!enabled && (
          <div className="flex items-center justify-center text-gray-500 text-sm px-6 text-center h-full">
            Live chat is not enabled for this environment.
          </div>
        )}
        {enabled && isLoading && (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="w-5 h-5 animate-spin text-hub-accent" />
          </div>
        )}
        {enabled && error && (
          <div className="text-sm text-red-400 text-center">{error}</div>
        )}
        {enabled && !isLoading && !error && messages.length === 0 && (
          <div className="text-sm text-gray-500 text-center">No messages yet.</div>
        )}
        {enabled && messages.map((msg) => (
          <div key={msg.id} className="flex items-start gap-2">
            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-hub-accent to-purple-500 flex items-center justify-center flex-shrink-0">
              <span className="text-xs text-white font-medium">
                {msg.username.charAt(0).toUpperCase()}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-0.5">
                <span className="text-sm font-medium text-gray-300">{msg.username}</span>
                <span className="text-xs text-gray-600">
                  {new Date(msg.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
              </div>
              <p className="text-sm text-gray-400 break-words">{msg.message}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="p-4 border-t border-hub-border">
        <div className="flex items-center gap-2">
          <input
            type="text"
            disabled={!enabled || !isConnected}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder={enabled ? "Send a message..." : "Chat is not enabled"}
            className="flex-1 h-10 px-4 rounded-xl bg-hub-surface border border-hub-border text-white placeholder-gray-600 focus:outline-none"
          />
          <button
            disabled={!enabled || !isConnected || !draft.trim()}
            onClick={handleSend}
            className={cn(
              'p-2 rounded-lg transition-all',
              enabled && isConnected && draft.trim()
                ? 'bg-hub-accent text-white hover:bg-hub-accent/80'
                : 'bg-hub-surface text-gray-600'
            )}
            title="Send message"
            aria-label="Send message"
          >
            <Send className="w-5 h-5" />
          </button>
        </div>
        {enabled && (
          <div className="mt-2 text-xs text-gray-500">
            {isConnected ? 'Connected' : 'Reconnecting...'}
          </div>
        )}
      </div>
    </div>
  );
}

// Related Streams Component
function RelatedStreams({ currentId, type }: { currentId: string; type?: Stream['type'] }) {
  const { data, isLoading, error } = useStreams(type ? { type } : undefined);
  const relatedStreams = (data?.streams ?? [])
    .filter((stream) => stream.id !== currentId)
    .slice(0, 4);

  return (
    <div className="hub-card p-6">
      <h2 className="font-semibold text-white mb-4">Related Streams</h2>
      {isLoading ? (
        <div className="flex items-center justify-center py-6">
          <Loader2 className="w-5 h-5 animate-spin text-hub-accent" />
        </div>
      ) : error ? (
        <div className="text-sm text-red-400">Failed to load related streams.</div>
      ) : relatedStreams.length > 0 ? (
        <div className="space-y-3">
          {relatedStreams.map((stream) => (
            <Link
              key={stream.id}
              to={`/stream/${stream.id}`}
              className="flex items-center gap-3 p-2 -mx-2 rounded-lg hover:bg-hub-surface transition-colors group"
            >
              <div className="w-16 h-10 bg-hub-darker rounded-lg flex items-center justify-center relative overflow-hidden">
                <Satellite className="w-4 h-4 text-gray-600" />
                {stream.status === 'live' && (
                  <div className="absolute top-1 left-1">
                    <span className="flex items-center gap-0.5 px-1 py-0.5 rounded bg-red-500 text-white text-[8px] font-bold">
                      <span className="w-1 h-1 rounded-full bg-white animate-pulse" />
                      LIVE
                    </span>
                  </div>
                )}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-white truncate group-hover:text-hub-accent transition-colors">
                  {stream.title}
                </p>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <span>{stream.source}</span>
                  <span>•</span>
                  <span className="flex items-center gap-1">
                    <Eye className="w-3 h-3" />
                    {stream.viewers.toLocaleString()}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>
      ) : (
        <div className="text-sm text-gray-500">No related streams available.</div>
      )}
    </div>
  );
}

export default function StreamView() {
  const { streamId } = useParams();
  const streamKey = streamId ?? '';
  const { data: stream, isLoading, error } = useStream(streamKey);
  const shouldConnectWebRTC = Boolean(streamId && !stream?.playbackUrl);
  const { mediaStream, isConnecting, error: streamError, reconnect } = useWebRTCStream(shouldConnectWebRTC ? streamId ?? null : null);
  const playerLatency = usePlayerStore((state) => state.latency);
  const [isLiked, setIsLiked] = useState(false);

  useStreamUpdates();

  const handleShare = async () => {
    if (!stream) return;
    try {
      await navigator.share({
        title: stream.title,
        text: `Watch ${stream.title} live on ASGARD Hubs`,
        url: window.location.href,
      });
    } catch {
      navigator.clipboard.writeText(window.location.href);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-hub-accent" />
      </div>
    );
  }

  if (error || !stream) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-red-400">Stream not available.</p>
      </div>
    );
  }

  const startedAt = new Date(stream.startedAt);
  const effectiveLatency = playerLatency || stream.latency;

  return (
    <div className="max-w-7xl mx-auto space-y-6">
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
          latency={effectiveLatency}
          mediaStream={mediaStream}
          playbackUrl={stream.playbackUrl}
          isConnecting={isConnecting}
          error={streamError?.message ?? null}
          onReconnect={shouldConnectWebRTC ? reconnect : undefined}
          resolution={stream.resolution}
          bitrate={stream.bitrate}
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
                <button 
                  onClick={() => setIsLiked(!isLiked)}
                  className={cn(
                    'p-2 rounded-lg transition-all',
                    isLiked 
                      ? 'bg-red-500/10 text-red-500' 
                      : 'hover:bg-hub-surface text-gray-400 hover:text-white'
                  )}
                  title={isLiked ? "Unlike" : "Like"}
                  aria-label={isLiked ? "Unlike" : "Like"}
                >
                  <Heart className={cn('w-5 h-5', isLiked && 'fill-current')} />
                </button>
                <button 
                  onClick={handleShare}
                  className="p-2 rounded-lg hover:bg-hub-surface transition-colors text-gray-400 hover:text-white"
                  title="Share stream"
                  aria-label="Share stream"
                >
                  <Share2 className="w-5 h-5" />
                </button>
              </div>
            </div>

            <p className="text-gray-400 leading-relaxed">
              {stream.description || 'No description available for this stream.'}
            </p>

            <div className="mt-4 grid sm:grid-cols-2 gap-4">
              <div className="p-4 rounded-xl bg-hub-surface/50">
                <div className="flex items-center gap-2 text-sm text-gray-500 mb-2">
                  <Users className="w-4 h-4" />
                  Stream Type
                </div>
                <p className="text-white font-medium">{stream.type}</p>
              </div>
              <div className="p-4 rounded-xl bg-hub-surface/50">
                <div className="flex items-center gap-2 text-sm text-gray-500 mb-2">
                  <Users className="w-4 h-4" />
                  Source Type
                </div>
                <p className="text-white font-medium">{stream.sourceType}</p>
              </div>
            </div>
          </motion.div>

          {/* Live Chat */}
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <LiveChat streamId={stream.id} />
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
                { label: 'Latency', value: formatLatency(effectiveLatency) },
                { label: 'Resolution', value: stream.resolution || 'N/A' },
                { label: 'Bitrate', value: stream.bitrate ? formatBitrate(stream.bitrate) : 'N/A' },
                { label: 'Source', value: stream.source },
                { label: 'Started', value: startedAt.toLocaleTimeString() },
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
          >
            <RelatedStreams currentId={stream.id} type={stream.type} />
          </motion.div>
        </div>
      </div>
    </div>
  );
}
