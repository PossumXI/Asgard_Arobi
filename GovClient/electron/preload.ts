/**
 * ASGARD Government Client - Preload Script
 * Secure context bridge for renderer process
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { contextBridge, ipcRenderer } from 'electron';

// Expose secure API to renderer
contextBridge.exposeInMainWorld('asgardAPI', {
  // Secure storage
  store: {
    get: (key: string) => ipcRenderer.invoke('store:get', key),
    set: (key: string, value: unknown) => ipcRenderer.invoke('store:set', key, value),
    delete: (key: string) => ipcRenderer.invoke('store:delete', key),
  },

  // Authentication
  auth: {
    validateAccessCode: (code: string) => ipcRenderer.invoke('auth:validateAccessCode', code),
  },

  // App info
  app: {
    getVersion: () => ipcRenderer.invoke('app:getVersion'),
    getPlatform: () => ipcRenderer.invoke('app:getPlatform'),
  },

  // Auto-updater
  update: {
    install: () => ipcRenderer.invoke('update:install'),
    onAvailable: (callback: () => void) => {
      ipcRenderer.on('update:available', callback);
      return () => ipcRenderer.removeListener('update:available', callback);
    },
    onDownloaded: (callback: () => void) => {
      ipcRenderer.on('update:downloaded', callback);
      return () => ipcRenderer.removeListener('update:downloaded', callback);
    },
  },

  // System notifications
  notification: {
    show: (title: string, body: string) => {
      new Notification(title, { body });
    },
  },
});

// Type definitions for renderer
export interface AsgardAPI {
  store: {
    get: (key: string) => Promise<unknown>;
    set: (key: string, value: unknown) => Promise<void>;
    delete: (key: string) => Promise<void>;
  };
  auth: {
    validateAccessCode: (code: string) => Promise<boolean>;
  };
  app: {
    getVersion: () => Promise<string>;
    getPlatform: () => Promise<string>;
  };
  update: {
    install: () => Promise<void>;
    onAvailable: (callback: () => void) => () => void;
    onDownloaded: (callback: () => void) => () => void;
  };
  notification: {
    show: (title: string, body: string) => void;
  };
}

declare global {
  interface Window {
    asgardAPI: AsgardAPI;
  }
}
