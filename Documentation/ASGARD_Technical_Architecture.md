# ASGARD Technical Architecture

**Deep Dive into System Design, Integration Patterns, and Implementation Details**

*Last Updated: January 24, 2026*

---

## Table of Contents

1. [Project Structure](#project-structure)
2. [Service Architecture](#service-architecture)
3. [Integration Patterns](#integration-patterns)
4. [Data Flow](#data-flow)
5. [API Design](#api-design)
6. [Algorithm Quality](#algorithm-quality)
7. [Code Quality Metrics](#code-quality-metrics)

---

## Project Structure

```
ASGARD/
├── cmd/                          # Executable entry points
│   ├── nysus/main.go            # Central orchestrator (2,500+ lines)
│   ├── silenus/main.go          # Satellite vision system
│   ├── hunoid/main.go           # Robotics controller
│   ├── giru/main.go             # Security scanner
│   ├── satnet_router/main.go    # DTN routing node
│   ├── satellite_tracker/main.go # Orbital tracking CLI
│   ├── satnet_verify/main.go    # DTN verification tool
│   └── db_migrate/main.go       # Database migration tool
│
├── internal/                     # Private application code
│   ├── api/                     # HTTP API layer
│   │   ├── handlers/            # Request handlers
│   │   │   ├── auth.go         # Authentication endpoints
│   │   │   ├── dashboard.go    # Dashboard data
│   │   │   ├── stream.go       # Streaming endpoints
│   │   │   ├── subscription.go # Billing endpoints
│   │   │   ├── health.go       # Health checks
│   │   │   └── pricilla.go     # PRICILLA API
│   │   ├── middleware/          # HTTP middleware
│   │   │   ├── auth.go         # JWT validation
│   │   │   ├── access.go       # Access control
│   │   │   ├── logging.go      # Request logging
│   │   │   └── recovery.go     # Panic recovery
│   │   ├── webrtc/             # WebRTC SFU
│   │   │   └── sfu.go          # Selective forwarding unit
│   │   ├── signaling/          # WebRTC signaling
│   │   │   └── server.go       # ICE/SDP exchange
│   │   └── router.go           # Chi router setup
│   │
│   ├── services/                # Business logic
│   │   ├── auth.go             # Authentication (Argon2id, JWT)
│   │   ├── dashboard.go        # Data aggregation
│   │   ├── stream.go           # Stream management
│   │   ├── subscription.go     # Stripe integration
│   │   ├── email.go            # SMTP service
│   │   ├── satellite_tracking.go # Orbit propagation
│   │   └── pricilla.go         # PRICILLA integration
│   │
│   ├── repositories/            # Data access layer
│   │   ├── user.go             # User CRUD
│   │   ├── satellite.go        # Satellite data
│   │   ├── hunoid.go           # Robot data
│   │   ├── mission.go          # Mission tracking
│   │   ├── alert.go            # Alert storage
│   │   ├── stream.go           # Stream metadata
│   │   └── subscription.go     # Subscription data
│   │
│   ├── platform/                # Infrastructure services
│   │   ├── db/                 # Database connections
│   │   │   ├── postgres.go     # PostgreSQL client
│   │   │   ├── mongodb.go      # MongoDB client
│   │   │   ├── config.go       # Connection config
│   │   │   └── models.go       # ORM models
│   │   ├── dtn/                # Delay Tolerant Networking
│   │   │   ├── node.go         # DTN node implementation
│   │   │   ├── router.go       # Routing algorithms
│   │   │   ├── rl_router.go    # RL-based routing
│   │   │   ├── storage.go      # Bundle storage
│   │   │   └── contact_predictor.go # Orbital contacts
│   │   ├── realtime/           # Real-time infrastructure
│   │   │   ├── bridge.go       # NATS-WebSocket bridge
│   │   │   ├── websocket.go    # WebSocket manager
│   │   │   └── access.go       # Channel access control
│   │   ├── satellite/          # Satellite tracking
│   │   │   ├── client.go       # TLE/N2YO API client
│   │   │   └── propagator.go   # SGP4 orbit propagation
│   │   └── observability/      # Monitoring
│   │       ├── metrics.go      # Prometheus metrics
│   │       └── tracing.go      # OpenTelemetry traces
│   │
│   ├── orbital/                 # Satellite-specific code
│   │   ├── hal/                # Hardware abstraction
│   │   │   ├── camera.go       # Camera controller
│   │   │   ├── power.go        # Power management
│   │   │   └── orbital_position.go # GPS/propagation
│   │   ├── vision/             # AI vision processing
│   │   │   ├── processor.go    # Detection interface
│   │   │   ├── simple_processor.go # Heuristic detection
│   │   │   ├── tflite_processor.go # TensorFlow Lite
│   │   │   └── yolo_processor.go # YOLO detection
│   │   └── tracking/           # Alert generation
│   │       └── tracker.go      # Detection→Alert pipeline
│   │
│   ├── robotics/                # Humanoid robot code
│   │   ├── control/            # Robot control
│   │   │   ├── interfaces.go   # Controller interfaces
│   │   │   ├── hunoid_controller.go
│   │   │   └── remote_hunoid.go # HTTP-based control
│   │   ├── ethics/             # Ethical decision making
│   │   │   └── kernel.go       # 4-rule ethical kernel
│   │   └── vla/                # Vision-Language-Action
│   │       ├── interface.go    # VLA model interface
│   │       ├── http_vla.go     # Remote VLA inference
│   │       └── openvla.go      # OpenVLA implementation
│   │
│   ├── security/                # Giru security system
│   │   ├── scanner/            # Packet analysis
│   │   │   ├── interface.go    # Scanner interface
│   │   │   ├── capture.go      # Packet capture
│   │   │   ├── analyzer.go     # Traffic analysis
│   │   │   └── log_ingestion.go # Log-based scanning
│   │   ├── threat/             # Threat detection
│   │   │   └── detector.go     # Anomaly detection
│   │   ├── mitigation/         # Response actions
│   │   │   └── responder.go    # Automated mitigation
│   │   └── events/             # Security events
│   │       ├── schema.go       # Event types
│   │       └── publisher.go    # NATS publisher
│   │
│   ├── controlplane/            # Unified control
│   │   ├── coordinator.go      # Policy coordination
│   │   ├── events.go           # Cross-domain events
│   │   └── unified.go          # Control plane logic
│   │
│   └── nysus/                   # Nysus-specific code
│       ├── api/                # Additional handlers
│       │   ├── server.go
│       │   └── websocket.go
│       └── events/             # Event bus
│           ├── bus.go          # Pub/sub implementation
│           └── types.go        # Event definitions
│
├── pkg/                         # Public packages
│   └── bundle/                 # DTN bundle protocol
│       ├── bundle.go           # Bundle struct (BPv7)
│       └── serialization.go    # Marshal/unmarshal
│
├── Pricilla/                    # PRICILLA guidance system
│   ├── cmd/percila/main.go     # Entry point
│   └── internal/
│       ├── guidance/           # AI trajectory planning
│       │   ├── ai_engine.go    # MARL + PINN
│       │   └── interfaces.go   # Core types
│       ├── physics/            # Physics models
│       │   ├── orbital_mechanics.go # Gravity, drag
│       │   └── precision_interceptor.go # Guidance laws
│       ├── prediction/         # Kalman filtering
│       │   └── predictor.go    # State estimation
│       ├── sensors/            # Sensor fusion
│       │   └── fusion.go       # Extended Kalman Filter
│       └── integration/        # ASGARD integration
│           ├── asgard.go       # Service clients
│           └── nats_bridge.go  # Event streaming
│
├── Data/                        # Database infrastructure
│   ├── docker-compose.yml      # Container orchestration
│   ├── migrations/
│   │   ├── postgres/           # 10 migration files
│   │   └── mongo/              # Collection setup
│   └── init_databases.ps1      # Initialization script
│
├── Control_net/                 # Kubernetes deployment
│   ├── deploy.ps1              # Deployment script
│   └── kubernetes/             # K8s manifests
│       ├── namespace.yaml
│       ├── secrets.yaml
│       ├── nysus/              # Nysus deployment
│       ├── silenus/            # Silenus deployment
│       ├── hunoid/             # Hunoid deployment
│       ├── giru/               # Giru deployment
│       └── kustomization.yaml
│
├── Websites/                    # Web portal (React/TypeScript)
│   └── src/
│       ├── pages/              # Route components
│       ├── components/         # Reusable UI
│       ├── lib/                # API clients, types
│       ├── hooks/              # React hooks
│       └── stores/             # Zustand state
│
├── Hubs/                        # Streaming app (React/TypeScript)
│   └── src/
│       ├── pages/              # Hub views
│       ├── components/         # Video player, cards
│       └── lib/                # WebRTC client
│
├── test/                        # Test suites
│   ├── integration/            # 10 test files, 68 tests
│   ├── e2e/                    # End-to-end tests
│   └── hil/                    # Hardware-in-loop tests
│
├── scripts/                     # Utility scripts
│   ├── integration_test.ps1    # Full test suite
│   ├── load_test_realtime.ps1  # WebSocket load test
│   ├── load_test_signaling.ps1 # WebRTC load test
│   └── pricilla_demo.ps1       # PRICILLA demonstration
│
└── Documentation/               # This documentation
```

---

## Service Architecture

### Nysus - The Orchestrator

```go
// Core components initialized in cmd/nysus/main.go

type NysusServer struct {
    // Database connections
    postgresDB  *sql.DB
    mongoDB     *mongo.Client
    
    // Event infrastructure
    eventBus    *events.Bus
    natsBridge  *realtime.Bridge
    
    // Real-time services
    wsManager   *realtime.WebSocketManager
    sfuServer   *webrtc.SFU
    signaling   *signaling.Server
    
    // Control plane
    coordinator *controlplane.Coordinator
    
    // HTTP router
    router      *chi.Mux
}
```

**Startup Sequence**:
1. Connect to PostgreSQL and MongoDB
2. Initialize event bus with handlers
3. Start NATS bridge with access control
4. Initialize WebRTC SFU and signaling
5. Register HTTP routes with middleware
6. Start control plane coordinator
7. Begin accepting connections

### Event Flow Architecture

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Silenus │────▶│  NATS   │────▶│  Nysus  │
│ (Alert) │     │ Broker  │     │ (Route) │
└─────────┘     └────┬────┘     └────┬────┘
                     │               │
                     ▼               ▼
              ┌─────────┐     ┌─────────┐
              │  Giru   │     │ WebSocket│
              │(Analyze)│     │ Clients  │
              └─────────┘     └─────────┘
```

**NATS Subjects**:
| Subject | Access Level | Purpose |
|---------|--------------|---------|
| `asgard.alerts.public` | Public | General alerts |
| `asgard.alerts.>` | Civilian | All alert types |
| `asgard.telemetry.>` | Civilian | Satellite telemetry |
| `asgard.military.alerts` | Military | Tactical alerts |
| `asgard.hunoids.status` | Military | Robot status |
| `asgard.gov.threats` | Government | Critical threats |
| `asgard.security.findings` | Government | Security events |
| `asgard.admin.>` | Admin | System administration |

---

## Integration Patterns

### 1. Service-to-Service Communication

**Pattern**: Event-driven via NATS with fallback HTTP

```go
// Publishing an alert (Silenus)
natsConn.Publish("asgard.alerts.fire", alertJSON)

// Subscribing to alerts (Nysus)
natsConn.Subscribe("asgard.alerts.>", func(msg *nats.Msg) {
    handleAlert(msg.Data)
})
```

### 2. Real-time Client Updates

**Pattern**: NATS → Bridge → WebSocket

```go
// Bridge receives NATS message
bridge.Subscribe("asgard.alerts.>", func(msg *nats.Msg) {
    // Check access control
    if canAccess(client.AccessLevel, msg.Subject) {
        wsManager.Broadcast(client.ID, msg.Data)
    }
})
```

### 3. Database Access

**Pattern**: Repository pattern with connection pooling

```go
// Repository interface
type AlertRepository interface {
    Create(ctx context.Context, alert *Alert) error
    GetByID(ctx context.Context, id string) (*Alert, error)
    List(ctx context.Context, filter AlertFilter) ([]*Alert, error)
}

// PostgreSQL implementation
func (r *postgresAlertRepo) Create(ctx context.Context, alert *Alert) error {
    query := `INSERT INTO alerts (id, type, severity, ...) VALUES ($1, $2, ...)`
    _, err := r.db.ExecContext(ctx, query, alert.ID, alert.Type, ...)
    return err
}
```

### 4. DTN Bundle Routing

**Pattern**: Store-and-forward with priority queuing

```go
// Bundle arrives at node
func (n *Node) HandleIncoming(b *bundle.Bundle) error {
    // Validate bundle
    if err := b.Validate(); err != nil {
        return err
    }
    
    // Store for forwarding
    n.storage.Store(ctx, b)
    
    // Find next hop
    nextHop, err := n.router.SelectNextHop(ctx, b, n.neighbors)
    if err != nil {
        // Store for later retry
        return nil
    }
    
    // Forward to next hop
    return n.forwardTo(nextHop, b)
}
```

---

## Data Flow

### Alert Generation Flow

```
1. SILENUS
   Camera → Frame Buffer → Vision Processor → Detection
   
2. TRACKING
   Detection → Alert Criteria → Deduplication → Alert
   
3. DTN
   Alert → Bundle → Storage → Router → Transmission
   
4. NYSUS
   Receive → Parse → Event Bus → Handlers
   
5. CLIENTS
   WebSocket → Access Check → Broadcast → UI Update
```

### Threat Detection Flow

```
1. GIRU CAPTURE
   Network Interface → Packet Capture → Buffer
   
2. ANALYSIS
   Packet → Entropy Calc → Pattern Match → Anomaly Score
   
3. DETECTION
   Anomaly → Threshold Check → Threat Classification
   
4. MITIGATION
   Threat → Severity Check → Action Selection → Execute
   
5. REPORTING
   Event → NATS Publish → Dashboard Update
```

### Mission Execution Flow

```
1. COMMAND
   User → API → Nysus → Event Bus → "mission.create"
   
2. PLANNING
   PRICILLA → Trajectory Calc → Waypoints → Validation
   
3. DISPATCH
   Nysus → Hunoid Selection → Mission Assignment
   
4. EXECUTION
   Hunoid → VLA Inference → Ethics Check → Action
   
5. MONITORING
   Telemetry → NATS → Dashboard → Real-time Update
```

---

## API Design

### REST Endpoints

**Authentication**:
```
POST /api/auth/signin          - Email/password login
POST /api/auth/signup          - New user registration
POST /api/auth/fido2/register  - WebAuthn registration
POST /api/auth/fido2/auth      - WebAuthn authentication
```

**Dashboard**:
```
GET  /api/dashboard/stats      - Aggregate statistics
GET  /api/alerts               - Alert list with filtering
GET  /api/missions             - Active missions
GET  /api/satellites           - Satellite fleet status
GET  /api/hunoids              - Robot fleet status
```

**Streaming**:
```
GET  /api/streams              - Available streams
GET  /api/streams/:id          - Stream details
POST /api/streams/:id/session  - Start WebRTC session
GET  /api/streams/featured     - Featured feeds
```

**Subscriptions**:
```
GET  /api/subscriptions/plans  - Available plans
POST /api/subscriptions/checkout - Create Stripe session
POST /api/subscriptions/portal - Customer portal
```

### WebSocket Endpoints

```
WS /ws/realtime?access={level} - Real-time events
WS /ws/signaling               - WebRTC signaling
WS /ws/events                  - Legacy event stream
```

### Response Format

```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

Error response:
```json
{
  "success": false,
  "error": {
    "code": "AUTH_REQUIRED",
    "message": "Authentication required"
  }
}
```

---

## Algorithm Quality

### 1. SGP4 Orbit Propagation

**Accuracy**: Sub-0.2° position error at LEO

```go
// SGP4 implementation with J2 perturbation
func (p *Propagator) Propagate(t time.Time) (lat, lon, alt float64) {
    // Calculate time since TLE epoch
    tsince := t.Sub(p.epoch).Minutes()
    
    // Mean motion (rad/min)
    n := p.n0 * (1 + p.n0dot*tsince + p.n0ddot*tsince*tsince)
    
    // Mean anomaly
    M := p.M0 + n*tsince
    
    // Kepler's equation (Newton-Raphson)
    E := solveKepler(M, p.e)
    
    // True anomaly
    nu := 2 * math.Atan2(
        math.Sqrt(1+p.e)*math.Sin(E/2),
        math.Sqrt(1-p.e)*math.Cos(E/2),
    )
    
    // J2 perturbations on RAAN and argument of perigee
    // ... (detailed calculations)
    
    return geodetic(r, v, t)
}
```

### 2. Ethical Kernel Evaluation

**Scoring**: 0.0-1.0 composite score with rule-based penalties

```go
func (k *EthicalKernel) Evaluate(ctx context.Context, action *vla.Action) (*EthicalDecision, error) {
    decision := &EthicalDecision{
        Score: 1.0,  // Start with perfect score
    }
    
    for _, rule := range k.rules {
        passed, reason := rule.Evaluate(ctx, action)
        if !passed {
            decision.Score -= 0.25  // Each violation reduces score
            decision.Violations = append(decision.Violations, reason)
        }
    }
    
    // Decision based on final score
    switch {
    case decision.Score >= 0.75:
        decision.Decision = DecisionApproved
    case decision.Score >= 0.5:
        decision.Decision = DecisionEscalated
    default:
        decision.Decision = DecisionRejected
    }
    
    return decision, nil
}
```

### 3. Threat Detection Entropy Analysis

**Method**: Shannon entropy for encrypted payload detection

```go
func calculateEntropy(data []byte) float64 {
    if len(data) == 0 {
        return 0
    }
    
    // Count byte frequencies
    freq := make(map[byte]int)
    for _, b := range data {
        freq[b]++
    }
    
    // Calculate Shannon entropy
    entropy := 0.0
    n := float64(len(data))
    for _, count := range freq {
        p := float64(count) / n
        entropy -= p * math.Log2(p)
    }
    
    return entropy  // Max ~8.0 for random data
}
```

### 4. PRICILLA Multi-Agent RL

**Architecture**: 7 specialized agents with consensus voting

```go
type MARLEngine struct {
    agents []Agent  // stealth, speed, fuel, threat, terrain, physics, multi-domain
}

func (e *MARLEngine) ComputeTrajectory(state *State) *Trajectory {
    proposals := make([]*Trajectory, len(e.agents))
    
    // Each agent proposes trajectory
    for i, agent := range e.agents {
        proposals[i] = agent.Propose(state)
    }
    
    // Consensus voting with weighted averaging
    return e.consensus(proposals, state.Priorities)
}
```

### 5. Contact Graph Routing

**Scoring**: Multi-factor route selection

```go
func (r *ContactGraphRouter) calculateRouteScore(neighbor *Neighbor, dest string, priority uint8) float64 {
    score := 0.0
    
    // Link quality (0-1)
    score += neighbor.LinkQuality * 0.3
    
    // Latency (inverse, normalized)
    latencyScore := 1.0 / (1.0 + neighbor.Latency.Seconds())
    score += latencyScore * 0.2
    
    // Bandwidth capacity
    bwScore := math.Min(float64(neighbor.Bandwidth)/1e6, 1.0)
    score += bwScore * 0.2
    
    // Path match (does EID prefix match destination?)
    if strings.HasPrefix(dest, neighbor.EID) {
        score += 0.3
    }
    
    return score
}
```

---

## Code Quality Metrics

### Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| `pkg/bundle` | 10 | 95% |
| `internal/platform/dtn` | 13 | 85% |
| `internal/robotics/ethics` | 8 | 90% |
| `internal/services/auth` | 3 | 70% |
| `internal/orbital/vision` | 5 | 80% |

### Code Statistics

| Metric | Value |
|--------|-------|
| Total Go files | 161 |
| Total lines of code | ~35,000 |
| Integration tests | 68 |
| External dependencies | 45 |
| Compiled binaries | 8 |

### Security Practices

- ✅ Argon2id password hashing
- ✅ JWT with configurable expiry
- ✅ FIDO2/WebAuthn support
- ✅ Access control on all endpoints
- ✅ Rate limiting on auth endpoints
- ✅ Input validation on all handlers
- ✅ SQL parameterized queries
- ✅ CORS configuration
- ✅ Secrets via environment variables

### Performance Benchmarks

| Operation | Time | Throughput |
|-----------|------|------------|
| SGP4 propagation | 3.0 ms/orbit | 333 orbits/sec |
| Gravity J2/J3/J4 | 29 ns | 34M ops/sec |
| Intercept calculation | 7.5 µs | 133K ops/sec |
| Bundle serialization | 15 µs | 67K bundles/sec |
| JWT validation | 50 µs | 20K tokens/sec |

---

## Conclusion

ASGARD's architecture is designed for:

1. **Reliability**: Store-and-forward messaging handles network partitions
2. **Scalability**: Stateless services can scale horizontally
3. **Security**: Defense in depth with access control at every layer
4. **Observability**: Prometheus metrics and OpenTelemetry tracing
5. **Maintainability**: Clean separation of concerns and interface-driven design

The codebase demonstrates production-grade engineering with comprehensive testing, clear documentation, and adherence to Go best practices.
