/**
 * ASGARD Government Client - Main Process
 * Secure Electron application for government and defense operations
 *
 * Copyright 2026 Arobi. All Rights Reserved.
 */

import { app, BrowserWindow, ipcMain, Tray, Menu, nativeImage, session } from 'electron';
import { autoUpdater } from 'electron-updater';
import Store from 'electron-store';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Encrypted store for sensitive data
const store = new Store({
  encryptionKey: process.env.STORE_ENCRYPTION_KEY || 'asgard-gov-default-key',
  name: 'asgard-gov-secure',
});

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;

// Security: Disable remote module and enable context isolation
app.on('web-contents-created', (_, contents) => {
  contents.on('will-navigate', (event, navigationUrl) => {
    const parsedUrl = new URL(navigationUrl);
    // Only allow navigation to localhost during dev and our production URLs
    const allowedOrigins = ['localhost', '127.0.0.1', 'aura-genesis.org'];
    if (!allowedOrigins.some(origin => parsedUrl.hostname.includes(origin))) {
      event.preventDefault();
    }
  });

  // Prevent new windows from being created
  contents.setWindowOpenHandler(() => {
    return { action: 'deny' };
  });
});

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1600,
    height: 1000,
    minWidth: 1200,
    minHeight: 800,
    title: 'ASGARD Command',
    icon: path.join(__dirname, '../assets/icon.png'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
    },
    frame: true,
    autoHideMenuBar: true,
    backgroundColor: '#0a0a0f',
  });

  // Security headers
  session.defaultSession.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': [
          "default-src 'self'; " +
          "script-src 'self' 'unsafe-inline'; " +
          "style-src 'self' 'unsafe-inline'; " +
          "img-src 'self' data: https:; " +
          "connect-src 'self' ws://localhost:* http://localhost:* https://api.aura-genesis.org wss://api.aura-genesis.org; " +
          "font-src 'self' data:;"
        ],
      },
    });
  });

  // Load app
  if (process.env.VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL);
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'));
  }

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Hide to tray instead of closing
  mainWindow.on('close', (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow?.hide();
    }
  });
}

function createTray(): void {
  const iconPath = path.join(__dirname, '../assets/tray-icon.png');
  const icon = nativeImage.createFromPath(iconPath);
  tray = new Tray(icon.resize({ width: 16, height: 16 }));

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Open ASGARD Command',
      click: () => mainWindow?.show(),
    },
    {
      label: 'Quick Status',
      submenu: [
        { label: 'Systems: Online', enabled: false },
        { label: 'Alerts: 0 Critical', enabled: false },
        { label: 'Missions: 3 Active', enabled: false },
      ],
    },
    { type: 'separator' },
    {
      label: 'Check for Updates',
      click: () => autoUpdater.checkForUpdatesAndNotify(),
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        app.isQuitting = true;
        app.quit();
      },
    },
  ]);

  tray.setToolTip('ASGARD Command');
  tray.setContextMenu(contextMenu);
  tray.on('double-click', () => mainWindow?.show());
}

// IPC Handlers for secure communication with renderer
ipcMain.handle('store:get', (_, key: string) => {
  return store.get(key);
});

ipcMain.handle('store:set', (_, key: string, value: unknown) => {
  store.set(key, value);
});

ipcMain.handle('store:delete', (_, key: string) => {
  store.delete(key);
});

ipcMain.handle('auth:validateAccessCode', async (_, code: string) => {
  // Validate access code against backend
  try {
    const response = await fetch('http://localhost:8080/api/gov/validate-access', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code }),
    });
    return response.ok;
  } catch {
    return false;
  }
});

ipcMain.handle('app:getVersion', () => {
  return app.getVersion();
});

ipcMain.handle('app:getPlatform', () => {
  return process.platform;
});

// Auto-updater events
autoUpdater.on('update-available', () => {
  mainWindow?.webContents.send('update:available');
});

autoUpdater.on('update-downloaded', () => {
  mainWindow?.webContents.send('update:downloaded');
});

ipcMain.handle('update:install', () => {
  autoUpdater.quitAndInstall();
});

// App lifecycle
app.whenReady().then(() => {
  createWindow();
  createTray();
  autoUpdater.checkForUpdatesAndNotify();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});

app.on('before-quit', () => {
  app.isQuitting = true;
});

// Extend app type for isQuitting flag
declare module 'electron' {
  interface App {
    isQuitting: boolean;
  }
}

app.isQuitting = false;
