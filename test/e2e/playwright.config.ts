import { defineConfig, devices } from '@playwright/test';

/**
 * ASGARD Demo Video Configuration
 * Records comprehensive demo videos of the ASGARD platform
 */
export default defineConfig({
  testDir: '.',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: 'html',
  
  // Global timeout for comprehensive demos (10 minutes)
  timeout: 600000,
  
  use: {
    // Base URL for the Websites app
    baseURL: 'http://localhost:3000',
    
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
    
    // Optimized timing for 60-second cinematic demo
    launchOptions: {
      slowMo: 300, // Smooth but efficient transitions
    },
  },

  // Output directory for videos
  outputDir: './demo-videos',

  projects: [
    {
      name: 'demo-chromium',
      use: { 
        ...devices['Desktop Chrome'],
        // Override for high-quality video
        video: {
          mode: 'on',
          size: { width: 1920, height: 1080 }
        },
      },
    },
  ],

  // Web server configuration - starts the apps automatically
  webServer: [
    {
      command: 'npm run dev',
      cwd: '../../Websites',
      url: 'http://localhost:3000',
      reuseExistingServer: true,
      timeout: 60000,
    },
    {
      command: 'npm run dev',
      cwd: '../../Hubs',
      url: 'http://localhost:3001',
      reuseExistingServer: true,
      timeout: 60000,
    },
  ],
});
