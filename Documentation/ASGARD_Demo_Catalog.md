# ASGARD Demonstration Catalog

This catalog lists all demo-ready capabilities across ASGARD systems, grouped by subsystem and cross-system integrations.

## Core Systems

- **Nysus (Orchestration)**
  - API health check and live stats
  - Event bus activity and MCP tools list
  - Dashboard metrics and alert feeds
- **Giru (Security)**
  - Threat zone retrieval
  - Security scan initiation and alert handling
  - Shadow Stack + Red/Blue team workflow (API mode)
- **Silenus (Orbital)**
  - Satellite tracking status (N2YO/TLE)
  - Vision pipeline alerts and telemetry feed
  - Prometheus metrics endpoint
- **Hunoid (Robotics)**
  - Mission scenarios (medical_aid, perimeter_check, hazard_response)
  - Ethical kernel approvals/rejections
  - Operator console controls and telemetry
- **Sat_Net (DTN)**
  - Router node status and bundle forwarding
  - Contact prediction and routing selection
- **Pricilla (Guidance)**
  - Mission creation and payload tracking
  - Terminal guidance and weather impact updates
  - ECM/jamming detection and avoidance
  - Targeting metrics and CEP estimates

## Interfaces & Hubs

- **Websites Portal (Public/Gov)**
  - Landing, Features, Pricing, Pricilla pages
  - User onboarding screens (Sign In/Sign Up)
  - Dashboard overview and alerts
  - Government portal access flow
- **Hubs Streaming UI**
  - Hubs home with category tiles
  - Civilian hub streams
  - Military hub streams
  - Interstellar hub feeds
  - Mission hub and live stream view

## AI Assistant (Giru JARVIS)

- Wake word activation & status transitions (standby/listening/speaking)
- Conversation panel with command responses
- Activity log entries for executed commands
- System status indicators (Pricilla/Nysus/Silenus/Hunoid/Security)

## Control & Infrastructure

- **Control_net (Kubernetes)**
  - Namespace and service deployments
  - Kustomize deployment flow
  - Secrets template review
- **Data Layer**
  - Docker Compose for Postgres/Mongo/NATS/Redis
  - Migration pipeline and health checks

## Integrated Scenarios

- Giru threat zone broadcast → Pricilla route optimization
- Nysus mission dispatch → Hunoid mission execution
- Silenus alert → Nysus event bus → Dashboard alert feed
- NATS realtime bridge → Websites dashboard live stats

## Demo Preconditions (Typical)

- Websites: `http://localhost:3000`
- Hubs: `http://localhost:3001`
- Nysus API: `http://localhost:8080/health`
- Giru API: `http://localhost:9090/health`
- Pricilla API: `http://localhost:8092/health`
- Silenus metrics: `http://localhost:9093/metrics`
