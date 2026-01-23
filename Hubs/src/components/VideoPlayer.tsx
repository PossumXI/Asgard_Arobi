import { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { 
  Play, 
  Pause, 
  Volume2, 
  VolumeX, 
  Maximize2, 
  Minimize2,
  Settings,
  Signal,
  AlertCircle
} from 'lucide-react';
import { cn, formatDuration, formatLatency, formatBitrate } from '@/lib/utils';

interface VideoPlayerProps {
  streamId: string;
  title: string;
  isLive?: boolean;
  latency?: number;
}

export default function VideoPlayer({ streamId, title, isLive = true, latency = 0 }: VideoPlayerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(true);
  const [isMuted, setIsMuted] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [showControls, setShowControls] = useState(true);
  const [currentTime, setCurrentTime] = useState(0);
  const [showStats, setShowStats] = useState(false);

  // Simulated stats
  const stats = {
    bitrate: 4500000,
    resolution: '1920x1080',
    fps: 30,
    codec: 'H.264',
    protocol: 'WebRTC',
  };

  useEffect(() => {
    let timeout: ReturnType<typeof setTimeout>;
    
    if (isPlaying) {
      timeout = setTimeout(() => setShowControls(false), 3000);
    }

    return () => clearTimeout(timeout);
  }, [showControls, isPlaying]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (isPlaying) {
        setCurrentTime((t) => t + 1);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isPlaying]);

  const toggleFullscreen = () => {
    if (!containerRef.current) return;

    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  };

  return (
    <div
      ref={containerRef}
      className="relative bg-black rounded-2xl overflow-hidden group"
      onMouseMove={() => setShowControls(true)}
      onMouseLeave={() => isPlaying && setShowControls(false)}
    >
      {/* Video Placeholder / Stream Area */}
      <div className="aspect-video bg-hub-darker flex items-center justify-center">
        <div className="text-center">
          <Signal className="w-16 h-16 text-gray-700 mx-auto mb-4 animate-pulse" />
          <p className="text-gray-500">Stream: {streamId}</p>
          <p className="text-sm text-gray-600 mt-1">WebRTC Connection Active</p>
        </div>
      </div>

      {/* Live Indicator */}
      {isLive && (
        <div className="absolute top-4 left-4 flex items-center gap-2">
          <span className="flex items-center gap-1.5 px-2 py-1 rounded bg-red-500 text-white text-xs font-bold">
            <span className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
            LIVE
          </span>
          <span className="px-2 py-1 rounded bg-black/50 text-white text-xs">
            {formatLatency(latency)} latency
          </span>
        </div>
      )}

      {/* Stats Overlay */}
      {showStats && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="absolute top-4 right-4 p-3 rounded-lg bg-black/80 text-xs font-mono text-gray-300 space-y-1"
        >
          <div>Bitrate: {formatBitrate(stats.bitrate)}</div>
          <div>Resolution: {stats.resolution}</div>
          <div>FPS: {stats.fps}</div>
          <div>Codec: {stats.codec}</div>
          <div>Protocol: {stats.protocol}</div>
        </motion.div>
      )}

      {/* Controls Overlay */}
      <motion.div
        initial={false}
        animate={{ opacity: showControls ? 1 : 0 }}
        className="absolute inset-0 bg-gradient-to-t from-black/90 via-transparent to-black/30"
      >
        {/* Top Bar */}
        <div className="absolute top-0 left-0 right-0 p-4 flex items-center justify-between">
          <h2 className="text-white font-medium truncate pr-4">{title}</h2>
        </div>

        {/* Center Play/Pause */}
        <div className="absolute inset-0 flex items-center justify-center">
          <button
            onClick={() => setIsPlaying(!isPlaying)}
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
              <div className="h-full bg-hub-accent w-0 rounded-full" />
            </div>
          </div>

          {/* Control Buttons */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <button
                onClick={() => setIsPlaying(!isPlaying)}
                className="p-2 rounded-lg hover:bg-white/10 transition-colors"
              >
                {isPlaying ? (
                  <Pause className="w-5 h-5 text-white" />
                ) : (
                  <Play className="w-5 h-5 text-white" />
                )}
              </button>

              <button
                onClick={() => setIsMuted(!isMuted)}
                className="p-2 rounded-lg hover:bg-white/10 transition-colors"
              >
                {isMuted ? (
                  <VolumeX className="w-5 h-5 text-white" />
                ) : (
                  <Volume2 className="w-5 h-5 text-white" />
                )}
              </button>

              <span className="text-sm text-white/80">
                {formatDuration(currentTime)}
              </span>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => setShowStats(!showStats)}
                className={cn(
                  'p-2 rounded-lg hover:bg-white/10 transition-colors',
                  showStats && 'bg-white/10'
                )}
              >
                <Settings className="w-5 h-5 text-white" />
              </button>

              <button
                onClick={toggleFullscreen}
                className="p-2 rounded-lg hover:bg-white/10 transition-colors"
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
      </motion.div>
    </div>
  );
}
