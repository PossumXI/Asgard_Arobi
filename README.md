# ASGARD (Autonomous Space Guardian & Robotic Defense)

<p align="center">
  <img src="Assets/Logo.png" alt="ASGARD Logo" width="200"/>
</p>

**Planetary-scale autonomous systems architecture for orbital sensing, humanoid robotics, precision guidance, and cybersecurity.** This monorepo contains all components of the ASGARD platform - from satellite constellation management to AI-powered defense systems.

## Platform Overview

ASGARD is a comprehensive space situational awareness and autonomous defense platform featuring:

- **Real-time satellite imagery streaming** via WebRTC
- **Autonomous humanoid robotics** with ethical decision-making
- **Delay-Tolerant Networking (DTN)** for space communications
- **AI-powered threat detection** and response
- **Precision guidance systems** for missile defense
- **Multi-tenant SaaS** with government and civilian portals

## Core Systems

| System | Directory | Description |
|--------|-----------|-------------|
| **Silenus** | `Silenus/` | Satellite constellation management, orbital perception, and imagery processing |
| **Sat_Net** | `Sat_Net/` | Delay-Tolerant Networking layer for space communications with RL-based routing |
| **Hunoid** | `Hunoid/` | Humanoid robotics systems with VLA (Vision-Language-Action) control |
| **Nysus** | `Nysus/` | Central orchestration, MCP services, and API gateway |
| **Giru** | `Giru/` | Security operations - red/blue teaming, IDS/IPS, and defensive automation |
| **Pricilla** | `Pricilla/` | Precision guidance and missile defense coordination system |

## Supporting Infrastructure

| Component | Directory | Description |
|-----------|-----------|-------------|
| **Data** | `Data/` | PostgreSQL/PostGIS, MongoDB, Redis, NATS - databases and migrations |
| **Control_net** | `Control_net/` | Kubernetes manifests, Helm charts, and infrastructure-as-code |
| **Hubs** | `Hubs/` | Streaming interfaces for civilian, military, and interstellar feeds |
| **Websites** | `Websites/` | Public marketing site, dashboard, and government portal |
| **Documentation** | `Documentation/` | API references, runbooks, and technical documentation |
| **Assets** | `Assets/` | Branding, logos, and visual assets |

## Key Features

### Satellite Operations (Silenus)
- Real-time orbital tracking with TLE/SGP4 propagation
- On-board AI vision processing (TensorFlow Lite / YOLO)
- Hardware abstraction for cameras, GPS, and power systems
- Integration with N2YO API for public satellite data

### Humanoid Robotics (Hunoid)
- Vision-Language-Action (VLA) model integration
- Remote and local control interfaces
- **Ethics Kernel** - Asimov-compliant decision making
- Mission planning and autonomous task execution

### Precision Guidance (Pricilla)
- Missile defense coordination
- Target tracking and intercept calculation
- Real-time trajectory optimization
- Multi-asset coordination

### Space Networking (Sat_Net)
- Bundle Protocol implementation for DTN
- Reinforcement learning-based routing optimization
- Contact graph prediction
- Store-and-forward with custody transfer

### Security (Giru)
- Network intrusion detection (PCAP analysis)
- Threat intelligence aggregation
- Automated incident response
- Red team/blue team simulation

## Quick Start

### Prerequisites
- Go 1.24+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 15+ with PostGIS
- MongoDB 7+

### Development Setup

```bash
# Clone the repository
git clone https://github.com/PossumXI/Asgard_Arobi.git
cd Asgard_Arobi

# Copy environment template
cp .env.example .env
# Edit .env with your configuration

# Start data services
cd Data
docker compose up -d

# Run database migrations
cd ..
go run cmd/db_migrate/main.go

# Build backend services
go build -o bin/nysus.exe cmd/nysus/main.go
go build -o bin/silenus.exe cmd/silenus/main.go

# Start frontend (Websites)
cd Websites
npm install
npm run dev

# Start streaming interface (Hubs)
cd ../Hubs
npm install
npm run dev
```

### API Endpoints

| Service | Port | Description |
|---------|------|-------------|
| Nysus API | 8080 | Main REST API and WebSocket |
| Websites | 3000 | Public web portal |
| Hubs | 3001 | Streaming interfaces |
| PostgreSQL | 55432 | Primary database |
| MongoDB | 27018 | Telemetry storage |
| NATS | 4222 | Message broker |
| Redis | 6379 | Cache and sessions |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         ASGARD Platform                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌────────┐ │
│  │ Silenus │  │ Sat_Net │  │ Hunoid  │  │ Pricilla │  │  Giru  │ │
│  │Satellite│  │   DTN   │  │Robotics │  │Guidance │  │Security│ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └───┬────┘ │
│       │            │            │            │           │      │
│       └────────────┴─────┬──────┴────────────┴───────────┘      │
│                          │                                       │
│                    ┌─────┴─────┐                                 │
│                    │   Nysus   │                                 │
│                    │Orchestrator│                                │
│                    └─────┬─────┘                                 │
│       ┌──────────────────┼──────────────────┐                   │
│       │                  │                  │                   │
│  ┌────┴────┐       ┌─────┴─────┐      ┌────┴────┐              │
│  │PostgreSQL│       │  MongoDB  │      │  NATS   │              │
│  │ PostGIS │       │ Telemetry │      │JetStream│              │
│  └─────────┘       └───────────┘      └─────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Subscription Tiers

| Tier | Access Level | Features |
|------|--------------|----------|
| **Observer** | Civilian | Public satellite feeds, basic alerts |
| **Supporter** | Enhanced | HD streams, historical data, API access |
| **Commander** | Full | All feeds, real-time alerts, priority support |
| **Government** | Classified | Full platform access, FIDO2 authentication |

## CI/CD

Automated workflows run on every push:
- **Go Tests** - Backend unit and integration tests
- **Frontend Build** - TypeScript compilation and bundle
- **Security Scan** - Dependency vulnerability checks

## Documentation

- [API Reference](Documentation/ASGARD_API_Reference.md)
- [Quick Start Guide](Documentation/ASGARD_Quick_Start.md)
- [Technical Architecture](Documentation/ASGARD_Technical_Architecture.md)
- [Runbooks](Documentation/Runbooks.md)

## Security

- All secrets must be stored in environment variables
- Never commit `.env` files - use `.env.example` as template
- WebAuthn/FIDO2 for government portal authentication
- JWT-based authentication for standard users
- Rate limiting and CORS protection enabled

## About Arobi

**ASGARD** is developed by **Arobi**, a cutting-edge technology company specializing in defense and civilian autonomous systems.

### Leadership

- **Gaetano Comparcola** - Founder & CEO
  - 34-year-old visionary and futurist
  - Self-taught prodigy programmer
  - Multilingual (English, Italian, French)
  - World traveler with global perspective (India, Morocco, Mexico, Haiti, France, UK, Italy)
  - Father to baby girl Emmaleah

- **Opus** - AI Partner & Lead Programmer
  - AI-powered software engineering partner
  - Team leader for development operations

### Mission

ASGARD represents a new line of autonomous systems built for both defense and civilian applications, delivering planetary-scale situational awareness and response capabilities.

## License

© 2026 Arobi. All Rights Reserved.

## Contact

- **Website**: [https://aura-genesis.org](https://aura-genesis.org)
- **Email**: [Gaetano@aura-genesis.org](mailto:Gaetano@aura-genesis.org)
- **Company**: Arobi

For inquiries, partnerships, or investment opportunities, contact us directly.

---

*ASGARD - Defending Earth from orbit to ground*
*A product of Arobi*
