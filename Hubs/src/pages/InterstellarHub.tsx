import { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Rocket, Clock, Radio, Orbit, Info, Satellite, Globe, Play, Pause, Loader2 } from 'lucide-react';
import StreamCard from '@/components/StreamCard';
import { cn, formatLatency } from '@/lib/utils';
import { useStreams, useStreamUpdates } from '@/hooks/useStreams';

// 3D Solar System Visualization Component
function SolarSystemVisualization() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [isPlaying, setIsPlaying] = useState(true);
  const [selectedPlanet, setSelectedPlanet] = useState<string | null>(null);
  const animationRef = useRef<number>();
  const timeRef = useRef(0);

  // Planet data
  const planets = [
    { name: 'Mercury', color: '#b5b5b5', distance: 40, size: 4, speed: 0.04, info: '57.9M km from Sun' },
    { name: 'Venus', color: '#e6c87a', distance: 60, size: 6, speed: 0.03, info: '108.2M km from Sun' },
    { name: 'Earth', color: '#4a90d9', distance: 85, size: 7, speed: 0.025, info: '149.6M km from Sun', hasAsgard: true },
    { name: 'Mars', color: '#d9534f', distance: 115, size: 5, speed: 0.02, info: '227.9M km from Sun', hasBase: true },
    { name: 'Jupiter', color: '#e0a050', distance: 160, size: 14, speed: 0.008, info: '778.5M km from Sun' },
    { name: 'Saturn', color: '#f0d090', distance: 200, size: 12, speed: 0.005, info: '1.4B km from Sun', hasRings: true },
  ];

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Set canvas size
    const resizeCanvas = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = rect.width * window.devicePixelRatio;
      canvas.height = rect.height * window.devicePixelRatio;
      ctx.scale(window.devicePixelRatio, window.devicePixelRatio);
    };
    resizeCanvas();
    window.addEventListener('resize', resizeCanvas);

    // Draw function
    const draw = () => {
      const w = canvas.offsetWidth;
      const h = canvas.offsetHeight;
      const cx = w / 2;
      const cy = h / 2;

      // Clear and draw background
      ctx.fillStyle = '#0a0a12';
      ctx.fillRect(0, 0, w, h);

      // Draw stars
      ctx.fillStyle = 'rgba(255, 255, 255, 0.5)';
      for (let i = 0; i < 200; i++) {
        const x = (Math.sin(i * 123.456) * 0.5 + 0.5) * w;
        const y = (Math.cos(i * 789.012) * 0.5 + 0.5) * h;
        const size = (Math.sin(i * 345.678) * 0.5 + 0.5) * 1.5 + 0.5;
        const twinkle = Math.sin(timeRef.current * 0.05 + i) * 0.5 + 0.5;
        ctx.globalAlpha = 0.3 + twinkle * 0.7;
        ctx.beginPath();
        ctx.arc(x, y, size, 0, Math.PI * 2);
        ctx.fill();
      }
      ctx.globalAlpha = 1;

      // Draw Sun
      const sunGradient = ctx.createRadialGradient(cx, cy, 0, cx, cy, 25);
      sunGradient.addColorStop(0, '#fff5d0');
      sunGradient.addColorStop(0.5, '#ffc107');
      sunGradient.addColorStop(1, '#ff8f00');
      ctx.fillStyle = sunGradient;
      ctx.beginPath();
      ctx.arc(cx, cy, 20, 0, Math.PI * 2);
      ctx.fill();

      // Sun glow
      const glowGradient = ctx.createRadialGradient(cx, cy, 15, cx, cy, 50);
      glowGradient.addColorStop(0, 'rgba(255, 193, 7, 0.3)');
      glowGradient.addColorStop(1, 'rgba(255, 193, 7, 0)');
      ctx.fillStyle = glowGradient;
      ctx.beginPath();
      ctx.arc(cx, cy, 50, 0, Math.PI * 2);
      ctx.fill();

      // Draw orbits and planets
      planets.forEach((planet) => {
        const scale = Math.min(w, h) / 500;
        const distance = planet.distance * scale;
        
        // Draw orbit path
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(cx, cy, distance, 0, Math.PI * 2);
        ctx.stroke();

        // Calculate planet position
        const angle = timeRef.current * planet.speed;
        const x = cx + Math.cos(angle) * distance;
        const y = cy + Math.sin(angle) * distance * 0.4; // Elliptical orbit effect

        // Draw planet
        const planetGradient = ctx.createRadialGradient(x - 2, y - 2, 0, x, y, planet.size * scale);
        planetGradient.addColorStop(0, planet.color);
        planetGradient.addColorStop(1, adjustColor(planet.color, -40));
        ctx.fillStyle = planetGradient;
        ctx.beginPath();
        ctx.arc(x, y, planet.size * scale, 0, Math.PI * 2);
        ctx.fill();

        // Draw Saturn's rings
        if (planet.hasRings) {
          ctx.strokeStyle = 'rgba(240, 208, 144, 0.4)';
          ctx.lineWidth = 3 * scale;
          ctx.beginPath();
          ctx.ellipse(x, y, planet.size * 1.8 * scale, planet.size * 0.5 * scale, 0.3, 0, Math.PI * 2);
          ctx.stroke();
        }

        // Draw ASGARD indicator for Earth
        if (planet.hasAsgard) {
          ctx.strokeStyle = '#00ff88';
          ctx.lineWidth = 1;
          ctx.beginPath();
          ctx.arc(x, y, planet.size * scale + 4, 0, Math.PI * 2);
          ctx.stroke();
          
          // Satellite orbit
          const satAngle = timeRef.current * 0.15;
          const satX = x + Math.cos(satAngle) * (planet.size * scale + 8);
          const satY = y + Math.sin(satAngle) * (planet.size * scale + 8) * 0.6;
          ctx.fillStyle = '#00ff88';
          ctx.beginPath();
          ctx.arc(satX, satY, 2, 0, Math.PI * 2);
          ctx.fill();
        }

        // Draw Mars base indicator
        if (planet.hasBase) {
          ctx.fillStyle = '#00ff88';
          ctx.font = '10px sans-serif';
          ctx.fillText('● Mars Base', x + planet.size * scale + 5, y + 3);
        }

        // Hover effect (simplified - actual hover would need mouse tracking)
        if (selectedPlanet === planet.name) {
          ctx.strokeStyle = 'rgba(0, 255, 136, 0.5)';
          ctx.lineWidth = 2;
          ctx.beginPath();
          ctx.arc(x, y, planet.size * scale + 10, 0, Math.PI * 2);
          ctx.stroke();
        }
      });

      // Draw communication lines
      const earthAngle = timeRef.current * 0.025;
      const marsAngle = timeRef.current * 0.02;
      const earthScale = Math.min(w, h) / 500;
      
      const earthX = cx + Math.cos(earthAngle) * (85 * earthScale);
      const earthY = cy + Math.sin(earthAngle) * (85 * earthScale) * 0.4;
      const marsX = cx + Math.cos(marsAngle) * (115 * earthScale);
      const marsY = cy + Math.sin(marsAngle) * (115 * earthScale) * 0.4;

      // Dashed line between Earth and Mars
      ctx.strokeStyle = 'rgba(0, 255, 136, 0.3)';
      ctx.lineWidth = 1;
      ctx.setLineDash([5, 5]);
      ctx.beginPath();
      ctx.moveTo(earthX, earthY);
      ctx.lineTo(marsX, marsY);
      ctx.stroke();
      ctx.setLineDash([]);

      // Traveling signal dots
      const signalProgress = (timeRef.current * 0.01) % 1;
      const signalX = earthX + (marsX - earthX) * signalProgress;
      const signalY = earthY + (marsY - earthY) * signalProgress;
      ctx.fillStyle = '#00ff88';
      ctx.beginPath();
      ctx.arc(signalX, signalY, 3, 0, Math.PI * 2);
      ctx.fill();

      // Legend
      ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
      ctx.font = '11px sans-serif';
      ctx.fillText('● ASGARD Satellite', 10, h - 30);
      ctx.fillText('● Active Base', 10, h - 15);
      ctx.fillStyle = '#00ff88';
      ctx.beginPath();
      ctx.arc(7, h - 33, 3, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.arc(7, h - 18, 3, 0, Math.PI * 2);
      ctx.fill();

      if (isPlaying) {
        timeRef.current += 1;
      }
      animationRef.current = requestAnimationFrame(draw);
    };

    draw();

    return () => {
      window.removeEventListener('resize', resizeCanvas);
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [isPlaying, selectedPlanet]);

  return (
    <div className="relative">
      <canvas
        ref={canvasRef}
        className="w-full aspect-video rounded-xl"
        style={{ background: '#0a0a12' }}
      />
      
      {/* Controls overlay */}
      <div className="absolute bottom-4 left-4 flex items-center gap-2">
        <button
          onClick={() => setIsPlaying(!isPlaying)}
          className="p-2 rounded-lg bg-black/50 backdrop-blur-sm text-white hover:bg-black/70 transition-colors"
        >
          {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
        </button>
        <span className="text-xs text-white/70 bg-black/50 backdrop-blur-sm px-2 py-1 rounded-lg">
          {isPlaying ? 'Visualization' : 'Paused'}
        </span>
      </div>

      {/* Planet selector */}
      <div className="absolute top-4 right-4 flex flex-wrap gap-1">
        {planets.map((planet) => (
          <button
            key={planet.name}
            onClick={() => setSelectedPlanet(selectedPlanet === planet.name ? null : planet.name)}
            className={cn(
              'px-2 py-1 text-xs rounded-lg transition-colors',
              selectedPlanet === planet.name
                ? 'bg-interstellar text-white'
                : 'bg-black/50 backdrop-blur-sm text-white/70 hover:bg-black/70'
            )}
          >
            {planet.name}
          </button>
        ))}
      </div>

      {/* Selected planet info */}
      {selectedPlanet && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          className="absolute bottom-4 right-4 p-3 rounded-lg bg-black/80 backdrop-blur-sm text-white text-sm"
        >
          <div className="font-medium mb-1">{selectedPlanet}</div>
          <div className="text-xs text-gray-400">
            {planets.find(p => p.name === selectedPlanet)?.info}
          </div>
        </motion.div>
      )}
    </div>
  );
}

// Helper function to adjust color brightness
function adjustColor(color: string, amount: number): string {
  const hex = color.replace('#', '');
  const r = Math.max(0, Math.min(255, parseInt(hex.substr(0, 2), 16) + amount));
  const g = Math.max(0, Math.min(255, parseInt(hex.substr(2, 2), 16) + amount));
  const b = Math.max(0, Math.min(255, parseInt(hex.substr(4, 2), 16) + amount));
  return `rgb(${r}, ${g}, ${b})`;
}

export default function InterstellarHub() {
  const [showInfo, setShowInfo] = useState(true);
  const { data, isLoading, error } = useStreams({ type: 'interstellar' });
  useStreamUpdates();

  const streams = data?.streams ?? [];
  const liveCount = streams.filter((stream) => stream.status === 'live').length;
  const delayedCount = streams.filter((stream) => stream.status === 'delayed').length;
  const offlineCount = streams.filter((stream) => stream.status === 'offline').length;
  const maxLatency = streams.length ? Math.max(...streams.map((stream) => stream.latency)) : 0;

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
          { label: 'Live Feeds', value: liveCount.toString(), status: 'active', icon: Rocket },
          { label: 'Delayed Feeds', value: delayedCount.toString(), status: 'active', icon: Globe },
          { label: 'Offline Feeds', value: offlineCount.toString(), status: 'active', icon: Satellite },
          { label: 'Signal Quality', value: 'N/A', status: 'unknown', icon: Radio },
        ].map((item, index) => (
          <motion.div
            key={item.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="hub-card p-4"
          >
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-interstellar/10">
                <item.icon className="w-5 h-5 text-interstellar" />
              </div>
              <div>
                <div className="text-2xl font-bold text-white">{item.value}</div>
                <div className="text-sm text-gray-500">{item.label}</div>
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {/* Stream Grid */}
      <div>
        <h2 className="text-lg font-semibold text-white mb-4">Active Feeds</h2>
        <div className="grid md:grid-cols-2 gap-4">
          {isLoading ? (
            <div className="col-span-full flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-hub-accent" />
            </div>
          ) : error ? (
            <div className="col-span-full text-center text-red-400">
              Failed to load interstellar streams.
            </div>
          ) : streams.length > 0 ? (
            streams.map((stream, index) => (
              <motion.div
                key={stream.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.1 }}
              >
                <StreamCard stream={stream} />
              </motion.div>
            ))
          ) : (
            <div className="col-span-full text-center text-gray-500">
              No interstellar streams available.
            </div>
          )}
        </div>
      </div>

      {/* 3D Solar System Visualization */}
      <div className="hub-card p-6">
        <h2 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <Orbit className="w-5 h-5 text-interstellar" />
          Solar System Overview
        </h2>
        <SolarSystemVisualization />
        <div className="mt-4 grid sm:grid-cols-3 gap-4">
          <div className="p-3 rounded-lg bg-hub-surface">
            <div className="text-sm text-gray-500 mb-1">Max Signal Delay</div>
            <div className="text-white font-medium">
              {maxLatency ? formatLatency(maxLatency) : 'N/A'}
            </div>
          </div>
          <div className="p-3 rounded-lg bg-hub-surface">
            <div className="text-sm text-gray-500 mb-1">Active Feeds</div>
            <div className="text-white font-medium">{streams.length}</div>
          </div>
          <div className="p-3 rounded-lg bg-hub-surface">
            <div className="text-sm text-gray-500 mb-1">Bandwidth</div>
            <div className="text-white font-medium">N/A</div>
          </div>
        </div>
      </div>
    </div>
  );
}
