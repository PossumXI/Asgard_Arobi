# Nysus - Central Nerve Center

## Overview
Nysus is the orchestration hub of ASGARD, functioning as the "central nervous system" that coordinates all subsystems including satellites (Silenus), humanoid robots (Hunoid), security (Giru), and guidance systems (Pricilla).

## Architecture
- **Event Bus**: Pub/sub system for real-time event distribution
- **Control Plane**: Unified cross-domain coordination via NATS
- **API Server**: RESTful + WebSocket APIs for web interfaces
- **MCP Server**: Model Context Protocol for LLM integration
- **AI Agents**: Specialized agents for analytics, coordination, security, and emergency response
- **Database Layer**: PostgreSQL for structured data, MongoDB for documents

## Directory Structure
```
Nysus/
├── README.md                    # This file
cmd/nysus/
├── main.go                      # Main entry point
internal/nysus/
├── api/
│   ├── server.go                # HTTP/WebSocket server
│   ├── websocket.go             # WebSocket handlers
│   ├── chat_store.go            # Chat message storage
│   ├── handlers_admin.go        # Admin API endpoints
│   ├── handlers_auth.go         # Authentication endpoints
│   ├── handlers_dashboard.go    # Dashboard statistics
│   ├── handlers_pricilla.go     # Pricilla integration
│   ├── handlers_satellite.go    # Satellite management
│   ├── handlers_streams.go      # Video stream handling
│   └── handlers_user.go         # User management
├── events/
│   ├── bus.go                   # Event bus implementation
│   └── types.go                 # Event type definitions
├── mcp/
│   └── server.go                # Model Context Protocol server
└── agents/
    └── coordinator.go           # AI agent coordinator
```

## Features Implemented

### Event System
- **Event Bus**: In-memory pub/sub with subscriber management
- **Event Types**: Alert, Threat, Telemetry, Mission, Command events
- **Control Plane Bridge**: Events forwarded to unified control plane

### MCP Server (NEW)
Exposes ASGARD capabilities as LLM-accessible tools:
- `get_satellite_status` - Query satellite telemetry
- `command_satellite` - Send commands to satellites
- `get_hunoid_status` - Query Hunoid unit status
- `dispatch_mission` - Dispatch Hunoid to missions
- `get_threat_status` - Query security threat landscape
- `initiate_scan` - Start security scans
- `calculate_trajectory` - Calculate trajectories via Pricilla

### AI Agents (NEW)
Specialized agents for automated operations:
| Agent | Type | Capabilities |
|-------|------|--------------|
| Analytics Agent | analytics | Telemetry analysis, anomaly detection, pattern recognition |
| Autonomous Agent | autonomous | Mission planning, resource allocation, contingency planning |
| Coordinator Agent | coordinator | Multi-satellite coordination, swarm management, orchestration |
| Security Agent | security | Threat assessment, vulnerability analysis, incident response |
| Emergency Agent | emergency | Disaster response, emergency coordination, resource mobilization |

### API Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/auth/signin` | POST | User authentication |
| `/api/auth/signup` | POST | User registration |
| `/api/dashboard/stats` | GET | Dashboard statistics |
| `/api/alerts` | GET | List alerts |
| `/api/missions` | GET | List missions |
| `/api/satellites` | GET | List satellites |
| `/api/hunoids` | GET | List hunoid units |
| `/api/streams` | GET | List video streams |
| `/ws/events` | WS | Event stream |
| `/ws/signaling` | WS | WebRTC SFU signaling |

### MCP Endpoints (NEW)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp/initialize` | POST | Initialize MCP session |
| `/mcp/tools/list` | GET | List available tools |
| `/mcp/tools/call` | POST | Execute a tool |
| `/mcp/resources/list` | GET | List available resources |
| `/mcp/resources/read` | GET | Read a resource |
| `/mcp/prompts/list` | GET | List available prompts |

## Build Status
**Phase: OPERATIONAL** (Full functionality including MCP and AI Agents)

## Usage

```powershell
# Run Nysus central server
$env:POSTGRES_HOST = "localhost"
$env:POSTGRES_PORT = "55432"
$env:POSTGRES_PASSWORD = "your-password"
$env:MONGO_HOST = "localhost"
$env:MONGO_PORT = "27018"

go run ./cmd/nysus/main.go -addr :8080
```

### Command-Line Flags
| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | :8080 | HTTP server address |
| `-db-host` | localhost | PostgreSQL host |
| `-db-port` | 55432 | PostgreSQL port |
| `-mongo-host` | localhost | MongoDB host |
| `-mongo-port` | 27017 | MongoDB port |

### Environment Variables
| Variable | Description |
|----------|-------------|
| `POSTGRES_HOST` | PostgreSQL server host |
| `POSTGRES_PORT` | PostgreSQL server port |
| `POSTGRES_PASSWORD` | Database password |
| `NATS_URL` | NATS server URL |
| `MCP_ADDR` | MCP server address (default: :8085) |
| `ASGARD_ALLOW_NO_DB` | Allow running without database |

## Dependencies
- Go 1.24+
- PostgreSQL 14+
- MongoDB 6+ (optional, default local port 27018)
- NATS JetStream (optional, for control plane)

## Integration Points
- **Silenus**: Receives satellite alerts and telemetry
- **Hunoid**: Mission dispatch and status tracking
- **Giru**: Security event integration
- **Pricilla**: Guidance system coordination
- **Hubs (Frontend)**: WebSocket connections for real-time UI
- **LLMs**: MCP tools for AI-assisted operations

## About Arobi

**Nysus** is part of the **ASGARD** platform, developed by **Arobi** - a cutting-edge technology company specializing in defense and civilian autonomous systems.

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
