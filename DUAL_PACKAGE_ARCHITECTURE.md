# ASGARD Dual-Package Architecture

## Overview

The ASGARD platform is distributed as two distinct packages:

1. **Civilian Package** - Public-facing web application for civilian use
2. **Defense/Government Package** - Secure Electron native application for authorized personnel

## Package Components

### Civilian Package

**Distribution**: Web Application (React + Vite)
**Access**: Public website at https://aura-genesis.org

| Component | Location | Port | Description |
|-----------|----------|------|-------------|
| Websites | `/Websites` | 3000 | Main marketing and civilian dashboard |
| Hubs | `/Hubs` | 3001 | Video streaming and civilian monitoring |
| Giru JARVIS | `/Giru/Giru(jarvis)` | 7777/7778 | AI voice assistant |
| Pricilla | `/Pricilla` | 8089 | Precision guidance (limited access) |

**Features Available:**
- Public information and marketing
- Civilian dashboard with limited telemetry
- Video streaming (tiered access)
- Weather and basic status information
- Customer support and contact
- Account management

### Defense/Government Package

**Distribution**: Electron Native Application
**Access**: Secure download with access code + FIDO2 authentication

| Component | Location | Port | Description |
|-----------|----------|------|-------------|
| GovClient | `/GovClient` | 3002 | Secure Electron application |
| Valkyrie | `/Valkyrie` | 8093/9093 | Autonomous flight control |
| Hunoid | `/internal/robotics` | 8090/9092 | Humanoid robotics control |
| Giru Security | `/Giru` | 9090/9091 | Security monitoring system |
| Nysus | `/cmd/nysus` | 8080/8085 | Central command system |
| Silenus | `/cmd/silenus` | 9093 | Satellite operations |

**Features Available:**
- Full mission command and control
- Real-time satellite imagery
- Valkyrie autonomous flight management
- Hunoid robotics coordination
- 360-degree threat perception
- Rescue prioritization system
- Military-grade encrypted communications
- Complete audit logging

## Security Architecture

### Civilian Package Security
- Standard HTTPS/TLS
- OAuth2/OpenID Connect authentication
- Rate limiting and DDoS protection
- Tiered data access based on subscription

### Government Package Security
- Two-factor authentication gate:
  1. Government Access Code (issued by administrator)
  2. FIDO2/WebAuthn with hardware security key
- Certificate pinning
- Encrypted local storage (electron-store)
- Auto-logout on inactivity
- Complete audit trail
- FedRAMP High / IL5 compliant infrastructure

## System Integration

### Shared Services

All packages communicate with these backend services:

```
                           ┌─────────────────┐
                           │    CIVILIANS    │
                           │  Web Browsers   │
                           └────────┬────────┘
                                    │
                           ┌────────▼────────┐
                           │    WEBSITES     │
                           │    (Public)     │
                           └────────┬────────┘
                                    │
    ┌───────────────────────────────┼───────────────────────────────┐
    │                               │                               │
    │                    ┌──────────▼──────────┐                    │
    │                    │       NYSUS         │                    │
    │                    │  Central Command    │                    │
    │                    │    (Port 8080)      │                    │
    │                    └──────────┬──────────┘                    │
    │                               │                               │
    │         ┌─────────────────────┼─────────────────────┐         │
    │         │                     │                     │         │
    │  ┌──────▼──────┐      ┌───────▼───────┐     ┌──────▼──────┐  │
    │  │   PRICILLA  │      │     GIRU      │     │   SILENUS   │  │
    │  │  Guidance   │      │   Security    │     │  Satellite  │  │
    │  │ (Port 8089) │      │ (Port 9090)   │     │ (Port 9093) │  │
    │  └─────────────┘      └───────────────┘     └─────────────┘  │
    │                               │                               │
    │                    ┌──────────▼──────────┐                    │
    │                    │    GIRU JARVIS      │                    │
    │                    │   Voice Control     │                    │
    │                    │  (Port 7777/7778)   │                    │
    │                    └──────────┬──────────┘                    │
    │                               │                               │
    │         ┌─────────────────────┼─────────────────────┐         │
    │         │                     │                     │         │
    │  ┌──────▼──────┐      ┌───────▼───────┐     ┌──────▼──────┐  │
    │  │  VALKYRIE   │      │    HUNOID     │     │  LIVEFEED   │  │
    │  │   Flight    │      │   Robotics    │     │  Streaming  │  │
    │  │ (Port 8093) │      │ (Port 8090)   │     │    (WS)     │  │
    │  └─────────────┘      └───────────────┘     └─────────────┘  │
    │                               │                               │
    └───────────────────────────────┼───────────────────────────────┘
                                    │
                           ┌────────▼────────┐
                           │   GOV CLIENT    │
                           │   (Electron)    │
                           └────────┬────────┘
                                    │
                           ┌────────▼────────┐
                           │   GOVERNMENT    │
                           │   Personnel     │
                           └─────────────────┘
```

### API Access Levels

| Endpoint | Civilian | Government |
|----------|----------|------------|
| `/api/status` | Read | Full |
| `/api/telemetry` | Limited | Full |
| `/api/missions` | None | Full |
| `/api/valkyrie/*` | None | Full |
| `/api/hunoid/*` | None | Full |
| `/api/satellite/*` | None | Full |
| `/api/military/*` | None | Full |
| `/ws/livefeed` | Tiered | Full |

## Deployment

### Civilian Package
- Deploy via standard web hosting (Vercel, Cloudflare, AWS)
- CDN distribution for static assets
- Auto-scaling based on traffic

### Government Package
- Distribute via secure download portal (`/gov/download`)
- Code-signed executables for Windows, macOS, Linux
- Auto-update via electron-updater
- Air-gapped deployment option available

## Version Compatibility

| Package | Version | Go | Node | Python |
|---------|---------|----|----|--------|
| Civilian | 1.0.0 | N/A | 18+ | N/A |
| Government | 1.0.0 | 1.24+ | 18+ | 3.13+ |
| Valkyrie | 1.0.0 | 1.24+ | N/A | N/A |
| Hunoid | 1.0.0 | 1.24+ | N/A | N/A |
| GIRU JARVIS | 2.0.0 | N/A | 18+ | 3.13+ |

---

Copyright 2026 Arobi. All Rights Reserved.
