/**
 * ASGARD Government Client - Government Login
 * Second security layer - WebAuthn/FIDO2 authentication
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, Key, Fingerprint, AlertTriangle } from 'lucide-react';
import { startAuthentication } from '@simplewebauthn/browser';
import { useAuthStore } from '@/stores/authStore';

export function GovLogin() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [authMethod, setAuthMethod] = useState<'password' | 'webauthn'>('password');
  const { setAuth } = useAuthStore();

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const response = await fetch('/api/auth/gov/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data = await response.json();
      setAuth(data.user, data.token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setIsLoading(false);
    }
  };

  const handleWebAuthnLogin = async () => {
    setError('');
    setIsLoading(true);

    try {
      // Get challenge from server
      const optionsResponse = await fetch('/api/auth/webauthn/authenticate/options', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      });

      if (!optionsResponse.ok) {
        throw new Error('Failed to get authentication options');
      }

      const options = await optionsResponse.json();

      // Authenticate with security key
      const credential = await startAuthentication(options);

      // Verify with server
      const verifyResponse = await fetch('/api/auth/webauthn/authenticate/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, credential }),
      });

      if (!verifyResponse.ok) {
        throw new Error('Authentication verification failed');
      }

      const data = await verifyResponse.json();
      setAuth(data.user, data.token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'WebAuthn authentication failed');
    } finally {
      setIsLoading(false);
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
            className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 mb-4"
          >
            <Shield className="w-8 h-8 text-white" />
          </motion.div>
          <h1 className="text-2xl font-bold text-white mb-2">Secure Login</h1>
          <p className="text-slate-400">Authenticate to access ASGARD Command</p>
        </div>

        {/* Auth Method Toggle */}
        <div className="flex bg-slate-800/50 rounded-lg p-1 mb-6">
          <button
            onClick={() => setAuthMethod('password')}
            className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${
              authMethod === 'password'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            <Key className="w-4 h-4" />
            Password
          </button>
          <button
            onClick={() => setAuthMethod('webauthn')}
            className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${
              authMethod === 'webauthn'
                ? 'bg-emerald-600 text-white'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            <Fingerprint className="w-4 h-4" />
            Security Key
          </button>
        </div>

        {/* Login Form */}
        <div className="bg-slate-800/50 backdrop-blur-xl border border-slate-700 rounded-2xl p-8">
          <form
            onSubmit={authMethod === 'password' ? handlePasswordLogin : (e) => { e.preventDefault(); handleWebAuthnLogin(); }}
            className="space-y-6"
          >
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
                Email Address
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="operator@agency.gov"
                className="w-full px-4 py-3 bg-slate-900 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                required
              />
            </div>

            {authMethod === 'password' && (
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-2">
                  Password
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter your password"
                  className="w-full px-4 py-3 bg-slate-900 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  required
                />
              </div>
            )}

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
              disabled={isLoading || !email}
              className="w-full py-3 bg-gradient-to-r from-emerald-500 to-teal-600 text-white font-semibold rounded-lg hover:from-emerald-600 hover:to-teal-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <motion.div
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                    className="w-5 h-5 border-2 border-white border-t-transparent rounded-full"
                  />
                  Authenticating...
                </span>
              ) : authMethod === 'webauthn' ? (
                <span className="flex items-center justify-center gap-2">
                  <Fingerprint className="w-5 h-5" />
                  Authenticate with Security Key
                </span>
              ) : (
                'Sign In'
              )}
            </button>
          </form>
        </div>

        {/* Footer */}
        <p className="mt-6 text-center text-xs text-slate-600">
          Protected by multi-factor authentication
        </p>
      </motion.div>
    </div>
  );
}
