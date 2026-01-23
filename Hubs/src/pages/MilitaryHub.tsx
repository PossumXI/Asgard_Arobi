import { useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, Lock, AlertTriangle, Key } from 'lucide-react';

export default function MilitaryHub() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);

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
                  className="w-full h-12 pl-11 pr-4 rounded-xl bg-hub-surface border border-hub-border text-white placeholder-gray-500 focus:outline-none focus:border-military"
                />
              </div>

              <button
                onClick={() => setIsAuthenticated(true)}
                className="w-full h-12 rounded-xl bg-military text-white font-medium hover:bg-military/90 transition-colors flex items-center justify-center gap-2"
              >
                <Key className="w-4 h-4" />
                Verify Clearance
              </button>
            </div>

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

      <div className="hub-card p-8 text-center">
        <Shield className="w-16 h-16 text-gray-700 mx-auto mb-4" />
        <p className="text-gray-500">
          Military streams are available to authorized personnel only.
        </p>
        <p className="text-sm text-gray-600 mt-2">
          This is a demonstration environment.
        </p>
      </div>
    </div>
  );
}
