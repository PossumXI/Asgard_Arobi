const { app, BrowserWindow, ipcMain, dialog, Menu, shell } = require("electron");
const path = require("path");

let mainWindow;
let monitorWindow;
let settingsWindow;
let adminWindow;

const createWindow = () => {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 900,
    backgroundColor: "#0b0f18",
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  mainWindow.loadFile(path.join(__dirname, "renderer", "index.html"));
  
  // Create application menu
  const template = [
    {
      label: "File",
      submenu: [
        {
          label: "Settings",
          accelerator: "CmdOrCtrl+,",
          click: () => openSettingsWindow(),
        },
        {
          label: "Open Monitor Dashboard",
          accelerator: "CmdOrCtrl+M",
          click: () => openMonitorWindow(),
        },
        {
          label: "Open Admin Portal",
          accelerator: "CmdOrCtrl+Shift+A",
          click: () => openAdminWindow(),
        },
        { type: "separator" },
        {
          label: "Quit",
          accelerator: "CmdOrCtrl+Q",
          click: () => app.quit(),
        },
      ],
    },
    {
      label: "View",
      submenu: [
        { role: "reload" },
        { role: "forceReload" },
        { role: "toggleDevTools" },
        { type: "separator" },
        { role: "resetZoom" },
        { role: "zoomIn" },
        { role: "zoomOut" },
        { type: "separator" },
        { role: "togglefullscreen" },
      ],
    },
    {
      label: "AI Models",
      submenu: [
        {
          label: "Get Free API Keys",
          submenu: [
            {
              label: "Groq (Free, Fast)",
              click: () => shell.openExternal("https://console.groq.com"),
            },
            {
              label: "Together AI (Free Tier)",
              click: () => shell.openExternal("https://api.together.xyz"),
            },
            {
              label: "Google AI Studio (Free)",
              click: () => shell.openExternal("https://makersuite.google.com/app/apikey"),
            },
          ],
        },
        { type: "separator" },
        {
          label: "Premium API Keys",
          submenu: [
            {
              label: "Anthropic Claude",
              click: () => shell.openExternal("https://console.anthropic.com"),
            },
            {
              label: "OpenAI",
              click: () => shell.openExternal("https://platform.openai.com"),
            },
            {
              label: "OpenRouter",
              click: () => shell.openExternal("https://openrouter.ai"),
            },
          ],
        },
        { type: "separator" },
        {
          label: "ElevenLabs (Voice)",
          click: () => shell.openExternal("https://elevenlabs.io"),
        },
      ],
    },
    {
      label: "Help",
      submenu: [
        {
          label: "Documentation",
          click: () => {
            shell.openExternal("file://" + path.join(__dirname, "README.md"));
          },
        },
        {
          label: "ASGARD Project",
          click: () => {
            const asgardPath = path.join(__dirname, "..", "..");
            shell.openPath(asgardPath);
          },
        },
      ],
    },
  ];
  
  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
};

const openMonitorWindow = () => {
  if (monitorWindow) {
    monitorWindow.focus();
    return;
  }
  
  monitorWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    backgroundColor: "#0a0e17",
    title: "GIRU Monitor - Control Station",
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
    },
  });
  
  monitorWindow.loadFile(path.join(__dirname, "renderer", "monitor.html"));
  
  monitorWindow.on("closed", () => {
    monitorWindow = null;
  });
};

const openSettingsWindow = () => {
  if (settingsWindow) {
    settingsWindow.focus();
    return;
  }
  
  settingsWindow = new BrowserWindow({
    width: 900,
    height: 800,
    backgroundColor: "#0a0e17",
    title: "GIRU Settings",
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
    },
  });
  
  settingsWindow.loadFile(path.join(__dirname, "renderer", "settings.html"));
  
  settingsWindow.on("closed", () => {
    settingsWindow = null;
  });
};

const openAdminWindow = () => {
  if (adminWindow) {
    adminWindow.focus();
    return;
  }

  adminWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    backgroundColor: "#0a0e17",
    title: "ASGARD Admin Portal",
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  const portalUrl = process.env.ASGARD_ADMIN_PORTAL_URL || "http://localhost:5173/dashboard/admin";
  adminWindow.loadURL(portalUrl).catch(() => {
    adminWindow.loadFile(path.join(__dirname, "renderer", "admin.html"));
  });

  adminWindow.on("closed", () => {
    adminWindow = null;
  });
};

app.whenReady().then(() => {
  createWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

ipcMain.handle("dialog:confirm", async (_event, message) => {
  const result = await dialog.showMessageBox(mainWindow, {
    type: "warning",
    buttons: ["Cancel", "Allow"],
    defaultId: 1,
    cancelId: 0,
    title: "Permission Required",
    message,
  });

  return result.response === 1;
});

// Handle opening monitor from renderer
ipcMain.on("open-monitor", () => {
  openMonitorWindow();
});

// Handle opening settings from renderer
ipcMain.on("open-settings", () => {
  openSettingsWindow();
});

// Handle opening admin portal from renderer
ipcMain.on("open-admin", () => {
  openAdminWindow();
});
