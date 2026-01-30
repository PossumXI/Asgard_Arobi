import { test, Page } from '@playwright/test';
import path from 'path';
import { pathToFileURL } from 'url';

const WEBSITES_URL = process.env.ASGARD_WEBSITES_URL ?? 'http://localhost:3000';
const HUBS_URL = process.env.ASGARD_HUBS_URL ?? 'http://localhost:3001';
const GIRU_JARVIS_PATH = path.resolve(__dirname, '../../Giru/Giru(jarvis)/renderer/index.html');
const GIRU_MONITOR_PATH = path.resolve(__dirname, '../../Giru/Giru(jarvis)/renderer/monitor.html');

test.describe('ASGARD Complete Demo', () => {
  test('Full system demo video', async ({ page }) => {
    test.setTimeout(600000);
    await page.setViewportSize({ width: 1920, height: 1080 });

    await page.setContent(getBackdropPage());
    await showTitle(page, 'ASGARD FULL SYSTEM DEMO', 'All systems, hubs, and AI assistant');
    await pause(3500);

    // -----------------------------------------------------------------------
    // Websites Portal
    // -----------------------------------------------------------------------
    await showSection(page, 'Websites Portal', 'Public + Government dashboards');
    await pause(1500);
    await safeGoto(page, WEBSITES_URL, 'Websites Landing');
    await pause(2500);
    await safeGoto(page, `${WEBSITES_URL}/features`, 'Features Overview');
    await pause(2500);
    await safeGoto(page, `${WEBSITES_URL}/pricilla`, 'Pricilla Overview');
    await pause(2500);
    await safeGoto(page, `${WEBSITES_URL}/dashboard`, 'Operations Dashboard');
    await pause(2500);
    await safeGoto(page, `${WEBSITES_URL}/gov`, 'Government Portal');
    await pause(2500);

    // -----------------------------------------------------------------------
    // Hubs Streaming UI
    // -----------------------------------------------------------------------
    await showSection(page, 'Hubs Streaming', 'Civilian / Military / Interstellar');
    await pause(1500);
    await safeGoto(page, HUBS_URL, 'Hubs Home');
    await pause(2000);
    await safeGoto(page, `${HUBS_URL}/civilian`, 'Civilian Hub');
    await pause(2000);
    await safeGoto(page, `${HUBS_URL}/military`, 'Military Hub');
    await pause(2000);
    await safeGoto(page, `${HUBS_URL}/interstellar`, 'Interstellar Hub');
    await pause(2000);
    await safeGoto(page, `${HUBS_URL}/missions`, 'Mission Hub');
    await pause(2500);

    // -----------------------------------------------------------------------
    // Control Hub (Control_net)
    // -----------------------------------------------------------------------
    await showSection(page, 'Control Hub', 'Kubernetes control plane');
    await pause(1500);
    await page.setContent(getControlHubPage());
    await pause(3500);

    // -----------------------------------------------------------------------
    // GIRU JARVIS - Voice + Command Demo
    // -----------------------------------------------------------------------
    await showSection(page, 'GIRU JARVIS', 'Voice control & command execution');
    await pause(1500);
    await safeGoto(page, pathToFileURL(GIRU_JARVIS_PATH).toString(), 'GIRU JARVIS UI');
    await pause(1500);
    await simulateJarvisConversation(page);
    await pause(4500);

    // -----------------------------------------------------------------------
    // GIRU Monitor
    // -----------------------------------------------------------------------
    await showSection(page, 'GIRU Monitor', 'Live activity dashboard');
    await pause(1500);
    await safeGoto(page, pathToFileURL(GIRU_MONITOR_PATH).toString(), 'GIRU Monitor UI');
    await pause(3500);

    // -----------------------------------------------------------------------
    // Integration Wrap-up
    // -----------------------------------------------------------------------
    await showSection(page, 'System Integration', 'End-to-end verification');
    await pause(1500);
    await page.setContent(getIntegrationPage());
    await pause(4000);

    await showTitle(page, 'DEMO COMPLETE', 'ASGARD systems showcased end-to-end');
    await pause(3000);
  });
});

// =============================================================================
// Helpers
// =============================================================================

async function pause(ms: number) {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

async function safeGoto(page: Page, url: string, label: string) {
  try {
    const response = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 15000 });
    if (!response || !response.ok()) {
      throw new Error(`Navigation failed (${response?.status() ?? 'no response'})`);
    }
  } catch (err) {
    await page.setContent(getFallbackPage(label, url, String(err)));
  }
}

async function showTitle(page: Page, title: string, subtitle: string) {
  await page.evaluate(({ title, subtitle }) => {
    const existing = document.getElementById('demo-title');
    if (existing) existing.remove();

    const container = document.createElement('div');
    container.id = 'demo-title';
    container.innerHTML = `
      <div class="title-main">${title}</div>
      <div class="title-sub">${subtitle}</div>
    `;
    container.style.cssText = `
      position: fixed;
      top: 50%; left: 50%;
      transform: translate(-50%, -50%);
      text-align: center;
      z-index: 10000;
    `;

    const style = document.createElement('style');
    style.id = 'title-style';
    style.textContent = `
      .title-main {
        font-size: 64px;
        font-weight: 800;
        letter-spacing: 6px;
        background: linear-gradient(135deg, #8b5cf6, #3b82f6, #06b6d4);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        margin-bottom: 16px;
      }
      .title-sub {
        font-size: 18px;
        color: rgba(255,255,255,0.7);
        letter-spacing: 4px;
        text-transform: uppercase;
      }
    `;
    document.getElementById('title-style')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(container);
  }, { title, subtitle });
}

async function showSection(page: Page, title: string, subtitle: string) {
  await page.evaluate(({ title, subtitle }) => {
    const existing = document.getElementById('section-card');
    if (existing) existing.remove();

    const card = document.createElement('div');
    card.id = 'section-card';
    card.innerHTML = `
      <div class="section-title">${title}</div>
      <div class="section-sub">${subtitle}</div>
    `;
    card.style.cssText = `
      position: fixed;
      top: 60px;
      left: 60px;
      padding: 18px 26px;
      background: rgba(10, 10, 26, 0.92);
      border: 1px solid rgba(139, 92, 246, 0.4);
      border-radius: 12px;
      color: white;
      z-index: 10000;
    `;

    const style = document.createElement('style');
    style.id = 'section-style';
    style.textContent = `
      .section-title {
        font-size: 18px;
        font-weight: 600;
        letter-spacing: 1px;
      }
      .section-sub {
        font-size: 12px;
        color: rgba(255,255,255,0.7);
        margin-top: 6px;
      }
    `;
    document.getElementById('section-style')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(card);
  }, { title, subtitle });
}

async function simulateJarvisConversation(page: Page) {
  await page.evaluate(() => {
    const core = document.getElementById('core');
    const statusText = document.getElementById('status-text');
    const voiceVisualizer = document.getElementById('voice-visualizer');
    const conversation = document.getElementById('conversation');
    const systemLog = document.getElementById('system-log');

    if (core) core.className = 'arc-reactor speaking';
    if (statusText) statusText.textContent = 'SPEAKING';
    if (voiceVisualizer) {
      voiceVisualizer.classList.add('active', 'speaking');
    }

    const addBubble = (role: string, text: string) => {
      if (!conversation) return;
      const bubble = document.createElement('div');
      bubble.className = `bubble ${role}`;
      bubble.textContent = text;
      conversation.appendChild(bubble);
      conversation.scrollTop = conversation.scrollHeight;
    };

    const addLog = (message: string, level = 'info') => {
      if (!systemLog) return;
      const entry = document.createElement('div');
      entry.className = `log-entry ${level}`;
      entry.textContent = `[${new Date().toLocaleTimeString('en-US', { hour12: false })}] ${message}`;
      systemLog.appendChild(entry);
      systemLog.scrollTop = systemLog.scrollHeight;
    };

    addLog('Wake word detected: "Giru"', 'info');
    addBubble('user', 'Giru, run system status and open Hubs.');
    addBubble('assistant', 'All core systems online. Opening Hubs dashboard now.');
    addLog('Command executed: system_status', 'success');
    addLog('Command executed: open_hubs', 'success');
  });
}

function getBackdropPage(): string {
  return `
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        min-height: 100vh;
        background: radial-gradient(circle at top, #1f1f3a, #0a0a1a);
        font-family: 'Segoe UI', sans-serif;
        color: white;
      }
    </style>
  </head>
  <body></body>
</html>`;
}

function getFallbackPage(label: string, url: string, reason: string): string {
  return `
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        min-height: 100vh;
        background: linear-gradient(135deg, #101025, #0a0a1a);
        color: white;
        font-family: 'Segoe UI', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .card {
        max-width: 720px;
        padding: 32px;
        border-radius: 16px;
        background: rgba(255,255,255,0.05);
        border: 1px solid rgba(139,92,246,0.4);
        box-shadow: 0 20px 60px rgba(0,0,0,0.4);
      }
      .title { font-size: 24px; font-weight: 700; margin-bottom: 12px; }
      .meta { font-size: 14px; color: rgba(255,255,255,0.7); }
      .reason { margin-top: 12px; font-size: 12px; color: rgba(255,255,255,0.6); }
    </style>
  </head>
  <body>
    <div class="card">
      <div class="title">${label}</div>
      <div class="meta">Fallback view (service offline)</div>
      <div class="meta">URL: ${url}</div>
      <div class="reason">Reason: ${reason}</div>
    </div>
  </body>
</html>`;
}

function getControlHubPage(): string {
  return `
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        min-height: 100vh;
        background: linear-gradient(135deg, #0a0a1a, #16162f);
        color: white;
        font-family: 'Segoe UI', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .panel {
        padding: 36px;
        border-radius: 18px;
        background: rgba(15, 15, 35, 0.95);
        border: 1px solid rgba(139,92,246,0.3);
        width: 700px;
      }
      .title { font-size: 24px; font-weight: 700; margin-bottom: 18px; }
      .grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 14px; }
      .item {
        padding: 14px 16px;
        background: rgba(255,255,255,0.04);
        border-radius: 12px;
        border: 1px solid rgba(139,92,246,0.2);
        display: flex;
        align-items: center;
        justify-content: space-between;
        font-size: 13px;
      }
      .status { color: #22c55e; font-weight: 600; }
    </style>
  </head>
  <body>
    <div class="panel">
      <div class="title">Control_net Deployment Overview</div>
      <div class="grid">
        <div class="item"><span>Nysus API</span><span class="status">Ready</span></div>
        <div class="item"><span>Silenus</span><span class="status">Ready</span></div>
        <div class="item"><span>Hunoid</span><span class="status">Ready</span></div>
        <div class="item"><span>Giru</span><span class="status">Ready</span></div>
        <div class="item"><span>PostgreSQL</span><span class="status">Ready</span></div>
        <div class="item"><span>MongoDB</span><span class="status">Ready</span></div>
      </div>
    </div>
  </body>
</html>`;
}

function getIntegrationPage(): string {
  return `
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        min-height: 100vh;
        background: linear-gradient(135deg, #0a0a1a, #14142c);
        color: white;
        font-family: 'Segoe UI', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .panel {
        padding: 36px 40px;
        border-radius: 18px;
        background: rgba(15, 15, 35, 0.95);
        border: 1px solid rgba(59,130,246,0.4);
        width: 760px;
      }
      .title { font-size: 26px; font-weight: 700; margin-bottom: 20px; }
      .row { display: flex; justify-content: space-between; margin-bottom: 12px; font-size: 14px; }
      .ok { color: #22c55e; font-weight: 600; }
    </style>
  </head>
  <body>
    <div class="panel">
      <div class="title">Integration Status</div>
      <div class="row"><span>Giru → Pricilla</span><span class="ok">Threat zones synced</span></div>
      <div class="row"><span>Silenus → Nysus</span><span class="ok">Telemetry streaming</span></div>
      <div class="row"><span>Hunoid → Nysus</span><span class="ok">Mission dispatch active</span></div>
      <div class="row"><span>NATS → WebSockets</span><span class="ok">Realtime bridge live</span></div>
      <div class="row"><span>Hubs → Users</span><span class="ok">Streams available</span></div>
    </div>
  </body>
</html>`;
}
