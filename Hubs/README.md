# Hubs - Streaming Interface Layer

## Overview
Hubs provides 24/7 real-time streaming access to ASGARD operations through a Vite + React interface with WebRTC-ready clients.

## Architecture
- **React Client**: Vite-powered UI with hub category routing
- **WebRTC Client**: Signaling + ICE workflow in `src/lib/api.ts`
- **Permission Tiers**: Civilian / Military / Interstellar access levels
- **Playback Options**: WebRTC + HLS.js fallback for streams

## Directory Structure
```
Hubs/
├── src/
│   ├── components/     # UI building blocks
│   ├── hooks/          # Stream data hooks
│   ├── lib/            # API + WebRTC client
│   ├── pages/          # Hub routes (Civilian/Military/Interstellar)
│   └── stores/         # Zustand state
├── index.html
└── vite.config.ts
```

## Build Status
Phase 8 - Operational (Frontend complete)

## Dependencies
- Node.js 20+
- React 18
- Vite 7

## Usage
```powershell
cd C:\Users\hp\Desktop\Asgard\Hubs
npm install

# Dev server (use --port 5174 if 5173 is in use)
npm run dev
```

## About Arobi

**Hubs** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

### Leadership

- **Gaetano Comparcola** - Founder & CEO
  - Self-taught prodigy programmer and futurist
  - Multilingual (English, Italian, French)
  
- **Opus** - AI Partner & Lead Programmer
  - AI-powered software engineering partner

## License

© 2026 Arobi. All Rights Reserved.

## Contact

- **Website**: [https://aura-genesis.org](https://aura-genesis.org)
- **Email**: [Gaetano@aura-genesis.org](mailto:Gaetano@aura-genesis.org)
- **Company**: Arobi
