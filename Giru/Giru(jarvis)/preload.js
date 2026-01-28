const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("giruShell", {
  confirm: (message) => ipcRenderer.invoke("dialog:confirm", message),
  openMonitor: () => ipcRenderer.send("open-monitor"),
});
