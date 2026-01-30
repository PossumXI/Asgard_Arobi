import { useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, Lock, AlertTriangle, Key, Crosshair, Target, Loader2 } from 'lucide-react';
import StreamCard from '@/components/StreamCard';
import { useStreams, useStreamUpdates } from '@/hooks/useStreams';
import { hubsApi } from '@/lib/api';

export default function MilitaryHub() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [accessCode, setAccessCode] = useState('');
  const [accessError, setAccessError] = useState<string | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const { data, isLoading, error } = useStreams({ type: 'military' });
  useStreamUpdates();

  const streams = data?.streams ?? [];
  const liveCount = streams.filter((stream) => stream.status === 'live').length;
  const delayedCount = streams.filter((stream) => stream.status === 'delayed').length;
  const offlineCount = streams.filter((stream) => stream.status === 'offline').length;

  if (!isAuthenticated) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="max-w-md w-full"
        >
          <div className="hub-card p-8 text-center">
            <div className="w-16 h-16 rounded-2xl bg-military/10 flex items-center justify-center mx-auto mb-6">
              <Shield className="w-8 h-8 text-military" />
            </div>
            
            <h1 className="text-2xl font-bold text-white mb-2">Military Hub</h1>
            <p className="text-gray-400 mb-6">
              Access to tactical feeds requires security clearance verification
            </p>

            <div className="p-4 rounded-xl bg-military/5 border border-military/20 mb-6">
              <div className="flex items-start gap-3">
                <AlertTriangle className="w-5 h-5 text-military flex-shrink-0 mt-0.5" />
                <div className="text-left">
                  <p className="text-sm font-medium text-white mb-1">
                    Restricted Access
                  </p>
                  <p className="text-xs text-gray-400">
                    This hub contains classified operational data. Unauthorized access 
                    attempts are logged and may be prosecuted.
                  </p>
                </div>
              </div>
            </div>

            <div className="space-y-3">
              <div className="relative">
                <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
                <input
                  type="password"
                  placeholder="Enter access code"
                  value={accessCode}
                  onChange={(e) => setAccessCode(e.target.value)}
                  className="w-full h-12 pl-11 pr-4 rounded-xl bg-hub-surface border border-hub-border text-white placeholder-gray-500 focus:outline-none focus:border-military"
                />
              </div>

              <button
                onClick={async () => {
                  setIsVerifying(true);
                  setAccessError(null);
                  try {
                    const result = await hubsApi.validateAccessCode(accessCode, 'hubs');
                    const clearance = result.clearanceLevel || 'public';
                    const clearanceRank: Record<string, number> = {
                      public: 0,
                      civilian: 1,
                      military: 2,
                      interstellar: 3,
                      government: 4,
                      admin: 5,
                    };
                    if (result.valid && clearanceRank[clearance] >= 2) {
                      setIsAuthenticated(true);
                    } else {
                      setAccessError('Access code not authorized for military hub.');
                    }
                  } catch (err) {
                    setAccessError('Failed to verify access code.');
                  } finally {
                    setIsVerifying(false);
                  }
                }}
                disabled={isVerifying || accessCode.trim() === ''}
                className="w-full h-12 rounded-xl bg-military text-white font-medium hover:bg-military/90 transition-colors flex items-center justify-center gap-2 disabled:opacity-60"
              >
                {isVerifying ? <Loader2 className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
                {isVerifying ? 'Verifying...' : 'Verify Clearance'}
              </button>
            </div>

            {accessError && (
              <p className="text-xs text-red-400 mt-3">{accessError}</p>
            )}

            <p className="text-xs text-gray-500 mt-6">
              Contact your commanding officer for access credentials
            </p>
          </div>
        </motion.div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 rounded-xl bg-military/10">
              <Shield className="w-6 h-6 text-military" />
            </div>
            <h1 className="text-2xl font-bold text-white">Military Hub</h1>
          </div>
          <p className="text-gray-400">
            Tactical operations and threat monitoring feeds
          </p>
        </div>
        <div className="flex items-center gap-2 bg-military/10 px-3 py-1.5 rounded-full">
          <Lock className="w-4 h-4 text-military" />
          <span className="text-sm text-military font-medium">
            Classified Access
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="hub-card p-6">
          <div className="flex items-center gap-3 mb-6">
            <Crosshair className="w-5 h-5 text-military" />
            <h2 className="text-lg font-bold text-white">Operational Feeds</h2>
          </div>
          
          <div className="space-y-4">
            {isLoading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="w-5 h-5 animate-spin text-military" />
              </div>
            ) : error ? (
              <div className="text-sm text-red-400">
                Failed to load military streams.
              </div>
            ) : streams.length > 0 ? (
              streams.map((stream) => (
                <StreamCard key={stream.id} stream={stream} layout="list" />
              ))
            ) : (
              <div className="text-sm text-gray-500">
                No authorized military feeds are available.
              </div>
            )}
          </div>
        </div>

        <div className="hub-card p-6">
          <div className="flex items-center gap-3 mb-6">
            <Target className="w-5 h-5 text-military" />
            <h2 className="text-lg font-bold text-white">Tactical Analysis</h2>
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div className="p-4 rounded-xl bg-hub-surface border border-hub-border">
              <p className="text-xs text-gray-500 uppercase mb-1">Live Feeds</p>
              <p className="text-2xl font-bold text-white">{liveCount}</p>
            </div>
            <div className="p-4 rounded-xl bg-hub-surface border border-hub-border">
              <p className="text-xs text-gray-500 uppercase mb-1">Delayed Feeds</p>
              <p className="text-2xl font-bold text-white">{delayedCount}</p>
            </div>
            <div className="p-4 rounded-xl bg-hub-surface border border-hub-border">
              <p className="text-xs text-gray-500 uppercase mb-1">Offline Feeds</p>
              <p className="text-2xl font-bold text-white">{offlineCount}</p>
            </div>
            <div className="p-4 rounded-xl bg-hub-surface border border-hub-border">
              <p className="text-xs text-gray-500 uppercase mb-1">Signal Quality</p>
              <p className="text-2xl font-bold text-white">N/A</p>
            </div>
          </div>
        </div>
      </div>

      <div className="hub-card p-8 text-center">
        <Shield className="w-16 h-16 text-gray-700 mx-auto mb-4" />
        <p className="text-gray-500">
          Military streams are available to authorized personnel only.
        </p>
        <p className="text-sm text-gray-600 mt-2">
          Feeds appear when authorized sources are active.
        </p>
      </div>
    </div>
  );
}
