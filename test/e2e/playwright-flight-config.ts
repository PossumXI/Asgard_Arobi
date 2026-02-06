import { defineConfig, devices } from '@playwright/test';

/**
 * ASGARD Flight Simulation Demo Configuration
 * Isolated configuration for running only the flight simulation demo
 */
export default defineConfig({
  testDir: './',
  fullyParallel: false,
  forbidOnly: false, // Allow .only for debugging
  retries: 0,
  workers: 1,
  reporter: 'html',
  
  // Global timeout for comprehensive demo (10 minutes)
  timeout: 600000,
  
  use: {
    // Capture trace on failure
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'on',

    // VIDEO RECORDING - This is what creates the demo video!
    video: {
      mode: 'on',
      size: { width: 1920, height: 1080 }
    },

    // Viewport settings for nice recording
    viewport: { width: 1920, height: 1080 },

    // Smooth transitions for cinematic demo
    launchOptions: {
      slowMo: 300,
    },
  },

  // Output directory for videos
  outputDir: './demo-videos',

  projects: [
    {
      name: 'flight-demo-chromium',
      testMatch: 'flight-simulation-demo.spec.ts',
      use: {
        ...devices['Desktop Chrome'],
        video: {
          mode: 'on',
          size: { width: 1920, height: 1080 }
        },
      },
    },
  ],

  // No webServer needed - the test uses page.setContent() for the dashboard
  // and starts Go backend services directly via pre-built binaries
});