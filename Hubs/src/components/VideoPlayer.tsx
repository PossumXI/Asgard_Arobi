import { useState, useRef, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Play, 
  Pause, 
  Volume2, 
  VolumeX, 
  Maximize2, 
  Minimize2,
  Settings,
  Wifi,
  WifiOff,
  RefreshCw,
  Loader2,
  Camera
} from 'lucide-react';
import { cn, formatDuration, formatLatency, formatBitrate } from '@/lib/utils';
import { useVideoElement } from '@/hooks/useStreams';
import { usePlayerStore } from '@/stores/hubStore';

interface VideoPlayerProps {
  streamId: string;
  title: string;
  isLive?: boolean;
  latency?: number;
  mediaStream?: MediaStream | null;
  playbackUrl?: string;
  isConnecting?: boolean;
  error?: string | null;
  onReconnect?: () => void;
  resolution?: string;
  bitrate?: number;
}

// Connection status indicator
function ConnectionStatus({ status }: { status: 'connected' | 'connecting' | 'disconnected' }) {
  const statusConfig = {
    connected: { icon: Wifi, color: 'text-green-400', label: 'Connected' },
    connecting: { icon: Loader2, color: 'text-yellow-400', label: 'Connecting...' },
    disconnected: { icon: WifiOff, color: 'text-red-400', label: 'Disconnected' },
  };

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div className={cn('flex items-center gap-1.5 text-xs', config.color)}>
      <Icon className={cn('w-3 h-3', status === 'connecting' && 'animate-spin')} />
      <span>{config.label}</span>
    </div>
  );
}

export default function VideoPlayer({
  streamId,
  title,
  isLive = true,
  latency = 0,
  mediaStream = null,
  playbackUrl,
  isConnecting = false,
  error = null,
  onReconnect,
  resolution,
  bitrate,
}: VideoPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [showControls, setShowControls] = useState(true);
  const [currentTime, setCurrentTime] = useState(0);
  const [showQualityMenu, setShowQualityMenu] = useState(false);

  const {
    isPlaying,
    togglePlay,
    isMuted,
    toggleMute,
    volume,
    setVolume,
    isFullscreen,
    setFullscreen,
    showStats,
    setShowStats,
    quality,
    setQuality,
    latency: storeLatency,
  } = usePlayerStore();

  const videoRef = useVideoElement(mediaStream, playbackUrl);

  const hasPlayback = Boolean(playbackUrl);
  const connectionStatus: 'connected' | 'connecting' | 'disconnected' = error
    ? 'disconnected'
    : isConnecting && !hasPlayback
      ? 'connecting'
      : hasPlayback || mediaStream
        ? 'connected'
        : 'disconnected';

  const stats = {
    bitrate: bitrate ?? 0,
    resolution: resolution ?? 'N/A',
    fps: 'N/A',
    codec: 'N/A',
    protocol: playbackUrl ? 'HLS' : 'WebRTC',
  };

  const effectiveLatency = latency || storeLatency;

  // Auto-hide controls
  useEffect(() => {
    let timeout: ReturnType<typeof setTimeout>;
    
    if (isPlaying && showControls) {
      timeout = setTimeout(() => setShowControls(false), 3000);
    }

    return () => clearTimeout(timeout);
  }, [showControls, isPlaying]);

  // Time counter
  useEffect(() => {
    const interval = setInterval(() => {
      if (isPlaying) {
        setCurrentTime((t) => t + 1);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isPlaying]);

  // Fullscreen toggle
  const toggleFullscreen = useCallback(() => {
    if (!containerRef.current) return;

    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen();
      setFullscreen(true);
    } else {
      document.exitFullscreen();
      setFullscreen(false);
    }
  }, [setFullscreen]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeydown = (e: KeyboardEvent) => {
      switch (e.key.toLowerCase()) {
        case ' ':
        case 'k':
          e.preventDefault();
          togglePlay();
          break;
        case 'm':
          toggleMute();
          break;
        case 'f':
          toggleFullscreen();
          break;
        case 's':
          setShowStats(!showStats);
          break;
      }
    };

    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  }, [toggleFullscreen, togglePlay, toggleMute, setShowStats, showStats]);

  return (
    <div
      ref={containerRef}
      className="relative bg-black rounded-2xl overflow-hidden group"
      onMouseMove={() => setShowControls(true)}
      onMouseLeave={() => isPlaying && setShowControls(false)}
    >
      {/* Video Area */}
      <div className="aspect-video bg-hub-darker relative">
        {connectionStatus === 'connecting' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center">
              <Loader2 className="w-12 h-12 text-hub-accent mx-auto mb-4 animate-spin" />
              <p className="text-gray-400">Establishing connection...</p>
              <p className="text-sm text-gray-600 mt-1">Connecting to {streamId}</p>
            </div>
          </div>
        )}

        {connectionStatus === 'disconnected' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="text-center">
              <WifiOff className="w-12 h-12 text-red-500 mx-auto mb-4" />
              <p className="text-gray-400">Stream unavailable</p>
              {onReconnect && (
                <button
                  onClick={onReconnect}
                  className="mt-4 px-4 py-2 rounded-lg bg-hub-accent text-white hover:bg-hub-accent/80 transition-colors flex items-center gap-2 mx-auto"
                >
                  <RefreshCw className="w-4 h-4" />
                  Reconnect
                </button>
              )}
            </div>
          </div>
        )}

        {connectionStatus === 'connected' && (
          <video
            ref={videoRef}
            className="absolute inset-0 w-full h-full object-cover"
            playsInline
            autoPlay
            muted={isMuted}
          />
        )}

        {/* Paused overlay */}
        <AnimatePresence>
          {!isPlaying && connectionStatus === 'connected' && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 flex items-center justify-center bg-black/50"
            >
              <div className="text-center">
                <Camera className="w-16 h-16 text-white/50 mx-auto mb-4" />
                <p className="text-white/70">Stream Paused</p>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Live Indicator */}
      {isLive && connectionStatus === 'connected' && (
        <div className="absolute top-4 left-4 flex items-center gap-2">
          <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-red-500 text-white text-xs font-bold shadow-lg">
            <span className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
            LIVE
          </span>
          <span className="px-2.5 py-1 rounded-lg bg-black/60 backdrop-blur-sm text-white text-xs">
            {formatLatency(effectiveLatency)} latency
          </span>
        </div>
      )}

      {/* Connection Status */}
      <div className="absolute top-4 right-4">
        <ConnectionStatus status={connectionStatus} />
      </div>

      {/* Stats Overlay */}
      <AnimatePresence>
        {showStats && (
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            className="absolute top-14 right-4 p-4 rounded-xl bg-black/80 backdrop-blur-sm text-xs font-mono text-gray-300 space-y-1.5 min-w-[180px]"
          >
            <div className="flex items-center justify-between">
              <span className="text-gray-500">Bitrate</span>
              <span>{stats.bitrate ? formatBitrate(stats.bitrate) : 'N/A'}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-500">Resolution</span>
              <span>{stats.resolution}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-500">FPS</span>
              <span>{stats.fps}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-500">Codec</span>
              <span>{stats.codec}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-500">Protocol</span>
              <span>{stats.protocol}</span>
            </div>
            <div className="border-t border-gray-700 mt-2 pt-2">
              <div className="flex items-center justify-between">
                <span className="text-gray-500">Packets Rx</span>
                <span>N/A</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-gray-500">Lost</span>
                <span>N/A</span>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Controls Overlay */}
      <motion.div
        initial={false}
        animate={{ opacity: showControls ? 1 : 0 }}
        transition={{ duration: 0.2 }}
        className="absolute inset-0 bg-gradient-to-t from-black/90 via-transparent to-black/30 pointer-events-none"
      >
        <div className="pointer-events-auto">
          {/* Top Bar */}
          <div className="absolute top-0 left-0 right-0 p-4 flex items-center justify-between">
            <h2 className="text-white font-medium truncate pr-4">{title}</h2>
          </div>

          {/* Center Play/Pause */}
          <div className="absolute inset-0 flex items-center justify-center">
            <button
              onClick={togglePlay}
              className="w-16 h-16 rounded-full bg-white/10 backdrop-blur-sm flex items-center justify-center hover:bg-white/20 transition-colors"
            >
              {isPlaying ? (
                <Pause className="w-8 h-8 text-white" />
              ) : (
                <Play className="w-8 h-8 text-white ml-1" fill="white" />
              )}
            </button>
          </div>

          {/* Bottom Controls */}
          <div className="absolute bottom-0 left-0 right-0 p-4">
            {/* Progress Bar */}
            <div className="mb-3">
              <div className="h-1 bg-white/20 rounded-full overflow-hidden">
                <motion.div 
                  className="h-full bg-hub-accent rounded-full"
                  style={{ width: '0%' }}
                />
              </div>
            </div>

            {/* Control Buttons */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <button
                  onClick={togglePlay}
                  className="p-2 rounded-lg hover:bg-white/10 transition-colors"
                >
                  {isPlaying ? (
                    <Pause className="w-5 h-5 text-white" />
                  ) : (
                    <Play className="w-5 h-5 text-white" />
                  )}
                </button>

                {/* Volume Control */}
                <div className="flex items-center gap-2 group/volume">
                  <button
                    onClick={toggleMute}
                    className="p-2 rounded-lg hover:bg-white/10 transition-colors"
                  >
                    {isMuted || volume === 0 ? (
                      <VolumeX className="w-5 h-5 text-white" />
                    ) : (
                      <Volume2 className="w-5 h-5 text-white" />
                    )}
                  </button>
                  <div className="w-0 overflow-hidden group-hover/volume:w-24 transition-all duration-200">
                    <input
                      type="range"
                      min="0"
                      max="100"
                      step="1"
                      value={isMuted ? 0 : volume}
                      onChange={(e) => {
                        setVolume(parseFloat(e.target.value));
                      }}
                      className="w-full accent-hub-accent"
                      title="Volume control"
                      aria-label="Volume"
                    />
                  </div>
                </div>

                <span className="text-sm text-white/80">
                  {formatDuration(currentTime)}
                </span>
              </div>

              <div className="flex items-center gap-2">
                {/* Quality Selector */}
                <div className="relative">
                  <button
                    onClick={() => setShowQualityMenu(!showQualityMenu)}
                    className={cn(
                      'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                      showQualityMenu ? 'bg-white/20 text-white' : 'hover:bg-white/10 text-white/80'
                    )}
                  >
                    {quality === 'auto' ? 'Auto' : quality}
                  </button>
                  <AnimatePresence>
                    {showQualityMenu && (
                      <motion.div
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: 10 }}
                        className="absolute bottom-full right-0 mb-2 p-1 rounded-lg bg-black/90 backdrop-blur-sm min-w-[100px]"
                      >
                        {(['auto', '1080p', '720p', '480p'] as const).map((q) => (
                          <button
                            key={q}
                            onClick={() => {
                              setQuality(q);
                              setShowQualityMenu(false);
                            }}
                            className={cn(
                              'w-full px-3 py-1.5 text-left text-xs rounded transition-colors',
                              quality === q 
                                ? 'bg-hub-accent text-white' 
                                : 'text-white/80 hover:bg-white/10'
                            )}
                          >
                            {q === 'auto' ? 'Auto' : q}
                          </button>
                        ))}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                <button
                  onClick={() => setShowStats(!showStats)}
                  className={cn(
                    'p-2 rounded-lg hover:bg-white/10 transition-colors',
                    showStats && 'bg-white/10'
                  )}
                  title="Toggle stats (S)"
                >
                  <Settings className="w-5 h-5 text-white" />
                </button>

                <button
                  onClick={toggleFullscreen}
                  className="p-2 rounded-lg hover:bg-white/10 transition-colors"
                  title="Toggle fullscreen (F)"
                >
                  {isFullscreen ? (
                    <Minimize2 className="w-5 h-5 text-white" />
                  ) : (
                    <Maximize2 className="w-5 h-5 text-white" />
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
