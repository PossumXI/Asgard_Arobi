import { test, Page } from '@playwright/test';
import { randomUUID } from 'crypto';

/**
 * PRICILLA Demo Video
 *
 * Focused showcase of Pricilla deployment and guidance accuracy.
 */
test.describe('Pricilla Demo', () => {
  test('Pricilla Guidance Showcase', async ({ page, request }) => {
    test.setTimeout(180000);

    console.log('🎯 PRICILLA DEMO');

    await page.goto('http://localhost:3000/pricilla');
    await page.waitForLoadState('networkidle');

    await showTitle(page, '🎯 PRICILLA', 'Precision Payload Guidance');
    await pause(2000);

    await showCallout(page, 'Deployment Ready', 'Mission package sealed and cleared for launch.');
    await pause(2200);

    await smoothScrollBy(page, 600);
    await showCallout(page, 'Multi-Agent AI', 'MARL + physics models select optimal trajectories.');
    await pause(2200);

    await smoothScrollBy(page, 600);
    await showCallout(page, 'Through-Wall WiFi Imaging', 'CSI imaging reveals occluded targets.');
    await pause(2400);

    await smoothScrollBy(page, 600);
    await showCallout(page, 'Sensor Fusion', 'Radar + lidar + visual + WiFi fused into one state.');
    await pause(2400);

    await smoothScrollBy(page, 600);
    await showCallout(page, 'Split-Second Replanning', 'Telemetry shifts trigger sub-second replan loops.');
    await pause(1200);

    await runTargetingScenario(request);
    await showMetricsOverlay(page, request, 'Targeting Metrics', 'Live replans + target updates');
    await pause(4800);

    await smoothScrollBy(page, 600);
    await showCallout(page, 'Accuracy Evidence', 'Validated with EKF confidence and physics constraints.');
    await pause(2400);

    await smoothScrollToBottom(page);
    await pause(2000);

    await showTitle(page, '✅ MISSION COMPLETE', 'Pricilla delivers with confidence.');
    await pause(2500);

    console.log('🎬 PRICILLA DEMO COMPLETE');
  });
});

async function smoothScrollToBottom(page: Page) {
  await page.evaluate(async () => {
    const scrollHeight = document.documentElement.scrollHeight;
    const viewportHeight = window.innerHeight;
    const scrollSteps = Math.ceil((scrollHeight - viewportHeight) / 240);

    for (let i = 0; i < scrollSteps; i++) {
      window.scrollBy({ top: 240, behavior: 'smooth' });
      await new Promise(r => setTimeout(r, 60));
    }
  });
}

async function smoothScrollBy(page: Page, amount: number) {
  await page.evaluate(async (distance) => {
    const steps = Math.max(1, Math.ceil(distance / 200));
    for (let i = 0; i < steps; i++) {
      window.scrollBy({ top: distance / steps, behavior: 'smooth' });
      await new Promise(r => setTimeout(r, 60));
    }
  }, amount);
}

async function pause(ms: number) {
  await new Promise(resolve => setTimeout(resolve, ms));
}

async function showTitle(page: Page, title: string, subtitle: string) {
  console.log(`\n🎬 ${title}: ${subtitle}\n`);
  await page.evaluate(({ title, subtitle }) => {
    const existing = document.getElementById('pricilla-demo-overlay');
    if (existing) existing.remove();

    const overlay = document.createElement('div');
    overlay.id = 'pricilla-demo-overlay';
    overlay.innerHTML = `
      <div class="pricilla-title">${title}</div>
      <div class="pricilla-subtitle">${subtitle}</div>
    `;

    overlay.style.cssText = `
      position: fixed;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      text-align: center;
      z-index: 999999;
      pointer-events: none;
      animation: pricillaFadeIn 0.5s ease-out, pricillaFadeOut 0.5s ease-in 2.5s forwards;
    `;

    const style = document.createElement('style');
    style.id = 'pricilla-demo-styles';
    style.textContent = `
      @keyframes pricillaFadeIn {
        0% { opacity: 0; transform: translate(-50%, -50%) scale(0.95); }
        100% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
      }
      @keyframes pricillaFadeOut {
        0% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
        100% { opacity: 0; transform: translate(-50%, -50%) scale(1.05); }
      }
      .pricilla-title {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 48px;
        font-weight: 700;
        color: white;
        text-shadow: 0 4px 30px rgba(0, 0, 0, 0.8), 0 0 60px rgba(10, 132, 255, 0.5);
        letter-spacing: 3px;
        margin-bottom: 16px;
      }
      .pricilla-subtitle {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 22px;
        font-weight: 400;
        color: rgba(255, 255, 255, 0.9);
        text-shadow: 0 2px 20px rgba(0, 0, 0, 0.6);
        letter-spacing: 1.5px;
      }
    `;

    document.getElementById('pricilla-demo-styles')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(overlay);

    setTimeout(() => {
      overlay.remove();
    }, 3200);
  }, { title, subtitle });
}

async function showCallout(page: Page, title: string, body: string) {
  await page.evaluate(({ title, body }) => {
    const existing = document.getElementById('pricilla-demo-callout');
    if (existing) existing.remove();

    const callout = document.createElement('div');
    callout.id = 'pricilla-demo-callout';
    callout.innerHTML = `
      <div class="pricilla-callout-title">${title}</div>
      <div class="pricilla-callout-body">${body}</div>
    `;

    callout.style.cssText = `
      position: fixed;
      bottom: 60px;
      right: 60px;
      max-width: 420px;
      padding: 18px 22px;
      border-radius: 16px;
      background: rgba(15, 23, 42, 0.85);
      backdrop-filter: blur(12px);
      color: white;
      z-index: 999999;
      box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
      border: 1px solid rgba(148, 163, 184, 0.3);
      animation: pricillaSlideIn 0.4s ease-out, pricillaSlideOut 0.4s ease-in 2.6s forwards;
    `;

    const style = document.createElement('style');
    style.id = 'pricilla-demo-callout-style';
    style.textContent = `
      @keyframes pricillaSlideIn {
        0% { opacity: 0; transform: translateY(20px); }
        100% { opacity: 1; transform: translateY(0); }
      }
      @keyframes pricillaSlideOut {
        0% { opacity: 1; transform: translateY(0); }
        100% { opacity: 0; transform: translateY(20px); }
      }
      .pricilla-callout-title {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 6px;
      }
      .pricilla-callout-body {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', sans-serif;
        font-size: 14px;
        line-height: 1.4;
        color: rgba(226, 232, 240, 0.95);
      }
    `;

    document.getElementById('pricilla-demo-callout-style')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(callout);

    setTimeout(() => {
      callout.remove();
    }, 3000);
  }, { title, body });
}

async function runTargetingScenario(request: any) {
  const baseUrl = 'http://localhost:8092';
  const missionId = randomUUID();
  const payloadId = 'payload-demo-001';

  const start = { x: 0, y: 0, z: 300 };
  let target = { x: 1500, y: 1000, z: 50 };

  await request.post(`${baseUrl}/api/v1/missions`, {
    data: {
      id: missionId,
      type: 'strike',
      payloadId,
      payloadType: 'uav',
      startPosition: start,
      targetPosition: target,
      priority: 1,
      stealthRequired: true
    }
  });

  await request.post(`${baseUrl}/api/v1/payloads`, {
    data: {
      id: payloadId,
      type: 'uav',
      position: start,
      velocity: { x: 80, y: 40, z: 0 },
      heading: 0,
      fuel: 95,
      battery: 98,
      health: 99,
      status: 'navigating'
    }
  });

  await request.post(`${baseUrl}/api/v1/wifi/routers`, {
    data: {
      id: 'router-metrics-01',
      position: { x: 400, y: 200, z: 5 },
      frequencyGhz: 5.8,
      txPowerDbm: 18
    }
  });

  await request.post(`${baseUrl}/api/v1/wifi/imaging`, {
    data: {
      routerId: 'router-metrics-01',
      receiverId: payloadId,
      pathLossDb: 65,
      multipathSpread: 5,
      confidence: 0.9
    }
  });

  for (let i = 1; i <= 4; i += 1) {
    target = { x: 1500 + i * 120, y: 1000 + i * 80, z: 50 };
    await request.post(`${baseUrl}/api/v1/missions/target/${missionId}`, { data: target });
    await request.put(`${baseUrl}/api/v1/payloads/${payloadId}`, {
      data: {
        id: payloadId,
        type: 'uav',
        position: { x: start.x + i * 200, y: start.y + i * 130, z: 280 },
        velocity: { x: 80, y: 40, z: 0 },
        heading: 0,
        fuel: 95,
        battery: 98,
        health: 99,
        status: 'navigating'
      }
    });
  }

  await request.put(`${baseUrl}/api/v1/payloads/${payloadId}`, {
    data: {
      id: payloadId,
      type: 'uav',
      position: target,
      velocity: { x: 20, y: 10, z: 0 },
      heading: 0,
      fuel: 92,
      battery: 96,
      health: 99,
      status: 'completed'
    }
  });
}

async function showMetricsOverlay(page: Page, request: any, title: string, subtitle: string) {
  const response = await request.get('http://localhost:8092/api/v1/metrics/targeting');
  const metrics = await response.json();

  await page.evaluate(({ title, subtitle, metrics }) => {
    const existing = document.getElementById('pricilla-metrics-overlay');
    if (existing) existing.remove();

    const overlay = document.createElement('div');
    overlay.id = 'pricilla-metrics-overlay';
    const formatVec = (vec: { x: number; y: number; z: number } | undefined) => {
      if (!vec) return 'n/a';
      return `${vec.x?.toFixed?.(1) ?? vec.x}, ${vec.y?.toFixed?.(1) ?? vec.y}, ${vec.z?.toFixed?.(1) ?? vec.z}`;
    };

    overlay.innerHTML = `
      <div class="metrics-title">${title}</div>
      <div class="metrics-subtitle">${subtitle}</div>
      <div class="metrics-grid">
        <div><strong>Target Updates:</strong> ${metrics.targetUpdates}</div>
        <div><strong>Replans:</strong> ${metrics.replanCount}</div>
        <div><strong>Last Reason:</strong> ${metrics.lastReplanReason || 'n/a'}</div>
        <div><strong>Trajectory:</strong> ${metrics.lastTrajectoryId || 'n/a'}</div>
        <div><strong>Mission:</strong> ${metrics.lastMissionId || 'n/a'}</div>
        <div><strong>Target Pos:</strong> ${formatVec(metrics.lastTargetPosition)}</div>
        <div><strong>Payload Pos:</strong> ${formatVec(metrics.lastPayloadPosition)}</div>
        <div><strong>Completion (m):</strong> ${metrics.completionDistance?.toFixed?.(2) ?? metrics.completionDistance}</div>
      </div>
    `;

    overlay.style.cssText = `
      position: fixed;
      top: 80px;
      right: 60px;
      min-width: 320px;
      padding: 18px 22px;
      border-radius: 16px;
      background: rgba(15, 23, 42, 0.88);
      backdrop-filter: blur(12px);
      color: white;
      z-index: 999999;
      box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
      border: 1px solid rgba(148, 163, 184, 0.3);
      animation: pricillaSlideIn 0.4s ease-out, pricillaSlideOut 0.4s ease-in 5.0s forwards;
    `;

    const style = document.createElement('style');
    style.id = 'pricilla-metrics-style';
    style.textContent = `
      .metrics-title {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 4px;
      }
      .metrics-subtitle {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', sans-serif;
        font-size: 13px;
        margin-bottom: 12px;
        color: rgba(226, 232, 240, 0.85);
      }
      .metrics-grid {
        display: grid;
        gap: 6px;
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', sans-serif;
        font-size: 13px;
        color: rgba(226, 232, 240, 0.95);
      }
    `;

    document.getElementById('pricilla-metrics-style')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(overlay);

    setTimeout(() => {
      overlay.remove();
    }, 5400);
  }, { title, subtitle, metrics });
}
