/**
 * ASGARD Government Client - Access Code Gate
 * First security layer - validates government access codes
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, Lock, AlertTriangle } from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';

export function AccessCodeGate() {
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [isValidating, setIsValidating] = useState(false);
  const { setAccessCode } = useAuthStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsValidating(true);

    try {
      // Validate with backend via electron IPC
      const isValid = await window.asgardAPI.auth.validateAccessCode(code);

      if (isValid) {
        // Set 24-hour expiry
        const expiry = Date.now() + 24 * 60 * 60 * 1000;
        setAccessCode(expiry);
      } else {
        setError('Invalid access code. Contact your administrator.');
      }
    } catch {
      setError('Validation failed. Check your connection.');
    } finally {
      setIsValidating(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full max-w-md"
      >
        {/* Header */}
        <div className="text-center mb-8">
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ type: 'spring', duration: 0.5 }}
            className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 mb-4"
          >
            <Shield className="w-10 h-10 text-white" />
          </motion.div>
          <h1 className="text-3xl font-bold text-white mb-2">ASGARD Command</h1>
          <p className="text-slate-400">Government & Defense Operations</p>
        </div>

        {/* Access Code Form */}
        <div className="bg-slate-800/50 backdrop-blur-xl border border-slate-700 rounded-2xl p-8">
          <div className="flex items-center gap-2 mb-6 text-amber-500">
            <Lock className="w-5 h-5" />
            <span className="text-sm font-medium">CLASSIFIED ACCESS</span>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label
                htmlFor="accessCode"
                className="block text-sm font-medium text-slate-300 mb-2"
              >
                Enter Access Code
              </label>
              <input
                id="accessCode"
                type="password"
                value={code}
                onChange={(e) => setCode(e.target.value.toUpperCase())}
                placeholder="XXXX-XXXX-XXXX-XXXX"
                className="w-full px-4 py-3 bg-slate-900 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent font-mono tracking-wider"
                required
                autoFocus
              />
            </div>

            {error && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="flex items-center gap-2 text-red-400 text-sm"
              >
                <AlertTriangle className="w-4 h-4" />
                {error}
              </motion.div>
            )}

            <button
              type="submit"
              disabled={isValidating || code.length < 8}
              className="w-full py-3 bg-gradient-to-r from-amber-500 to-orange-600 text-white font-semibold rounded-lg hover:from-amber-600 hover:to-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            >
              {isValidating ? (
                <span className="flex items-center justify-center gap-2">
                  <motion.div
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                    className="w-5 h-5 border-2 border-white border-t-transparent rounded-full"
                  />
                  Validating...
                </span>
              ) : (
                'Verify Access'
              )}
            </button>
          </form>

          <p className="mt-6 text-xs text-slate-500 text-center">
            This system is for authorized government personnel only.
            Unauthorized access attempts are logged and prosecuted.
          </p>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-xs text-slate-600">
          ASGARD Command v1.0.0 | Arobi
        </p>
      </motion.div>
    </div>
  );
}
