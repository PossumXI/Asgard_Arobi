# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ASGARD** (Autonomous Space Guardian & Robotic Defense) is a planetary-scale autonomous systems platform. This is a production-grade monorepo combining Go backend services, React/TypeScript frontends, and Kubernetes deployment configurations.

**Tech Stack**: Go 1.24, Node.js 20, React 18, Vite, PostgreSQL 15 (PostGIS), MongoDB 7, NATS, Redis, Docker/Kubernetes

## Build Commands

### Go Backend
```bash
# Build all services
go build -o bin/silenus.exe ./cmd/silenus/main.go
go build -o bin/nysus.exe ./cmd/nysus/main.go
go build -o bin/hunoid.exe ./cmd/hunoid/main.go
go build -o bin/giru.exe ./cmd/giru/main.go
go build -o bin/pricilla.exe ./cmd/pricilla/main.go

# Run tests
go test -v -race -coverprofile=coverage.out ./...

# Format and lint
gofmt -w .
go vet ./...
```

### Frontend - Websites (Port 3000)
```bash
cd Websites
npm install && npm run dev      # Development
npm run build                   # Production build
npm run lint                    # ESLint
npm test                        # Vitest
```

### Frontend - Hubs (Port 3001)
```bash
cd Hubs
npm install && npm run dev      # Development
npm run build                   # Production build
npm run lint                    # ESLint
```

### Infrastructure
```bash
cd Data && docker compose up -d   # PostgreSQL, MongoDB, NATS, Redis
go run ./cmd/db_migrate/main.go   # Run migrations
```

## Architecture

### Core Systems (Go - cmd/)
- **Silenus** - Orbital satellite constellation management with vision processing
- **Nysus** - Central orchestration hub (REST API :8080, WebSocket, MCP :8085)
- **Hunoid** - Autonomous humanoid robotics with ethics oversight
- **Giru** - AI security system, threat detection (:9090)
- **Pricilla** - Precision guidance for drones/robots/spacecraft
- **Sat_Net** - Delay-tolerant networking (Bundle Protocol v7)

### Shared Libraries (internal/)
- `internal/api/` - HTTP/WebSocket server components
- `internal/orbital/` - Satellite HAL and vision processing
- `internal/robotics/` - Robot control, ethics, VLA integration
- `internal/security/` - Threat detection, red/blue team automation
- `internal/platform/` - Database, DTN, observability
- `internal/nysus/` - Event bus, MCP protocol, AI agents

### Frontends
- **Websites/** - Public portal & government dashboard (Vite + React + TypeScript)
- **Hubs/** - Real-time streaming interface with HLS/WebRTC

### Infrastructure
- **Data/** - Docker Compose for local databases
- **Control_net/** - Kubernetes manifests and Helm charts

## Key Ports

| Service | Port | Purpose |
|---------|------|---------|
| Websites | 3000 | Public portal |
| Hubs | 3001 | Streaming interface |
| Nysus API | 8080 | REST + WebSocket |
| Nysus MCP | 8085 | LLM integration |
| Giru | 9090 | Security API |
| PostgreSQL | 55432 | Database |
| MongoDB | 27018 | Document store |
| NATS | 4222 | Message broker |

## Code Conventions

### Go
- Entry points in `cmd/*/main.go` are self-contained (~500-1700 lines)
- gofmt formatting enforced in CI
- Structured logging, no silent failures
- HAL pattern for hardware abstraction

### TypeScript/React
- Path alias: `@/*` maps to `src/*`
- Zustand for global state
- TanStack Query for data fetching
- Radix UI primitives with TailwindCSS

## Environment Setup

Copy `.env.example` to `.env` and configure:
- `POSTGRES_PASSWORD`, `MONGO_PASSWORD`, `REDIS_PASSWORD`
- `JWT_SECRET`, `STRIPE_SECRET_KEY`, `N2YO_API_KEY`

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`) runs:
- Go: `gofmt -l .`, `go vet ./...`, `govulncheck ./...`, tests with race detection
- Frontend: `npm run lint`, `npm run typecheck`, `npm run build`, `npm test`

All checks must pass before merge.

## Critical Principles

1. **Production-grade**: All code is fully functional, no stubs or TODO comments
2. **Secure**: JWT auth, WebAuthn/FIDO2 for government, AES-256 encryption
3. **Observable**: Prometheus metrics, structured logging
4. **Testable**: Unit, integration, and E2E tests required
