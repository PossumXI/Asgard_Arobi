/**
 * ASGARD Government Client - Loading Screen
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { motion } from 'framer-motion';
import { Shield } from 'lucide-react';

export function LoadingScreen() {
  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center">
      <div className="text-center">
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 mb-6"
        >
          <Shield className="w-10 h-10 text-white" />
        </motion.div>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.3 }}
        >
          <h1 className="text-2xl font-bold text-white mb-2">ASGARD Command</h1>
          <p className="text-slate-400 mb-6">Initializing secure connection...</p>

          <div className="flex items-center justify-center gap-2">
            <motion.div
              animate={{ scale: [1, 1.2, 1] }}
              transition={{ duration: 0.6, repeat: Infinity, delay: 0 }}
              className="w-2 h-2 bg-amber-500 rounded-full"
            />
            <motion.div
              animate={{ scale: [1, 1.2, 1] }}
              transition={{ duration: 0.6, repeat: Infinity, delay: 0.2 }}
              className="w-2 h-2 bg-amber-500 rounded-full"
            />
            <motion.div
              animate={{ scale: [1, 1.2, 1] }}
              transition={{ duration: 0.6, repeat: Infinity, delay: 0.4 }}
              className="w-2 h-2 bg-amber-500 rounded-full"
            />
          </div>
        </motion.div>
      </div>
    </div>
  );
}
