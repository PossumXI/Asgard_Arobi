# Hubs - Streaming Interface Layer

## Overview
Hubs provides 24/7 real-time streaming access to ASGARD operations through WebRTC-powered interfaces.

## Architecture
- **Signaling Server**: Go-based WebRTC signaling
- **Stream Bridge**: Converts Sat_Net feeds to WebRTC
- **Permission Tiers**: Civilian / Military / Interstellar access levels

## Directory Structure
```
Hubs/
├── cmd/                 # Streaming server
├── internal/
│   ├── signaling/      # WebRTC signaling
│   ├── bridge/         # Feed conversion
│   └── auth/           # Tier-based access control
└── web/                # React viewer application
```

## Build Status
Phase 6 - Pending development

## Dependencies
- Go 1.21+
- pion/webrtc
- React 18
