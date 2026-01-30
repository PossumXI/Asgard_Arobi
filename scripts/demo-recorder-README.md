# ASGARD Demo Recorder

Comprehensive Playwright-based demo recorder for capturing video demonstrations of all ASGARD systems.

## Features

- **Video Recording**: Captures full video of browser interactions
- **Screenshot Capture**: Takes screenshots at key moments
- **Manifest Generation**: Creates a JSON manifest of all recordings
- **Multi-System Support**: Records demos for Website, Valkyrie, Pricilla, and Giru JARVIS
- **Service Health Checks**: Automatically skips unavailable services
- **Flexible Recording Options**: Record all systems or specific ones

## Prerequisites

- Node.js 18.0.0 or higher
- npm or yarn
- Running ASGARD services (locally)

## Installation

```powershell
# Navigate to scripts directory
cd scripts

# Install dependencies
npm install

# Install Playwright browsers
npm run install:playwright
```

## Usage

### Record All Systems (Default)

```powershell
npm run demo
# or
npx ts-node demo-recorder.ts
```

### Record Specific Systems

```powershell
# Website only
npm run demo:website

# Valkyrie only
npm run demo:valkyrie

# Pricilla only
npm run demo:pricilla

# Giru JARVIS only
npm run demo:giru
```

### Command Line Options

```powershell
npx ts-node demo-recorder.ts [options]

Options:
  --all           Record all demos (default)
  --website-only  Record only website demo
  --valkyrie-only Record only Valkyrie demo
  --pricilla-only Record only Pricilla demo
  --giru-only     Record only Giru JARVIS demo
  --headless      Run in headless mode (no visible browser)
  --slow          Add delays between actions for better visibility
```

### Examples

```powershell
# Full demo with slow mode for presentations
npx ts-node demo-recorder.ts --all --slow

# Quick headless recording of Valkyrie
npx ts-node demo-recorder.ts --valkyrie-only --headless

# Record website and Pricilla with visible browser
npx ts-node demo-recorder.ts --website-only
npx ts-node demo-recorder.ts --pricilla-only
```

## Service Ports

| System | URL | Description |
|--------|-----|-------------|
| Website | http://localhost:3000 | Main ASGARD marketing website |
| Valkyrie | http://localhost:8093 | Autonomous vehicle system |
| Pricilla | http://localhost:8092 | Ad targeting system |
| Giru JARVIS | http://localhost:5000 | AI assistant |

## Output Structure

```
scripts/
└── demo-output/
    ├── manifest.json           # Recording manifest with metadata
    ├── screenshots/            # PNG screenshots
    │   ├── website-landing.png
    │   ├── website-valkyrie.png
    │   ├── valkyrie-health.png
    │   └── ...
    └── videos/                 # WebM video recordings
        ├── website/
        ├── valkyrie/
        ├── pricilla/
        └── giru/
```

## Manifest Format

The `manifest.json` contains metadata about all recordings:

```json
{
  "version": "1.0.0",
  "generatedAt": "2026-01-30T10:00:00.000Z",
  "totalDuration": 120.5,
  "recordings": [
    {
      "id": "screenshot-website-landing-2026-01-30T10-00-00-000Z",
      "name": "website-landing",
      "description": "ASGARD Landing Page",
      "type": "screenshot",
      "path": "screenshots/2026-01-30T10-00-00-000Z-website-landing.png",
      "timestamp": "2026-01-30T10:00:00.000Z",
      "system": "Website",
      "url": "http://localhost:3000"
    }
  ],
  "systems": [
    { "name": "Website", "status": "recorded" },
    { "name": "Valkyrie", "status": "recorded" },
    { "name": "Pricilla", "status": "skipped", "error": "Service not available" },
    { "name": "Giru JARVIS", "status": "recorded" }
  ]
}
```

## Demo Sequences

### Website Demo
1. Landing page showcase
2. Navigate to Valkyrie product page
3. Navigate to Pricilla product page
4. Navigate to Giru product page
5. Navigate to Contact page
6. Show Pricing page
7. Demonstrate Sign-up flow

### Valkyrie Demo
1. Health check endpoint (`/health`)
2. Status endpoint (`/status`)
3. State endpoint (`/state`)
4. Sensors API (`/api/v1/sensors`)
5. Navigation API (`/api/v1/navigation`)

### Pricilla Demo
1. Health check (`/health`)
2. Status endpoint (`/status`)
3. Targeting API (`/api/v1/targeting`)
4. Metrics endpoint (`/metrics`)

### Giru JARVIS Demo
1. Main JARVIS UI interface
2. Monitor dashboard
3. Status page
4. API health endpoint
5. Voice assistant interface

## Troubleshooting

### Service Not Available
If a service is not running, the recorder will automatically skip it and note the status in the manifest.

### Playwright Browser Issues
```powershell
# Reinstall browsers
npx playwright install chromium --force
```

### Permission Issues
```powershell
# Run with elevated permissions on Windows
Start-Process powershell -Verb runAs -ArgumentList "cd scripts; npm run demo"
```

## Clean Output

Remove all recordings:

```powershell
npm run clean
# or
Remove-Item -Recurse -Force demo-output
```

## Integration with CI/CD

For automated demo generation in CI/CD:

```yaml
- name: Generate Demo
  run: |
    cd scripts
    npm ci
    npx playwright install chromium
    npm run demo -- --headless
  
- name: Upload Artifacts
  uses: actions/upload-artifact@v3
  with:
    name: demo-recordings
    path: scripts/demo-output/
```

## License

Part of ASGARD Systems - Internal Use
