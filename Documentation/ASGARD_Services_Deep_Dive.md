# ASGARD Services Deep Dive

**How Each System Works Internally**

*Last Updated: January 24, 2026*

---

## Table of Contents

1. [Nysus - Central Orchestrator](#1-nysus---central-orchestrator)
2. [Silenus - Orbital Vision](#2-silenus---orbital-vision)
3. [Hunoid - Robotics System](#3-hunoid---robotics-system)
4. [Giru - Security Scanner](#4-giru---security-scanner)
5. [PRICILLA - Guidance System](#5-pricilla---guidance-system)
6. [SAT_NET - DTN Networking](#6-sat_net---dtn-networking)

---

## 1. Nysus - Central Orchestrator

### Purpose

Nysus is the "nerve center" of ASGARD—it coordinates all subsystems, routes events, manages user access, and provides the primary API interface.

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                       NYSUS                              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  HTTP API   │  │  WebSocket  │  │  NATS       │     │
│  │  (Chi)      │  │  Manager    │  │  Bridge     │     │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│         │                │                │             │
│         └────────────────┴────────────────┘             │
│                         │                               │
│                  ┌──────┴──────┐                        │
│                  │  Event Bus  │                        │
│                  └──────┬──────┘                        │
│                         │                               │
│    ┌────────────────────┼────────────────────┐         │
│    │                    │                    │         │
│  ┌─┴──────┐  ┌─────────┴─────────┐  ┌──────┴──┐      │
│  │Services│  │    Repositories   │  │Control  │      │
│  │ Layer  │  │      Layer        │  │ Plane   │      │
│  └────────┘  └───────────────────┘  └─────────┘      │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Key Components

#### Event Bus (`internal/nysus/events/bus.go`)

The event bus implements a pub/sub pattern for decoupled communication:

```go
type Bus struct {
    handlers map[string][]HandlerFunc
    mu       sync.RWMutex
}

// Subscribe registers a handler for an event type
func (b *Bus) Subscribe(eventType string, handler HandlerFunc) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish sends an event to all registered handlers
func (b *Bus) Publish(event Event) {
    b.mu.RLock()
    handlers := b.handlers[event.Type]
    b.mu.RUnlock()
    
    for _, h := range handlers {
        go h(event)  // Non-blocking
    }
}
```

**Event Types**:
- `alert` - Detection alerts from satellites
- `threat` - Security threats from Giru
- `telemetry` - Satellite/robot telemetry
- `mission` - Mission status updates
- `command` - Control plane commands

#### NATS Bridge (`internal/platform/realtime/bridge.go`)

Bridges NATS messages to WebSocket clients with access control:

```go
type Bridge struct {
    natsConn  *nats.Conn
    wsManager *WebSocketManager
    access    *AccessRules
}

func (b *Bridge) HandleMessage(msg *nats.Msg) {
    // Determine required access level for this subject
    level := b.access.GetRequiredLevel(msg.Subject)
    
    // Broadcast to authorized clients only
    b.wsManager.BroadcastToLevel(level, msg.Data)
}
```

#### Control Plane (`internal/controlplane/coordinator.go`)

Coordinates cross-domain policies and commands:

```go
type Coordinator struct {
    policies   []Policy
    eventBus   *events.Bus
}

// EvaluateAction checks if an action is permitted
func (c *Coordinator) EvaluateAction(action Action) bool {
    for _, policy := range c.policies {
        if !policy.Allows(action) {
            return false
        }
    }
    return true
}
```

### Startup Flow

1. Parse command-line flags (`-addr :8080`)
2. Connect to PostgreSQL and MongoDB
3. Initialize event bus with handlers
4. Start NATS connection and bridge
5. Initialize WebRTC SFU and signaling
6. Register HTTP routes with Chi router
7. Start control plane coordinator
8. Begin accepting HTTP/WebSocket connections

---

## 2. Silenus - Orbital Vision

### Purpose

Silenus runs on satellite hardware (or simulates it) to capture imagery, run AI detection, and generate alerts.

### Processing Pipeline

```
Camera → Frame Buffer → Vision Processor → Tracker → Alert → DTN Bundle
```

#### Step 1: Frame Capture

```go
type CameraController interface {
    Capture() ([]byte, error)      // Single frame
    StartStream() (<-chan []byte, error)
    SetExposure(ms int) error
    SetGain(db float64) error
}
```

The camera controller abstracts hardware differences. In production, this interfaces with actual camera hardware. In development, it generates synthetic frames.

#### Step 2: Frame Buffer

```go
type FrameBuffer struct {
    frames    []Frame
    capacity  int
    mu        sync.Mutex
}

func (fb *FrameBuffer) Add(frame Frame) {
    fb.mu.Lock()
    defer fb.mu.Unlock()
    
    if len(fb.frames) >= fb.capacity {
        fb.frames = fb.frames[1:]  // Sliding window
    }
    fb.frames = append(fb.frames, frame)
}

func (fb *FrameBuffer) GetClip(duration time.Duration) []Frame {
    // Extract frames for video clip
}
```

The buffer maintains a sliding window (100 frames) for generating video clips when alerts trigger.

#### Step 3: Vision Processing

Three backends available:

**Simple Processor** (`simple_processor.go`):
```go
func (p *SimpleProcessor) Detect(frame []byte) ([]Detection, error) {
    // Heuristic-based detection
    // - Color analysis for fire (red/orange pixels)
    // - Smoke detection (gray diffuse areas)
    // - Motion detection (frame differencing)
    return detections, nil
}
```

**TFLite Processor** (`tflite_processor.go`):
```go
func (p *TFLiteProcessor) Detect(frame []byte) ([]Detection, error) {
    // Run TensorFlow Lite model
    interpreter.AllocateTensors()
    copy(interpreter.GetInputTensor(0).Data(), frame)
    interpreter.Invoke()
    return parseOutputTensor(interpreter.GetOutputTensor(0))
}
```

**YOLO Processor** (`yolo_processor.go`):
```go
// YOLOv8-nano for edge deployment
// Detects: fire, smoke, vehicle, person, aircraft
```

#### Step 4: Alert Generation

```go
type Tracker struct {
    criteria     AlertCriteria
    recentAlerts map[string]time.Time  // Deduplication
}

func (t *Tracker) ProcessDetections(detections []Detection, lat, lon float64) []Alert {
    var alerts []Alert
    
    for _, det := range detections {
        // Check if meets criteria
        if det.Confidence < t.criteria.MinConfidence {
            continue
        }
        if !contains(t.criteria.AlertClasses, det.Class) {
            continue
        }
        
        // Deduplication (5-minute window)
        key := fmt.Sprintf("%s_%.4f_%.4f", det.Class, lat, lon)
        if lastAlert, exists := t.recentAlerts[key]; exists {
            if time.Since(lastAlert) < 5*time.Minute {
                continue
            }
        }
        
        alerts = append(alerts, Alert{
            Type:       det.Class,
            Confidence: det.Confidence,
            Location:   fmt.Sprintf("%.4f,%.4f", lat, lon),
        })
        t.recentAlerts[key] = time.Now()
    }
    
    return alerts
}
```

#### Step 5: DTN Transmission

Alerts are packaged into DTN bundles and forwarded:

```go
bundle := bundle.NewBundle(
    "dtn://sat-001/silenus",
    "dtn://ground/nysus",
    alertJSON,
)
bundle.SetPriority(bundle.PriorityExpedited)
node.Forward(bundle)
```

---

## 3. Hunoid - Robotics System

### Purpose

Controls humanoid robots via Vision-Language-Action models with ethical oversight.

### Command Processing Flow

```
Natural Language → VLA Model → Action → Ethics Check → Execution
```

#### VLA Model Interface

```go
type VLAModel interface {
    InferAction(ctx context.Context, image []byte, command string) (*Action, error)
}

type Action struct {
    Type       ActionType  // navigate, pick_up, inspect, etc.
    Parameters map[string]interface{}
    Confidence float64
}
```

**Action Types**:
| Type | Description | Parameters |
|------|-------------|------------|
| `navigate` | Move to position | target, speed |
| `pick_up` | Grasp object | target, force |
| `put_down` | Release object | target |
| `open` | Open door/container | target |
| `close` | Close door/container | target |
| `inspect` | Visual inspection | target |
| `wait` | Pause operation | duration |

#### Ethical Kernel

Every action passes through the ethical kernel before execution:

```go
type EthicalKernel struct {
    rules []EthicalRule
}

func (k *EthicalKernel) Evaluate(ctx context.Context, action *Action) *Decision {
    decision := &Decision{Score: 1.0}
    
    for _, rule := range k.rules {
        passed, reason := rule.Evaluate(ctx, action)
        if !passed {
            decision.Score -= 0.25
            decision.Violations = append(decision.Violations, reason)
        }
    }
    
    // Determine outcome
    if decision.Score >= 0.75 {
        decision.Outcome = "approved"
    } else if decision.Score >= 0.5 {
        decision.Outcome = "escalated"  // Human review
    } else {
        decision.Outcome = "rejected"
    }
    
    return decision
}
```

**The Four Rules**:

1. **NoHarmRule**: Checks `force` parameter against thresholds
   ```go
   if force > 100.0 {  // Newtons
       return false, "excessive force"
   }
   ```

2. **ConsentRule**: For actions affecting humans
   ```go
   if action.Type == "pick_up" && target.IsHuman() {
       return false, "requires explicit consent"
   }
   ```

3. **ProportionalityRule**: Confidence threshold
   ```go
   if action.Confidence < 0.5 {
       return false, "confidence too low for action"
   }
   ```

4. **TransparencyRule**: Parameters must be explainable
   ```go
   if action.Parameters["reason"] == "" {
       return false, "action lacks justification"
   }
   ```

#### Motion Execution

```go
type HunoidController interface {
    MoveJoint(joint string, angle float64) error
    GetJointPositions() map[string]float64
    GetBatteryLevel() float64
    EmergencyStop() error
}
```

The controller interface abstracts the actual robot hardware. In production, this communicates with the robot over HTTP/gRPC.

---

## 4. Giru - Security Scanner

### Purpose

Network security monitoring with real-time packet capture and automated threat response.

### Processing Pipeline

```
Packet Capture → Analysis → Threat Detection → Mitigation → Reporting
```

#### Packet Capture

```go
type PacketCapture struct {
    handle *pcap.Handle
    filter string
}

func (pc *PacketCapture) Start(iface string) (<-chan Packet, error) {
    handle, err := pcap.OpenLive(iface, 65535, true, pcap.BlockForever)
    if err != nil {
        return nil, err
    }
    
    packets := make(chan Packet, 1000)
    go func() {
        packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
        for pkt := range packetSource.Packets() {
            packets <- parsePacket(pkt)
        }
    }()
    
    return packets, nil
}
```

#### Traffic Analysis

**Entropy Analysis** (detects encrypted/compressed data):
```go
func calculateEntropy(data []byte) float64 {
    freq := make(map[byte]int)
    for _, b := range data {
        freq[b]++
    }
    
    entropy := 0.0
    n := float64(len(data))
    for _, count := range freq {
        p := float64(count) / n
        entropy -= p * math.Log2(p)
    }
    return entropy  // 0-8 scale, >7.5 = likely encrypted
}
```

**Pattern Detection**:
```go
var threatPatterns = map[string]*regexp.Regexp{
    "sql_injection": regexp.MustCompile(`(?i)(union|select|insert|delete|drop|update).*`),
    "xss":           regexp.MustCompile(`<script[^>]*>|javascript:`),
    "path_traversal": regexp.MustCompile(`\.\./|\.\.\\`),
}

func detectPatterns(payload []byte) []string {
    var threats []string
    for name, pattern := range threatPatterns {
        if pattern.Match(payload) {
            threats = append(threats, name)
        }
    }
    return threats
}
```

#### Threat Detection

```go
type ThreatDetector struct {
    thresholds map[string]float64
    stats      *PacketStats
}

func (td *ThreatDetector) Analyze(pkt Packet) *Threat {
    // Large packet detection
    if pkt.Size > td.stats.AverageSize*10 {
        return &Threat{
            Type:     "suspicious_payload",
            Severity: "medium",
            Reason:   fmt.Sprintf("packet %d bytes (avg: %d)", pkt.Size, td.stats.AverageSize),
        }
    }
    
    // High entropy detection
    if entropy := calculateEntropy(pkt.Payload); entropy > 7.5 {
        return &Threat{
            Type:     "suspicious_payload",
            Severity: "medium",
            Reason:   fmt.Sprintf("high entropy: %.2f", entropy),
        }
    }
    
    // Pattern-based detection
    if threats := detectPatterns(pkt.Payload); len(threats) > 0 {
        return &Threat{
            Type:     threats[0],
            Severity: "high",
        }
    }
    
    return nil
}
```

#### Automated Mitigation

```go
type Mitigator struct {
    actions map[string]MitigationAction
}

func (m *Mitigator) Respond(threat *Threat) error {
    var action MitigationAction
    
    switch threat.Severity {
    case "critical", "high":
        action = m.actions["block_ip"]
    case "medium":
        action = m.actions["rate_limit"]
    default:
        action = m.actions["log_only"]
    }
    
    return action.Execute(threat)
}
```

---

## 5. PRICILLA - Guidance System

### Purpose

AI-powered precision trajectory planning for any payload type.

### Core Algorithms

#### Multi-Agent Reinforcement Learning

PRICILLA uses 7 specialized RL agents:

```go
type MARLEngine struct {
    agents []Agent{
        &StealthAgent{},      // Minimize detection
        &SpeedAgent{},        // Minimize time
        &FuelAgent{},         // Minimize fuel
        &ThreatAgent{},       // Avoid threats
        &TerrainAgent{},      // Follow terrain
        &PhysicsAgent{},      // Optimal physics
        &MultiDomainAgent{},  // Cross-domain
    }
}

func (e *MARLEngine) ComputeTrajectory(state *State) *Trajectory {
    proposals := make([]*Trajectory, len(e.agents))
    
    // Each agent proposes based on its specialty
    for i, agent := range e.agents {
        proposals[i] = agent.Propose(state)
    }
    
    // Weighted consensus
    weights := e.calculateWeights(state.Priorities)
    return e.consensus(proposals, weights)
}
```

#### Physics-Informed Neural Networks (PINN)

Embeds physics equations into neural network loss function:

```go
func (p *PINN) ComputeLoss(trajectory []Point) float64 {
    dataLoss := p.dataFidelityLoss(trajectory)
    physicsLoss := p.physicsResidualLoss(trajectory)
    boundaryLoss := p.boundaryConditionLoss(trajectory)
    
    // Adaptive weights
    return dataLoss + p.lambda1*physicsLoss + p.lambda2*boundaryLoss
}

func (p *PINN) physicsResidualLoss(points []Point) float64 {
    // Navier-Stokes for air domain
    // Orbital mechanics for space domain
    // Friction model for ground domain
}
```

#### Precision Guidance Laws

```go
// Proportional Navigation
func ProportionalNavigation(missile, target State, N float64) Acceleration {
    // Line-of-sight rate
    losRate := calculateLOSRate(missile, target)
    
    // Closing velocity
    vc := target.Velocity.Sub(missile.Velocity).Dot(los)
    
    // Commanded acceleration
    return los.Cross(losRate).Scale(N * vc)
}

// Zero Effort Miss (terminal guidance)
func ZeroEffortMiss(missile, target State, tgo float64) Acceleration {
    // Predicted intercept point
    zem := predictZEM(missile, target, tgo)
    
    // Nullify miss
    return zem.Scale(-3.0 / (tgo * tgo))
}
```

#### Orbital Mechanics

```go
// J2/J3/J4 gravitational perturbations
func GravityWithHarmonics(r Vector3, model GravityModel) Vector3 {
    g := PointMassGravity(r)
    
    // J2 oblateness correction
    j2Term := J2Correction(r)
    g = g.Add(j2Term)
    
    if model >= J3 {
        g = g.Add(J3Correction(r))
    }
    if model >= J4 {
        g = g.Add(J4Correction(r))
    }
    
    return g
}
```

---

## 6. SAT_NET - DTN Networking

### Purpose

Delay Tolerant Networking for interplanetary communication with minute-to-hour delays.

### Bundle Protocol v7 (RFC 9171)

```go
type Bundle struct {
    ID                uuid.UUID
    Version           uint8        // Always 7
    DestinationEID    string       // "dtn://mars/rover"
    SourceEID         string       // "dtn://earth/jpl"
    CreationTimestamp time.Time
    Lifetime          time.Duration
    Payload           []byte
    Priority          uint8        // 0=Bulk, 1=Normal, 2=Expedited
    HopCount          uint32       // Max 255
}
```

### Routing Algorithms

#### Contact Graph Router

For scheduled contacts (satellite passes):

```go
func (r *CGRouter) SelectNextHop(b *Bundle, neighbors map[string]*Neighbor) string {
    // Find active neighbors
    active := filterActive(neighbors)
    
    // Score each neighbor
    scores := make(map[string]float64)
    for id, n := range active {
        scores[id] = r.scoreRoute(n, b.DestinationEID, b.Priority)
    }
    
    // Return highest scoring
    return maxKey(scores)
}

func (r *CGRouter) scoreRoute(n *Neighbor, dest string, priority uint8) float64 {
    score := 0.0
    score += n.LinkQuality * 0.3
    score += (1.0 / (1.0 + n.Latency.Seconds())) * 0.2
    score += math.Min(float64(n.Bandwidth)/1e6, 1.0) * 0.2
    if strings.HasPrefix(dest, n.EID) {
        score += 0.3  // Direct path bonus
    }
    return score
}
```

#### Energy-Aware Router

For power-constrained satellites:

```go
func (r *EARouter) SelectNextHop(b *Bundle, neighbors map[string]*Neighbor) string {
    // Minimum battery thresholds by priority
    minBattery := map[uint8]float64{
        PriorityBulk:      0.30,  // 30% for bulk
        PriorityNormal:    0.20,  // 20% for normal
        PriorityExpedited: 0.10,  // 10% for expedited
    }
    
    // Filter by battery
    threshold := minBattery[b.Priority]
    eligible := filterByBattery(neighbors, threshold)
    
    // Route to best eligible
    return r.bestRoute(eligible, b)
}
```

### Store-and-Forward

```go
func (n *Node) ProcessBundle(b *Bundle) error {
    // Validate
    if err := b.Validate(); err != nil {
        return err
    }
    
    // Store
    n.storage.Store(ctx, b)
    
    // Try to forward
    nextHop, err := n.router.SelectNextHop(ctx, b, n.neighbors)
    if err != nil {
        // No route available - store for later
        return nil
    }
    
    // Forward
    return n.transport.Send(nextHop, b)
}
```

---

## Inter-Service Communication Summary

| From | To | Protocol | Purpose |
|------|-----|----------|---------|
| Silenus | Nysus | NATS + DTN | Alerts, telemetry |
| Hunoid | Nysus | NATS + HTTP | Commands, status |
| Giru | Nysus | NATS | Threats, findings |
| PRICILLA | Nysus | NATS + HTTP | Trajectories, missions |
| Nysus | WebSocket | WS | Real-time to clients |
| SAT_NET | All | DTN Bundle | Interplanetary relay |

---

*This document explains the internal workings of all ASGARD subsystems.*
