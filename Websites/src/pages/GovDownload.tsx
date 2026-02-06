/**
 * ASGARD Websites - Government Client Download Page
 * Secure download portal for government personnel
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Shield, Download, Key, Monitor, Apple, Terminal, CheckCircle, AlertTriangle, Lock } from 'lucide-react';

interface DownloadInfo {
  platform: 'windows' | 'mac' | 'linux';
  filename: string;
  size: string;
  version: string;
  downloadUrl: string;
}

const downloads: DownloadInfo[] = [
  {
    platform: 'windows',
    filename: 'ASGARD-Command-1.0.0-Setup.exe',
    size: '148 MB',
    version: '1.0.0',
    downloadUrl: '/api/gov/download/windows',
  },
  {
    platform: 'mac',
    filename: 'ASGARD-Command-1.0.0.dmg',
    size: '156 MB',
    version: '1.0.0',
    downloadUrl: '/api/gov/download/mac',
  },
  {
    platform: 'linux',
    filename: 'ASGARD-Command-1.0.0.AppImage',
    size: '142 MB',
    version: '1.0.0',
    downloadUrl: '/api/gov/download/linux',
  },
];

const getPlatformIcon = (platform: string) => {
  switch (platform) {
    case 'windows':
      return Monitor;
    case 'mac':
      return Apple;
    case 'linux':
      return Terminal;
    default:
      return Monitor;
  }
};

const detectPlatform = (): 'windows' | 'mac' | 'linux' => {
  const userAgent = navigator.userAgent.toLowerCase();
  if (userAgent.includes('win')) return 'windows';
  if (userAgent.includes('mac')) return 'mac';
  return 'linux';
};

export function GovDownload() {
  const [accessCode, setAccessCode] = useState('');
  const [isVerified, setIsVerified] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState('');
  const [detectedPlatform] = useState(detectPlatform());
  const [downloadStarted, setDownloadStarted] = useState<string | null>(null);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsVerifying(true);

    try {
      const response = await fetch('/api/gov/validate-access', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: accessCode }),
      });

      if (response.ok) {
        const data = await response.json();
        if (data.valid && data.clearanceLevel && ['government', 'admin'].includes(data.clearanceLevel)) {
          setIsVerified(true);
          // Store verification token
          sessionStorage.setItem('gov_download_token', data.token);
        } else {
          setError('Access code does not have sufficient clearance level.');
        }
      } else {
        setError('Invalid access code. Please contact your administrator.');
      }
    } catch {
      setError('Verification failed. Please check your connection.');
    } finally {
      setIsVerifying(false);
    }
  };

  const handleDownload = async (download: DownloadInfo) => {
    setDownloadStarted(download.platform);

    const token = sessionStorage.getItem('gov_download_token');
    if (!token) {
      setError('Session expired. Please verify your access code again.');
      setIsVerified(false);
      return;
    }

    try {
      const response = await fetch(download.downloadUrl, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = download.filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        a.remove();
      } else {
        setError('Download failed. Your session may have expired.');
      }
    } catch {
      setError('Download failed. Please try again.');
    } finally {
      setDownloadStarted(null);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 py-20">
      <div className="container mx-auto px-4 max-w-4xl">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-12"
        >
          <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-gradient-to-br from-amber-500 to-orange-600 mb-6">
            <Shield className="w-10 h-10 text-white" />
          </div>
          <h1 className="text-4xl font-bold text-white mb-4">ASGARD Command</h1>
          <p className="text-xl text-slate-400">Government & Defense Operations Client</p>
        </motion.div>

        {!isVerified ? (
          /* Access Code Verification */
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="bg-slate-800/50 backdrop-blur-xl border border-slate-700 rounded-2xl p-8 max-w-md mx-auto"
          >
            <div className="flex items-center gap-2 mb-6 text-amber-500">
              <Lock className="w-5 h-5" />
              <span className="text-sm font-medium">CLASSIFIED ACCESS REQUIRED</span>
            </div>

            <form onSubmit={handleVerify} className="space-y-6">
              <div>
                <label htmlFor="accessCode" className="block text-sm font-medium text-slate-300 mb-2">
                  Government Access Code
                </label>
                <div className="relative">
                  <Key className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500" />
                  <input
                    id="accessCode"
                    type="password"
                    value={accessCode}
                    onChange={(e) => setAccessCode(e.target.value.toUpperCase())}
                    placeholder="XXXX-XXXX-XXXX-XXXX"
                    className="w-full pl-10 pr-4 py-3 bg-slate-900 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent font-mono tracking-wider"
                    required
                    autoFocus
                  />
                </div>
              </div>

              {error && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="flex items-center gap-2 text-red-400 text-sm p-3 bg-red-500/10 border border-red-500/30 rounded-lg"
                >
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                  {error}
                </motion.div>
              )}

              <button
                type="submit"
                disabled={isVerifying || accessCode.length < 8}
                className="w-full py-3 bg-gradient-to-r from-amber-500 to-orange-600 text-white font-semibold rounded-lg hover:from-amber-600 hover:to-orange-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center justify-center gap-2"
              >
                {isVerifying ? (
                  <>
                    <motion.div
                      animate={{ rotate: 360 }}
                      transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                      className="w-5 h-5 border-2 border-white border-t-transparent rounded-full"
                    />
                    Verifying...
                  </>
                ) : (
                  <>
                    <Shield className="w-5 h-5" />
                    Verify Access
                  </>
                )}
              </button>
            </form>

            <p className="mt-6 text-xs text-slate-500 text-center">
              This download is restricted to authorized government and defense personnel only.
              All access attempts are logged and monitored.
            </p>
          </motion.div>
        ) : (
          /* Download Options */
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="space-y-6"
          >
            <div className="flex items-center justify-center gap-2 text-emerald-400 mb-8">
              <CheckCircle className="w-5 h-5" />
              <span>Access verified. You may now download the application.</span>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {downloads.map((download) => {
                const Icon = getPlatformIcon(download.platform);
                const isRecommended = download.platform === detectedPlatform;
                const isDownloading = downloadStarted === download.platform;

                return (
                  <motion.div
                    key={download.platform}
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    className={`relative bg-slate-800/50 border rounded-xl p-6 ${
                      isRecommended ? 'border-amber-500/50' : 'border-slate-700'
                    }`}
                  >
                    {isRecommended && (
                      <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 bg-amber-500 text-white text-xs font-medium rounded-full">
                        Recommended
                      </div>
                    )}

                    <div className="text-center mb-4">
                      <Icon className="w-12 h-12 mx-auto text-slate-400 mb-3" />
                      <h3 className="text-lg font-semibold text-white capitalize">
                        {download.platform}
                      </h3>
                      <p className="text-sm text-slate-500">v{download.version}</p>
                    </div>

                    <div className="text-center mb-4">
                      <p className="text-xs text-slate-500">{download.filename}</p>
                      <p className="text-sm text-slate-400">{download.size}</p>
                    </div>

                    <button
                      onClick={() => handleDownload(download)}
                      disabled={isDownloading}
                      className={`w-full py-2 rounded-lg font-medium flex items-center justify-center gap-2 transition-all ${
                        isRecommended
                          ? 'bg-amber-500 text-white hover:bg-amber-600'
                          : 'bg-slate-700 text-white hover:bg-slate-600'
                      } disabled:opacity-50`}
                    >
                      {isDownloading ? (
                        <>
                          <motion.div
                            animate={{ rotate: 360 }}
                            transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                            className="w-4 h-4 border-2 border-white border-t-transparent rounded-full"
                          />
                          Downloading...
                        </>
                      ) : (
                        <>
                          <Download className="w-4 h-4" />
                          Download
                        </>
                      )}
                    </button>
                  </motion.div>
                );
              })}
            </div>

            {/* Installation Instructions */}
            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6 mt-8">
              <h3 className="text-lg font-semibold text-white mb-4">Installation Instructions</h3>
              <div className="space-y-4 text-sm text-slate-400">
                <div>
                  <h4 className="font-medium text-white mb-1">Windows</h4>
                  <p>Run the installer and follow the prompts. You may need to allow the app through Windows Defender.</p>
                </div>
                <div>
                  <h4 className="font-medium text-white mb-1">macOS</h4>
                  <p>Open the DMG file and drag ASGARD Command to your Applications folder. Right-click and select Open the first time.</p>
                </div>
                <div>
                  <h4 className="font-medium text-white mb-1">Linux</h4>
                  <p>Make the AppImage executable with `chmod +x` and run it. Or use your distribution's package manager.</p>
                </div>
              </div>
            </div>

            {/* Security Notice */}
            <div className="bg-amber-500/10 border border-amber-500/30 rounded-xl p-6">
              <h3 className="text-lg font-semibold text-amber-400 mb-2">Security Notice</h3>
              <p className="text-sm text-slate-400">
                This application is designed for use on secure government networks. Do not install on personal devices
                or unsecured systems. All communications are encrypted and require valid government credentials.
              </p>
            </div>
          </motion.div>
        )}
      </div>
    </div>
  );
}

export default GovDownload;
