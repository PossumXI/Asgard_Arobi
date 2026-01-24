import { test, expect, Page } from '@playwright/test';

/**
 * ASGARD Platform - Professional Demo Video
 * 
 * Duration: ~60 seconds
 * Resolution: 1920x1080 (Full HD)
 * 
 * A cinematic walkthrough of the ASGARD Autonomous Space Defense Platform
 */

test.describe('ASGARD Professional Demo', () => {
  
  test('60-Second Platform Showcase', async ({ page }) => {
    test.setTimeout(180000);
    
    // ═══════════════════════════════════════════════════════════════════
    // ACT 1: THE VISION - ASGARD Marketing Website (Port 3000)
    // ═══════════════════════════════════════════════════════════════════
    
    console.log('🚀 ACT 1: THE VISION');
    
    // Scene 1: Hero Landing - First Impression
    await page.goto('http://localhost:3000');
    await page.waitForLoadState('networkidle');
    await showTitle(page, '🛡️ ASGARD', 'Autonomous Space Defense Platform');
    await pause(2500);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 2: Features - Capabilities Showcase
    console.log('📍 Showcasing Platform Capabilities...');
    await navigateTo(page, 'http://localhost:3000/features', 'Features');
    await showTitle(page, '⚡ CAPABILITIES', 'AI-Powered Space Intelligence');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 3: About - The Mission
    console.log('📍 Revealing The Mission...');
    await navigateTo(page, 'http://localhost:3000/about', 'About');
    await showTitle(page, '🎯 THE MISSION', 'Protecting Earth from Space');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 4: Pricing - Access Tiers
    console.log('📍 Displaying Access Tiers...');
    await navigateTo(page, 'http://localhost:3000/pricing', 'Pricing');
    await showTitle(page, '💎 ACCESS TIERS', 'From Civilian to Commander');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 5: Sign In - Secure Authentication
    console.log('📍 Demonstrating Secure Access...');
    await navigateTo(page, 'http://localhost:3000/signin', 'Sign In');
    await showTitle(page, '🔐 SECURE ACCESS', 'Military-Grade Authentication');
    await pause(1500);
    
    // Fill demo credentials for visual effect
    await fillDemoCredentials(page);
    await pause(2000);
    
    // ═══════════════════════════════════════════════════════════════════
    // ACT 2: THE OPERATIONS - ASGARD Hubs (Port 3001)
    // ═══════════════════════════════════════════════════════════════════
    
    console.log('🌍 ACT 2: THE OPERATIONS');
    
    // Scene 6: Hubs Home - Stream Discovery
    await page.goto('http://localhost:3001');
    await page.waitForLoadState('networkidle');
    await showTitle(page, '📡 LIVE OPERATIONS', 'Real-Time Global Intelligence');
    await pause(2500);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 7: Civilian Hub - Humanitarian Operations
    console.log('📍 Civilian Intelligence Hub...');
    await navigateTo(page, 'http://localhost:3001/civilian', 'Civilian');
    await showTitle(page, '🌐 CIVILIAN HUB', 'Humanitarian Satellite Feeds');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 8: Military Hub - Tactical Operations
    console.log('📍 Military Operations Hub...');
    await navigateTo(page, 'http://localhost:3001/military', 'Military');
    await showTitle(page, '🎖️ MILITARY HUB', 'Tactical Defense Networks');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 9: Interstellar Hub - Beyond Earth
    console.log('📍 Interstellar Operations Hub...');
    await navigateTo(page, 'http://localhost:3001/interstellar', 'Interstellar');
    await showTitle(page, '🚀 INTERSTELLAR HUB', 'Beyond Earth Operations');
    await pause(2000);
    await smoothScrollToBottom(page);
    await pause(1500);
    
    // Scene 10: Live Stream - Real-Time Intelligence
    console.log('📍 Accessing Live Stream...');
    const streamCard = page.locator('[class*="stream"], [class*="card"], a[href*="stream"]').first();
    if (await streamCard.isVisible({ timeout: 2000 }).catch(() => false)) {
      await streamCard.click();
      await page.waitForLoadState('networkidle');
      await showTitle(page, '📺 LIVE FEED', 'Real-Time Satellite Intelligence');
      await pause(3000);
    }
    
    // ═══════════════════════════════════════════════════════════════════
    // ACT 3: THE FUTURE - Port 3002 (if available)
    // ═══════════════════════════════════════════════════════════════════
    
    console.log('🔮 ACT 3: THE FUTURE');
    
    // Check if port 3002 has content
    try {
      const response = await page.goto('http://localhost:3002', { timeout: 5000 });
      if (response?.ok()) {
        await page.waitForLoadState('networkidle');
        await showTitle(page, '🔮 ADVANCED SYSTEMS', 'Next-Generation Defense');
        await pause(2500);
        await smoothScrollToBottom(page);
        await pause(1500);
      }
    } catch {
      console.log('Port 3002 not available, continuing...');
    }
    
    // ═══════════════════════════════════════════════════════════════════
    // FINALE: THE PROMISE
    // ═══════════════════════════════════════════════════════════════════
    
    console.log('✨ FINALE');
    
    // Return to landing for dramatic finish
    await page.goto('http://localhost:3000');
    await page.waitForLoadState('networkidle');
    await showTitle(page, '🛡️ ASGARD', 'The Future of Space Defense');
    await pause(3000);
    
    console.log('🎬 DEMO COMPLETE');
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Navigate to a URL with fallback
 */
async function navigateTo(page: Page, url: string, linkText: string) {
  try {
    // Try clicking navigation link first
    const link = page.locator(`a:has-text("${linkText}"), button:has-text("${linkText}")`).first();
    if (await link.isVisible({ timeout: 1000 })) {
      await link.click();
      await page.waitForLoadState('networkidle');
      return;
    }
  } catch {}
  // Fallback to direct navigation
  await page.goto(url);
  await page.waitForLoadState('networkidle');
}

/**
 * Smooth scroll to bottom of page
 */
async function smoothScrollToBottom(page: Page) {
  await page.evaluate(async () => {
    const scrollHeight = document.documentElement.scrollHeight;
    const viewportHeight = window.innerHeight;
    const scrollSteps = Math.ceil((scrollHeight - viewportHeight) / 200);
    
    for (let i = 0; i < scrollSteps; i++) {
      window.scrollBy({ top: 200, behavior: 'smooth' });
      await new Promise(r => setTimeout(r, 50));
    }
  });
}

/**
 * Pause execution
 */
async function pause(ms: number) {
  await new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Fill demo credentials with visual effect
 */
async function fillDemoCredentials(page: Page) {
  const emailInput = page.locator('input[type="email"], input[name="email"]').first();
  const passwordInput = page.locator('input[type="password"], input[name="password"]').first();
  
  if (await emailInput.isVisible({ timeout: 2000 }).catch(() => false)) {
    await emailInput.click();
    await emailInput.type('commander@asgard.space', { delay: 50 });
  }
  
  if (await passwordInput.isVisible({ timeout: 2000 }).catch(() => false)) {
    await passwordInput.click();
    await passwordInput.type('SecureAccess2026!', { delay: 30 });
  }
}

/**
 * Show cinematic title overlay
 */
async function showTitle(page: Page, title: string, subtitle: string) {
  console.log(`\n🎬 ${title}: ${subtitle}\n`);
  
  await page.evaluate(({ title, subtitle }) => {
    // Remove existing overlay
    const existing = document.getElementById('asgard-demo-overlay');
    if (existing) existing.remove();
    
    // Create cinematic overlay
    const overlay = document.createElement('div');
    overlay.id = 'asgard-demo-overlay';
    overlay.innerHTML = `
      <div class="asgard-title">${title}</div>
      <div class="asgard-subtitle">${subtitle}</div>
    `;
    
    overlay.style.cssText = `
      position: fixed;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      text-align: center;
      z-index: 999999;
      pointer-events: none;
      animation: asgardFadeIn 0.5s ease-out, asgardFadeOut 0.5s ease-in 2.5s forwards;
    `;
    
    // Add styles
    const style = document.createElement('style');
    style.id = 'asgard-demo-styles';
    style.textContent = `
      @keyframes asgardFadeIn {
        0% { opacity: 0; transform: translate(-50%, -50%) scale(0.9); }
        100% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
      }
      @keyframes asgardFadeOut {
        0% { opacity: 1; transform: translate(-50%, -50%) scale(1); }
        100% { opacity: 0; transform: translate(-50%, -50%) scale(1.1); }
      }
      .asgard-title {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 48px;
        font-weight: 700;
        color: white;
        text-shadow: 0 4px 30px rgba(0, 0, 0, 0.8), 0 0 60px rgba(10, 132, 255, 0.5);
        letter-spacing: 4px;
        margin-bottom: 16px;
      }
      .asgard-subtitle {
        font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
        font-size: 24px;
        font-weight: 400;
        color: rgba(255, 255, 255, 0.9);
        text-shadow: 0 2px 20px rgba(0, 0, 0, 0.6);
        letter-spacing: 2px;
      }
    `;
    
    // Remove old styles if present
    document.getElementById('asgard-demo-styles')?.remove();
    document.head.appendChild(style);
    document.body.appendChild(overlay);
    
    // Auto-remove after animation
    setTimeout(() => {
      overlay.remove();
    }, 3500);
  }, { title, subtitle });
}
