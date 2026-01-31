import { test, Page, APIRequestContext } from '@playwright/test';
import path from 'path';
import { pathToFileURL } from 'url';

const WEBSITES_URL = process.env.ASGARD_WEBSITES_URL ?? 'http://localhost:3000';
const HUBS_URL = process.env.ASGARD_HUBS_URL ?? 'http://localhost:3001';
const NYSUS_HEALTH_URL = process.env.ASGARD_NYSUS_HEALTH ?? 'http://localhost:8080/health';
const GIRU_HEALTH_URL = process.env.ASGARD_GIRU_HEALTH ?? 'http://localhost:9090/health';
const PRICILLA_HEALTH_URL = process.env.ASGARD_PRICILLA_HEALTH ?? 'http://localhost:8092/health';
const SILENUS_HEALTH_URL = process.env.ASGARD_SILENUS_HEALTH ?? 'http://localhost:9093/healthz';
const HUNOID_HEALTH_URL = process.env.ASGARD_HUNOID_HEALTH ?? 'http://localhost:9094/healthz';
const NATS_HEALTH_URL = process.env.ASGARD_NATS_HEALTH ?? 'http://localhost:8222/healthz';
const NYSUS_METRICS_URL = process.env.ASGARD_NYSUS_METRICS ?? 'http://localhost:8080/metrics';
const GIRU_METRICS_URL = process.env.ASGARD_GIRU_METRICS ?? 'http://localhost:9091/metrics';
const PRICILLA_METRICS_URL = process.env.ASGARD_PRICILLA_METRICS ?? 'http://localhost:9092/metrics';
const SILENUS_METRICS_URL = process.env.ASGARD_SILENUS_METRICS ?? 'http://localhost:9093/metrics';
const HUNOID_METRICS_URL = process.env.ASGARD_HUNOID_METRICS ?? 'http://localhost:9094/metrics';
const VALKYRIE_HEALTH_URL = process.env.ASGARD_VALKYRIE_HEALTH ?? 'http://localhost:8093/health';
const VALKYRIE_STATUS_URL = process.env.ASGARD_VALKYRIE_STATUS ?? 'http://localhost:8093/api/v1/status';
const VALKYRIE_STATE_URL = process.env.ASGARD_VALKYRIE_STATE ?? 'http://localhost:8093/api/v1/state';
const VALKYRIE_METRICS_URL = process.env.ASGARD_VALKYRIE_METRICS ?? 'http://localhost:9095/metrics';
const GIRU_JARVIS_PATH = path.resolve(__dirname, '../../Giru/Giru(jarvis)/renderer/index.html');
const GIRU_MONITOR_PATH = path.resolve(__dirname, '../../Giru/Giru(jarvis)/renderer/monitor.html');

test.describe('ASGARD Complete Demo', () => {
  test('Full system demo video', async ({ page, request }) => {
    test.setTimeout(600000);
    await page.setViewportSize({ width: 1920, height: 1080 });

    await page.setContent(getBackdropPage());
    await showTitle(page, 'ASGARD FULL SYSTEM DEMO', 'All systems, hubs, and AI assistant');
    await pause(3500);

    await ensureLiveOverlay(page);
    const stopLiveMetrics = startLiveMetricsPolling(page, request);

    // -----------------------------------------------------------------------
    // Websites Portal
    // -----------------------------------------------------------------------
    await showSection(page, 'Websites Portal', 'Public + Government dashboards');
    await pause(1500);
    await safeGoto(page, WEBSITES_URL, 'Websites Landing');
    await pause(2500);
    await safeGoto(page, `${WEBSITES_URL}/about`, 'About');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/features`, 'Features Overview');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/pricilla`, 'Pricilla Overview');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/valkyrie`, 'Valkyrie Overview');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/giru`, 'Giru Overview');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/pricing`, 'Pricing');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/contact`, 'Contact');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/signin`, 'Sign In');
    await pause(1500);
    await safeGoto(page, `${WEBSITES_URL}/signup`, 'Sign Up');
    await pause(1500);
    await safeGoto(page, `${WEBSITES_URL}/dashboard`, 'Operations Dashboard');
    await pause(2000);
    await safeGoto(page, `${WEBSITES_URL}/dashboard/admin`, 'Admin Hub');
    await pause(1500);
    await safeGoto(page, `${WEBSITES_URL}/dashboard/command`, 'Command Hub');
    await pause(1500);
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
    const controlHealth = await fetchHealthSnapshot(request);
    await page.setContent(getControlHubPage(controlHealth));
    await pause(3500);

    // -----------------------------------------------------------------------
    // Valkyrie Flight System
    // -----------------------------------------------------------------------
    await showSection(page, 'Valkyrie Flight System', 'Live status + metrics validation');
    await pause(1500);
    const valkyrieSnapshot = await fetchValkyrieSnapshot(request);
    await page.setContent(getValkyrieStatusPage(valkyrieSnapshot));
    await pause(3500);

    // -----------------------------------------------------------------------
    // GIRU JARVIS - Voice + Command Demo
    // -----------------------------------------------------------------------
    await showSection(page, 'GIRU JARVIS', 'Electron command center + live monitor');
    await pause(1500);
    await safeGoto(page, pathToFileURL(GIRU_JARVIS_PATH).toString(), 'GIRU JARVIS Electron UI');
    await pause(1500);
    await triggerJarvisCommand(page, 'Giru, report system status and show Pricilla trajectory.');
    await pause(6000);

    // -----------------------------------------------------------------------
    // GIRU Monitor
    // -----------------------------------------------------------------------
    await showSection(page, 'GIRU Monitor', 'Live activity dashboard');
    await pause(1500);
    await safeGoto(page, pathToFileURL(GIRU_MONITOR_PATH).toString(), 'GIRU Electron Monitor UI');
    await pause(3500);

    // -----------------------------------------------------------------------
    // Integration Wrap-up
    // -----------------------------------------------------------------------
    await showSection(page, 'System Integration', 'End-to-end verification');
    await pause(1500);
    const healthSnapshot = await fetchHealthSnapshot(request);
    await page.setContent(getIntegrationPage(healthSnapshot));
    await pause(4000);

    await showTitle(page, 'DEMO COMPLETE', 'ASGARD systems showcased end-to-end');
    await pause(3000);

    stopLiveMetrics();
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

async function ensureLiveOverlay(page: Page) {
  await page.evaluate(() => {
    if (document.getElementById('live-metrics-overlay')) return;
    const overlay = document.createElement('div');
    overlay.id = 'live-metrics-overlay';
    overlay.innerHTML = `
      <div class="live-title">Live Metrics</div>
      <div class="live-grid">
        <div class="live-card">
          <div class="live-label">Nysus</div>
          <div class="live-value" data-metric="nysus-health">...</div>
          <div class="live-sub" data-metric="nysus-http">HTTP: ...</div>
          <div class="live-sub" data-metric="nysus-events">Events: ...</div>
          <div class="live-sub" data-metric="nysus-db">DB: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">Giru</div>
          <div class="live-value" data-metric="giru-health">...</div>
          <div class="live-sub" data-metric="giru-threats">Threats: ...</div>
          <div class="live-sub" data-metric="giru-packets">Packets: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">Pricilla</div>
          <div class="live-value" data-metric="pricilla-health">...</div>
          <div class="live-sub" data-metric="pricilla-missions">Missions: ...</div>
          <div class="live-sub" data-metric="pricilla-trajectory">Trajectories: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">Silenus</div>
          <div class="live-value" data-metric="silenus-health">...</div>
          <div class="live-sub" data-metric="silenus-frames">Frames: ...</div>
          <div class="live-sub" data-metric="silenus-alerts">Alerts: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">Hunoid</div>
          <div class="live-value" data-metric="hunoid-health">...</div>
          <div class="live-sub" data-metric="hunoid-actions">Actions: ...</div>
          <div class="live-sub" data-metric="hunoid-ethics">Ethics: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">Valkyrie</div>
          <div class="live-value" data-metric="valkyrie-health">...</div>
          <div class="live-sub" data-metric="valkyrie-mode">Mode: ...</div>
          <div class="live-sub" data-metric="valkyrie-metrics">Metrics: ...</div>
        </div>
        <div class="live-card">
          <div class="live-label">NATS</div>
          <div class="live-value" data-metric="nats-health">...</div>
        </div>
      </div>
    `;
    overlay.style.cssText = `
      position: fixed;
      bottom: 30px;
      left: 30px;
      padding: 18px 20px;
      background: rgba(10,10,26,0.92);
      border: 1px solid rgba(139,92,246,0.4);
      border-radius: 14px;
      color: white;
      z-index: 20000;
      min-width: 320px;
      box-shadow: 0 12px 40px rgba(0,0,0,0.45);
      font-family: 'Segoe UI', sans-serif;
    `;
    const style = document.createElement('style');
    style.textContent = `
      #live-metrics-overlay .live-title {
        font-size: 14px;
        font-weight: 700;
        letter-spacing: 1px;
        margin-bottom: 10px;
        text-transform: uppercase;
        color: rgba(255,255,255,0.7);
      }
      #live-metrics-overlay .live-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(140px, 1fr));
        gap: 12px;
      }
      #live-metrics-overlay .live-card {
        background: rgba(255,255,255,0.04);
        border-radius: 10px;
        padding: 10px 12px;
        border: 1px solid rgba(255,255,255,0.08);
      }
      #live-metrics-overlay .live-label {
        font-size: 12px;
        font-weight: 600;
        margin-bottom: 6px;
      }
      #live-metrics-overlay .live-value {
        font-size: 13px;
        font-weight: 600;
        color: #22c55e;
      }
      #live-metrics-overlay .live-sub {
        font-size: 11px;
        color: rgba(255,255,255,0.7);
        margin-top: 4px;
      }
    `;
    document.head.appendChild(style);
    document.body.appendChild(overlay);
  });
}

function startLiveMetricsPolling(page: Page, request: APIRequestContext) {
  let active = true;
  let inFlight = false;
  let timer: ReturnType<typeof setTimeout> | null = null;

  const poll = async () => {
    if (!active || inFlight) return;
    inFlight = true;
    try {
      const payload = await collectLiveMetrics(request);
      await updateLiveOverlay(page, payload);
    } finally {
      inFlight = false;
      if (active) {
        timer = setTimeout(poll, 2000);
      }
    }
  };

  void poll();
  return () => {
    active = false;
    if (timer) clearTimeout(timer);
  };
}

async function collectLiveMetrics(request: APIRequestContext) {
  const [nysusHealth, giruHealth, pricillaHealth, silenusHealth, hunoidHealth, valkyrieHealth, natsHealth] = await Promise.all([
    fetchJson(request, NYSUS_HEALTH_URL),
    fetchJson(request, GIRU_HEALTH_URL),
    fetchJson(request, PRICILLA_HEALTH_URL),
    fetchJson(request, SILENUS_HEALTH_URL),
    fetchJson(request, HUNOID_HEALTH_URL),
    fetchJson(request, VALKYRIE_HEALTH_URL),
    fetchText(request, NATS_HEALTH_URL),
  ]);

  const [nysusMetrics, giruMetrics, pricillaMetrics, silenusMetrics, hunoidMetrics, valkyrieMetrics, valkyrieStatus] = await Promise.all([
    fetchText(request, NYSUS_METRICS_URL),
    fetchText(request, GIRU_METRICS_URL),
    fetchText(request, PRICILLA_METRICS_URL),
    fetchText(request, SILENUS_METRICS_URL),
    fetchText(request, HUNOID_METRICS_URL),
    fetchText(request, VALKYRIE_METRICS_URL),
    fetchJson(request, VALKYRIE_STATUS_URL),
  ]);

  return {
    health: {
      nysus: nysusHealth?.status ?? 'error',
      giru: giruHealth?.status ?? 'error',
      pricilla: pricillaHealth?.status ?? 'error',
      silenus: silenusHealth?.status ?? 'error',
      hunoid: hunoidHealth?.status ?? 'error',
      valkyrie: valkyrieHealth?.status ?? 'error',
      nats: natsHealth?.includes('ok') ? 'ok' : 'error',
    },
    metrics: {
      nysus: {
        httpRequests: sumMetric(nysusMetrics, 'asgard_http_requests_total'),
        eventsProcessed: sumMetric(nysusMetrics, 'asgard_events_processed_total'),
        postgresConns: sumMetric(nysusMetrics, 'asgard_database_connections', { database: 'postgres', state: 'open' }),
      },
      giru: {
        threats: sumMetric(giruMetrics, 'asgard_security_threats_detected_total'),
        packets: sumMetric(giruMetrics, 'asgard_security_packets_scanned_total'),
      },
      pricilla: {
        missions: sumMetric(pricillaMetrics, 'asgard_pricilla_missions_total'),
        missionsActive: sumMetric(pricillaMetrics, 'asgard_pricilla_missions_active'),
        trajectories: sumMetric(pricillaMetrics, 'asgard_pricilla_trajectories_planned_total'),
      },
      silenus: {
        frames: sumMetric(silenusMetrics, 'asgard_satellite_frames_processed_total'),
        alerts: sumMetric(silenusMetrics, 'asgard_satellite_alerts_generated_total'),
      },
      hunoid: {
        actions: sumMetric(hunoidMetrics, 'asgard_hunoid_actions_executed_total'),
        ethics: sumMetric(hunoidMetrics, 'asgard_hunoid_ethics_rejections_total'),
      },
      valkyrie: {
        mode: typeof valkyrieStatus?.flight_mode === 'string' ? valkyrieStatus.flight_mode : 'unknown',
        metricsUp: Boolean(valkyrieMetrics && valkyrieMetrics.length > 0),
      },
    },
  };
}

async function updateLiveOverlay(page: Page, payload: ReturnType<typeof collectLiveMetrics> extends Promise<infer T> ? T : never) {
  const viewModel = {
    nysusHealth: `Status: ${payload.health.nysus}`,
    giruHealth: `Status: ${payload.health.giru}`,
    pricillaHealth: `Status: ${payload.health.pricilla}`,
    silenusHealth: `Status: ${payload.health.silenus}`,
    hunoidHealth: `Status: ${payload.health.hunoid}`,
    valkyrieHealth: `Status: ${payload.health.valkyrie}`,
    natsHealth: `Status: ${payload.health.nats}`,
    nysusHttp: `HTTP: ${formatMetric(payload.metrics.nysus.httpRequests)}`,
    nysusEvents: `Events: ${formatMetric(payload.metrics.nysus.eventsProcessed)}`,
    nysusDb: `DB: ${formatMetric(payload.metrics.nysus.postgresConns)}`,
    giruThreats: `Threats: ${formatMetric(payload.metrics.giru.threats)}`,
    giruPackets: `Packets: ${formatMetric(payload.metrics.giru.packets)}`,
    pricillaMissions: `Missions: ${formatMetric(payload.metrics.pricilla.missions)}`,
    pricillaTraj: `Traj: ${formatMetric(payload.metrics.pricilla.trajectories)}`,
    silenusFrames: `Frames: ${formatMetric(payload.metrics.silenus.frames)}`,
    silenusAlerts: `Alerts: ${formatMetric(payload.metrics.silenus.alerts)}`,
    hunoidActions: `Actions: ${formatMetric(payload.metrics.hunoid.actions)}`,
    hunoidEthics: `Ethics: ${formatMetric(payload.metrics.hunoid.ethics)}`,
    valkyrieMode: `Mode: ${payload.metrics.valkyrie.mode}`,
    valkyrieMetrics: `Metrics: ${payload.metrics.valkyrie.metricsUp ? 'OK' : 'N/A'}`,
  };

  await page.evaluate((data) => {
    const setText = (key: string, value: string) => {
      const el = document.querySelector(`[data-metric="${key}"]`);
      if (el) el.textContent = value;
    };

    setText('nysus-health', data.nysusHealth);
    setText('giru-health', data.giruHealth);
    setText('pricilla-health', data.pricillaHealth);
    setText('nats-health', data.natsHealth);

    setText('nysus-http', data.nysusHttp);
    setText('nysus-events', data.nysusEvents);
    setText('nysus-db', data.nysusDb);

    setText('giru-threats', data.giruThreats);
    setText('giru-packets', data.giruPackets);

    setText('pricilla-missions', data.pricillaMissions);
    setText('pricilla-trajectory', data.pricillaTraj);

    setText('silenus-health', data.silenusHealth);
    setText('silenus-frames', data.silenusFrames);
    setText('silenus-alerts', data.silenusAlerts);

    setText('hunoid-health', data.hunoidHealth);
    setText('hunoid-actions', data.hunoidActions);
    setText('hunoid-ethics', data.hunoidEthics);

    setText('valkyrie-health', data.valkyrieHealth);
    setText('valkyrie-mode', data.valkyrieMode);
    setText('valkyrie-metrics', data.valkyrieMetrics);
  }, viewModel);
}

function formatMetric(value: number | null) {
  if (value === null || Number.isNaN(value)) return 'N/A';
  if (value > 1000) return `${Math.round(value).toLocaleString()}`;
  return `${value}`;
}

async function fetchJson(request: APIRequestContext, url: string) {
  try {
    const response = await request.get(url, { timeout: 4000 });
    if (!response.ok()) return null;
    return await response.json();
  } catch {
    return null;
  }
}

async function fetchText(request: APIRequestContext, url: string) {
  try {
    const response = await request.get(url, { timeout: 4000 });
    if (!response.ok()) return '';
    return await response.text();
  } catch {
    return '';
  }
}

function sumMetric(source: string, name: string, labels?: Record<string, string>): number | null {
  if (!source) return null;
  const regex = new RegExp(`^${name}(\\{[^}]*\\})?\\s+([0-9eE+\\-.]+)$`, 'gm');
  let total = 0;
  let found = false;
  for (const match of source.matchAll(regex)) {
    const labelBlock = match[1];
    if (labels && labelBlock) {
      const parsed = parseLabelBlock(labelBlock);
      const matches = Object.entries(labels).every(([k, v]) => parsed[k] === v);
      if (!matches) continue;
    } else if (labels && !labelBlock) {
      continue;
    }
    const value = Number(match[2]);
    if (!Number.isNaN(value)) {
      total += value;
      found = true;
    }
  }
  return found ? Math.round(total * 1000) / 1000 : null;
}

function parseLabelBlock(block: string): Record<string, string> {
  const labels: Record<string, string> = {};
  const inner = block.replace(/^\{|\}$/g, '');
  if (!inner) return labels;
  inner.split(',').forEach((pair) => {
    const delimiterIndex = pair.indexOf('=');
    if (delimiterIndex <= 0) return;
    const key = pair.slice(0, delimiterIndex);
    const raw = pair.slice(delimiterIndex + 1);
    if (!key || !raw) return;
    labels[key.trim()] = raw.trim().replace(/^"|"$/g, '');
  });
  return labels;
}

async function fetchValkyrieSnapshot(request: APIRequestContext) {
  const [health, status, state, metrics] = await Promise.all([
    fetchJson(request, VALKYRIE_HEALTH_URL),
    fetchJson(request, VALKYRIE_STATUS_URL),
    fetchJson(request, VALKYRIE_STATE_URL),
    fetchText(request, VALKYRIE_METRICS_URL),
  ]);

  return {
    health: health?.status ?? 'error',
    status: status ?? {},
    state: state ?? {},
    metricsUp: Boolean(metrics && metrics.length > 0),
    metricsBytes: metrics ? metrics.length : 0,
  };
}

async function fetchHealthSnapshot(request: APIRequestContext) {
  const [nysus, giru, pricilla, silenus, hunoid, valkyrie, nats] = await Promise.all([
    fetchJson(request, NYSUS_HEALTH_URL),
    fetchJson(request, GIRU_HEALTH_URL),
    fetchJson(request, PRICILLA_HEALTH_URL),
    fetchJson(request, SILENUS_HEALTH_URL),
    fetchJson(request, HUNOID_HEALTH_URL),
    fetchJson(request, VALKYRIE_HEALTH_URL),
    fetchText(request, NATS_HEALTH_URL),
  ]);
  return {
    nysus: nysus?.status ?? 'error',
    giru: giru?.status ?? 'error',
    pricilla: pricilla?.status ?? 'error',
    silenus: silenus?.status ?? 'error',
    hunoid: hunoid?.status ?? 'error',
    valkyrie: valkyrie?.status ?? 'error',
    nats: nats?.includes('ok') ? 'ok' : 'error',
  };
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

async function triggerJarvisCommand(page: Page, command: string) {
  const toggle = page.locator('#toggle-text');
  if (await toggle.count()) {
    await toggle.click();
  }
  const input = page.locator('#text-input');
  if (await input.count()) {
    await input.fill(command);
    await input.press('Enter');
  }
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

function getValkyrieStatusPage(snapshot: {
  health: string;
  status: Record<string, any>;
  state: Record<string, any>;
  metricsUp: boolean;
  metricsBytes: number;
}): string {
  const status = snapshot.status ?? {};
  const state = snapshot.state ?? {};
  const position = state.position ?? {};
  const velocity = state.velocity ?? {};

  const flightMode = typeof status.flight_mode === 'string' ? status.flight_mode : 'unknown';
  const simMode = status.simulation_mode ? 'sim' : 'live';
  const armed = status.armed ? 'armed' : 'disarmed';
  const mavlink = status.mavlink_connected ? 'connected' : 'offline';
  const aiActive = status.ai_active ? 'active' : 'inactive';
  const securityActive = status.security_active ? 'active' : 'inactive';
  const failsafeActive = status.failsafe_active ? 'active' : 'inactive';
  const fusionActive = status.fusion_active ? 'active' : 'inactive';
  const confidence = typeof status.confidence === 'number' ? status.confidence.toFixed(2) : 'N/A';
  const metrics = snapshot.metricsUp ? `OK (${snapshot.metricsBytes} bytes)` : 'unavailable';

  return `
<!DOCTYPE html>
<html>
  <head>
    <style>
      body {
        margin: 0;
        min-height: 100vh;
        background: linear-gradient(135deg, #0a0a1a, #101824);
        color: white;
        font-family: 'Segoe UI', sans-serif;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .panel {
        padding: 34px 38px;
        border-radius: 18px;
        background: rgba(12, 15, 30, 0.95);
        border: 1px solid rgba(16, 185, 129, 0.3);
        width: 760px;
      }
      .title { font-size: 24px; font-weight: 700; margin-bottom: 16px; }
      .grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
      .row { display: flex; justify-content: space-between; font-size: 13px; padding: 10px 12px; background: rgba(255,255,255,0.04); border-radius: 10px; border: 1px solid rgba(16,185,129,0.2); }
      .label { color: rgba(255,255,255,0.7); }
      .value { color: #34d399; font-weight: 600; }
      .sub { margin-top: 10px; font-size: 12px; color: rgba(255,255,255,0.6); }
    </style>
  </head>
  <body>
    <div class="panel">
      <div class="title">Valkyrie Live Status</div>
      <div class="grid">
        <div class="row"><span class="label">Health</span><span class="value">${snapshot.health}</span></div>
        <div class="row"><span class="label">Mode</span><span class="value">${flightMode}</span></div>
        <div class="row"><span class="label">Simulation</span><span class="value">${simMode}</span></div>
        <div class="row"><span class="label">Armed</span><span class="value">${armed}</span></div>
        <div class="row"><span class="label">MAVLink</span><span class="value">${mavlink}</span></div>
        <div class="row"><span class="label">Fusion</span><span class="value">${fusionActive}</span></div>
        <div class="row"><span class="label">AI</span><span class="value">${aiActive}</span></div>
        <div class="row"><span class="label">Security</span><span class="value">${securityActive}</span></div>
        <div class="row"><span class="label">Failsafe</span><span class="value">${failsafeActive}</span></div>
        <div class="row"><span class="label">Confidence</span><span class="value">${confidence}</span></div>
        <div class="row"><span class="label">Metrics</span><span class="value">${metrics}</span></div>
        <div class="row"><span class="label">Position</span><span class="value">${position.x ?? 0}, ${position.y ?? 0}, ${position.z ?? 0}</span></div>
        <div class="row"><span class="label">Velocity</span><span class="value">${velocity.x ?? 0}, ${velocity.y ?? 0}, ${velocity.z ?? 0}</span></div>
      </div>
      <div class="sub">Ports: API 8093 | Metrics 9095 (override via ASGARD_VALKYRIE_METRICS)</div>
    </div>
  </body>
</html>`;
}

function getControlHubPage(health: { nysus: string; giru: string; pricilla: string; silenus: string; hunoid: string; valkyrie: string; nats: string }): string {
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
        <div class="item"><span>Nysus API</span><span class="status">${health.nysus}</span></div>
        <div class="item"><span>Giru</span><span class="status">${health.giru}</span></div>
        <div class="item"><span>Pricilla</span><span class="status">${health.pricilla}</span></div>
        <div class="item"><span>NATS</span><span class="status">${health.nats}</span></div>
        <div class="item"><span>Silenus</span><span class="status">${health.silenus}</span></div>
        <div class="item"><span>Hunoid</span><span class="status">${health.hunoid}</span></div>
        <div class="item"><span>Valkyrie</span><span class="status">${health.valkyrie}</span></div>
      </div>
    </div>
  </body>
</html>`;
}

function getIntegrationPage(health: { nysus: string; giru: string; pricilla: string; silenus: string; hunoid: string; valkyrie: string; nats: string }): string {
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
      <div class="row"><span>Nysus</span><span class="ok">${health.nysus}</span></div>
      <div class="row"><span>Giru</span><span class="ok">${health.giru}</span></div>
      <div class="row"><span>Pricilla</span><span class="ok">${health.pricilla}</span></div>
      <div class="row"><span>Silenus</span><span class="ok">${health.silenus}</span></div>
      <div class="row"><span>Hunoid</span><span class="ok">${health.hunoid}</span></div>
      <div class="row"><span>Valkyrie</span><span class="ok">${health.valkyrie}</span></div>
      <div class="row"><span>NATS</span><span class="ok">${health.nats}</span></div>
    </div>
  </body>
</html>`;
}
