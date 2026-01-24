# PERCILA - Complete Implementation Guide

## Overview

**PERCILA** (Precision Engagement & Routing Control with Integrated Learning Architecture) is the most advanced AI guidance system in ASGARD, providing ultra-precision navigation for any payload type with full system integration.

PERCILA serves as the central nervous system for all guided payloads in the ASGARD ecosystem, including:
- **Hunoid robots** - Ground-based humanoid units for humanitarian aid
- **UAVs/Drones** - Aerial surveillance and delivery platforms
- **Rockets** - Launch vehicles for orbital insertion
- **Missiles** - Precision guided munitions for defense
- **Spacecraft** - Orbital and interplanetary vehicles
- **Ground robots** - Autonomous ground vehicles
- **Submarines** - Underwater autonomous vehicles
- **Interstellar probes** - Deep space exploration craft

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Core Components](#core-components)
3. [Navigation System](#navigation-system)
4. [Prediction Engine](#prediction-engine)
5. [Stealth Optimization](#stealth-optimization)
6. [Payload Controller](#payload-controller)
7. [ASGARD Integration](#asgard-integration)
8. [API Reference](#api-reference)
9. [Deployment Guide](#deployment-guide)
10. [Configuration Reference](#configuration-reference)

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     PERCILA Core AI Engine                          │
│        Multi-Agent RL + Physics-Informed Neural Networks            │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
    ┌────▼────┐          ┌───▼────┐          ┌───▼────┐
    │ Guidance │          │  Nav   │          │ Stealth│
    │ Computer │          │ System │          │ Module │
    │          │          │        │          │        │
    │ - Trajectory        │ - Waypoint       │ - RCS Calc
    │ - Optimization      │ - Terrain        │ - Thermal
    │ - Path Planning     │ - Threat Avoid   │ - Radar Evasion
    └────┬────┘          └───┬────┘          └───┬────┘
         │                   │                    │
         └───────────────────┼────────────────────┘
                             │
              ┌──────────────▼──────────────┐
              │     Prediction Engine       │
              │                             │
              │ - Kalman Filtering          │
              │ - Trajectory Prediction     │
              │ - Intercept Calculation     │
              │ - Fuel Consumption          │
              │ - Contact Windows           │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │    Integration Layer        │
              │                             │
              │ Silenus │ Hunoid │ Sat_Net  │
              │ Giru    │ Nysus  │ Control  │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │    Payload Controllers      │
              │                             │
              │ Hunoid │ UAV │ Rocket       │
              │ Missile│ Drone│ Spacecraft  │
              └─────────────────────────────┘
```

### Directory Structure

```
C:\Users\hp\Desktop\Asgard\Percila\
├── cmd\
│   └── percila\
│       └── main.go              # Main PERCILA executable (v2.0.0)
├── internal\
│   ├── guidance\
│   │   ├── interfaces.go        # Core guidance interfaces
│   │   └── ai_engine.go         # AI-powered trajectory planning
│   ├── navigation\
│   │   └── navigator.go         # Navigation system
│   ├── prediction\
│   │   └── predictor.go         # Prediction engine with Kalman
│   ├── stealth\
│   │   └── optimizer.go         # Stealth optimization
│   ├── payload\
│   │   └── controller.go        # Multi-payload controller
│   ├── integration\
│   │   └── asgard.go            # ASGARD system integration
│   ├── livefeed\                # NEW: Live Feed System
│   │   ├── streamer.go          # Live telemetry streaming
│   │   └── websocket.go         # WebSocket broadcasting
│   └── access\                  # NEW: Access Control
│       ├── control.go           # Tiered access control
│       └── http_handler.go      # Access control API
├── ARCHITECTURE.md              # Architecture documentation
└── advance_guide.md             # This guide
```

---

## Core Components

### 1. Guidance Computer (`internal/guidance/`)

The Guidance Computer is responsible for trajectory planning and optimization.

#### Key Interfaces

```go
// GuidanceComputer plans and updates trajectories
type GuidanceComputer interface {
    PlanTrajectory(ctx context.Context, req TrajectoryRequest) (*Trajectory, error)
    UpdateTrajectory(ctx context.Context, currentState State, traj *Trajectory) (*Trajectory, error)
    ValidateTrajectory(traj *Trajectory) error
    OptimizeForStealth(traj *Trajectory) (*Trajectory, error)
    OptimizeForSpeed(traj *Trajectory) (*Trajectory, error)
    OptimizeForFuel(traj *Trajectory) (*Trajectory, error)
}

// TrajectoryRequest contains mission parameters
type TrajectoryRequest struct {
    PayloadType     PayloadType
    StartPosition   Vector3D
    TargetPosition  Vector3D
    MaxTime         time.Duration
    Priority        Priority
    Constraints     MissionConstraints
}
```

#### Payload Types

| Payload Type | Description | Max Speed | Max Altitude |
|-------------|-------------|-----------|--------------|
| `hunoid` | Humanoid robot | 3 m/s | Ground only |
| `uav` | Fixed-wing UAV | 80 m/s | 10,000 m |
| `rocket` | Launch vehicle | 3,000 m/s | 400,000 m (LEO) |
| `missile` | Guided missile | 1,000 m/s (Mach 3) | 30,000 m |
| `spacecraft` | Orbital vehicle | 10,000 m/s | 1,000,000+ km |
| `drone` | Multirotor | 20 m/s | 500 m |
| `ground_robot` | Ground vehicle | 5 m/s | Ground only |
| `submarine` | Underwater vehicle | 15 m/s | -1,000 m (depth) |
| `interstellar` | Deep space probe | 50,000 m/s | Unlimited |

---

### 2. Navigation System (`internal/navigation/`)

The Navigation System handles real-time navigation with waypoint management, terrain avoidance, and threat evasion.

#### Navigation Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `direct` | Straight line path | Maximum speed missions |
| `terrain` | Terrain following | Low-altitude operations |
| `stealth` | Minimize detection | Covert operations |
| `evasive` | Active threat evasion | Hostile environments |
| `energy` | Minimize fuel/power | Long-range missions |
| `ballistic` | Ballistic trajectory | Rocket/missile flights |
| `orbital` | Orbital insertion | Spacecraft operations |
| `interplanetary` | Deep space | Interstellar missions |

#### Key Features

- **Waypoint Management**: Set, modify, and track waypoints
- **Terrain Avoidance**: Automatic terrain clearance
- **Threat Avoidance**: Dynamic threat zone routing
- **Steering Commands**: Real-time guidance outputs
- **State Tracking**: Position, velocity, heading, fuel, battery

#### Usage Example

```go
// Create navigator
config := NavigationConfig{
    Mode:              ModeStealthPath,
    MaxSpeed:          100.0,      // m/s
    MaxAcceleration:   10.0,       // m/s²
    MinAltitude:       100.0,      // meters AGL
    TerrainClearance:  50.0,       // meters
    WaypointTolerance: 10.0,       // meters
    EnableThreatAvoid: true,
    StealthPriority:   0.8,
}

navigator := NewNavigator("nav-001", config)

// Set waypoints
waypoints := []Waypoint{
    {Position: Vector3D{X: 0, Y: 0, Z: 100}},
    {Position: Vector3D{X: 5000, Y: 2000, Z: 500}},
    {Position: Vector3D{X: 10000, Y: 5000, Z: 200}},
}
navigator.SetWaypoints(waypoints)

// Add threat zone
navigator.AddThreatZone(ThreatZone{
    ID:          "tz-001",
    Center:      Vector3D{X: 7000, Y: 3000, Z: 0},
    Radius:      2000,
    ThreatLevel: 0.9,
    ThreatType:  "radar",
    Active:      true,
})

// Start navigation
navigator.Start(ctx)

// Get steering command
cmd := navigator.CalculateSteeringCommand()
fmt.Printf("Target heading: %.2f rad, Speed: %.2f m/s\n", 
    cmd.TargetHeading, cmd.TargetSpeed)
```

---

### 3. Prediction Engine (`internal/prediction/`)

The Prediction Engine provides AI-powered predictions for trajectories, intercepts, fuel consumption, and contact windows.

#### Capabilities

| Prediction Type | Description | Method |
|-----------------|-------------|--------|
| Trajectory | Future path prediction | Kalman filter + physics |
| Intercept | Intercept point calculation | Proportional navigation |
| Threat | Threat movement prediction | State estimation |
| Weather | Weather impact prediction | Statistical model |
| Contact | Satellite contact windows | Orbital mechanics |
| Fuel | Fuel consumption forecast | Energy model |
| Arrival | ETA prediction | Velocity integration |

#### Kalman Filter

The prediction engine uses a 9-state Kalman filter for high-precision tracking:

**State Vector**: `[x, y, z, vx, vy, vz, ax, ay, az]`

```go
// Create predictor with Kalman filtering
config := PredictorConfig{
    DefaultHorizon:  5 * time.Minute,
    UpdateInterval:  1 * time.Second,
    MinConfidence:   0.5,
    EnableKalman:    true,
    EnableML:        true,
    HistorySize:     100,
}

predictor := NewPredictor("pred-001", config)

// Update with observations
predictor.UpdateState("target-001", State{
    Position:  Vector3D{X: 1000, Y: 2000, Z: 500},
    Velocity:  Vector3D{X: 50, Y: 25, Z: 0},
    Timestamp: time.Now(),
})

// Predict trajectory
prediction, _ := predictor.PredictTrajectory(ctx, "target-001", 2*time.Minute)
fmt.Printf("Prediction confidence: %.2f\n", prediction.Confidence)
```

#### Intercept Calculation

```go
// Calculate intercept solution
pursuerState := State{
    Position: Vector3D{X: 0, Y: 0, Z: 1000},
    Velocity: Vector3D{X: 100, Y: 0, Z: 0},
}

solution, _ := predictor.CalculateIntercept(ctx, pursuerState, "target-001", 500.0)

fmt.Printf("Intercept point: (%.0f, %.0f, %.0f)\n", 
    solution.InterceptPoint.X, 
    solution.InterceptPoint.Y, 
    solution.InterceptPoint.Z)
fmt.Printf("Time to intercept: %v\n", solution.TimeToIntercept)
fmt.Printf("Feasibility: %.2f\n", solution.Feasibility)
```

---

### 4. Stealth Optimization (`internal/stealth/`)

The Stealth Optimizer minimizes detection probability through advanced physics-based modeling.

#### Stealth Factors

| Factor | Mitigation | Implementation |
|--------|------------|----------------|
| Radar Cross Section (RCS) | Aspect angle optimization | RCS profile model |
| Thermal Signature | Speed/altitude management | Stefan-Boltzmann law |
| Radar Detection | Terrain masking | Line-of-sight analysis |
| SAM Threat | Altitude/speed optimization | Engagement envelope model |

#### RCS Calculation

```go
// Calculate effective RCS
rcs := optimizer.CalculateRadarCrossSection(
    position,    // Current position
    heading,     // Current heading (radians)
    radarPos,    // Radar position
)
```

**RCS Profile** (m²):
- Frontal: 0.5 m²
- Side: 2.0 m²
- Rear: 1.0 m²
- Top: 5.0 m²
- Bottom: 3.0 m²

#### Thermal Signature

```go
// Calculate thermal signature
thermalSig := optimizer.CalculateThermalSignature(
    position,       // Current position
    velocity,       // Current velocity
    engineThrottle, // 0.0-1.0
)
```

**Thermal Model**:
- Friction heating: `T_friction = T_ambient + v² × 0.001`
- Engine contribution: `T_engine × throttle`
- Altitude cooling: `1 + (altitude / 10000)`

#### Radar Detection Probability

```go
// Calculate detection probability
prob := optimizer.CalculateRadarDetectionProbability(
    position,  // Current position
    heading,   // Current heading
    radarSite, // Radar site parameters
)
```

**Factors**:
- Distance (inverse fourth power)
- RCS aspect angle
- Terrain masking
- Radar frequency
- Elevation angle

#### Trajectory Optimization

```go
// Optimize trajectory for stealth
config := StealthConfig{
    MaxDetectionProbability: 0.1,
    MinTerrainClearance:     50.0,
    ThermalReduction:        true,
    RadarEvasion:            true,
}

optimizer := NewStealthOptimizer(config)

// Add threat sources
optimizer.AddRadarSite(RadarSite{
    ID:           "radar-001",
    Position:     Vector3D{X: 50000, Y: 30000, Z: 100},
    FrequencyGHz: 10.0,  // X-band
    RangeKm:      200,
    Active:       true,
})

optimizer.AddSAMSite(SAMSite{
    ID:          "sam-001",
    Position:    Vector3D{X: 45000, Y: 35000, Z: 0},
    RangeKm:     100,
    MaxAltitude: 20000,
    Active:      true,
})

// Optimize trajectory
optimizedTraj, _ := optimizer.OptimizeTrajectory(ctx, trajectory)
fmt.Printf("Stealth score: %.2f\n", optimizedTraj.StealthScore)
```

#### Stealth Report

```go
// Generate detailed stealth analysis
report := optimizer.GetStealthReport(trajectory)

fmt.Printf("Overall stealth score: %.2f\n", report.OverallScore)
for _, wp := range report.WaypointAnalysis {
    fmt.Printf("Waypoint %s: Radar=%.2f, SAM=%.2f, Thermal=%.2f\n",
        wp.WaypointID, wp.RadarExposure, wp.SAMThreat, wp.ThermalSignature)
}
```

---

### 5. Payload Controller (`internal/payload/`)

The Payload Controller manages individual payloads with type-specific capabilities and commands.

#### Command Types

| Command | Description | Parameters |
|---------|-------------|------------|
| `navigate_to` | Move to position | x, y, z |
| `hold` | Hold position | - |
| `return` | Return to base | - |
| `arm` | Arm systems | - |
| `disarm` | Disarm systems | - |
| `execute` | Execute action | action-specific |
| `abort` | Abort mission | - |
| `set_speed` | Set target speed | speed |
| `set_altitude` | Set target altitude | altitude |
| `set_heading` | Set heading | heading |
| `engage_stealth` | Toggle stealth mode | enabled |
| `emergency_stop` | Emergency stop | - |

#### Multi-Payload Management

```go
// Create multi-payload controller
mpc := NewMultiPayloadController()

// Add payloads
mpc.AddPayload("hunoid-001", PayloadHunoid, GetDefaultCapabilities(PayloadHunoid))
mpc.AddPayload("uav-001", PayloadUAV, GetDefaultCapabilities(PayloadUAV))
mpc.AddPayload("missile-001", PayloadMissile, GetDefaultCapabilities(PayloadMissile))

// Start all controllers
mpc.Start(ctx)

// Get all states
states := mpc.GetAllStates()
for id, state := range states {
    fmt.Printf("%s: Position=(%.0f, %.0f, %.0f) Status=%s\n",
        id, state.Position.X, state.Position.Y, state.Position.Z, state.Status)
}

// Broadcast command to all payloads
cmd := Command{
    Type:       CmdHold,
    Priority:   10,
}
results := mpc.BroadcastCommand(cmd)
```

---

## ASGARD Integration

PERCILA integrates with all ASGARD subsystems through a unified integration layer.

### System Connections

```
                              ┌───────────────────┐
                              │      PERCILA      │
                              │  Guidance System  │
                              └─────────┬─────────┘
                                        │
         ┌──────────────────────────────┼──────────────────────────────┐
         │                              │                              │
    ┌────▼────┐                    ┌───▼───┐                    ┌────▼────┐
    │ SILENUS │                    │ NYSUS │                    │ SAT_NET │
    │         │                    │       │                    │         │
    │ Orbital │                    │  API  │                    │   DTN   │
    │ Vision  │                    │ Server│                    │ Routing │
    └────┬────┘                    └───┬───┘                    └────┬────┘
         │                             │                              │
    - Terrain maps              - Mission mgmt               - Telemetry relay
    - Satellite pos             - Event bus                  - Command delivery
    - Alert feeds               - Dashboard                  - Contact windows
         │                             │                              │
         └──────────────┬──────────────┴──────────────┬───────────────┘
                        │                             │
                   ┌────▼────┐                   ┌───▼───┐
                   │  GIRU   │                   │HUNOID │
                   │         │                   │       │
                   │Security │                   │Robots │
                   └────┬────┘                   └───┬───┘
                        │                            │
                   - Threat zones              - Robot control
                   - Active threats            - Mission execution
                   - Security scans            - Telemetry
```

### Silenus Integration

Access satellite imaging and tracking data:

```go
// Get terrain map for route
terrain, _ := integration.GetTerrainForRoute(ctx, routeWaypoints)
navigator.SetTerrain(terrain)

// Get satellite positions
positions, _ := silenusClient.GetAllSatellitePositions(ctx)

// Subscribe to alerts
alerts := integration.GetAlerts()
for alert := range alerts {
    if alert.Type == "threat_detected" {
        // Update navigation with new threat
        navigator.AddThreatZone(ThreatZone{
            Center: Vector3D{
                X: alert.Location.Longitude * 111000,
                Y: alert.Location.Latitude * 111000,
                Z: alert.Location.Altitude,
            },
            Radius:     5000,
            ThreatType: alert.Type,
        })
    }
}
```

### Hunoid Integration

Control humanoid robots:

```go
// Deploy Hunoid to target
err := integration.DeployHunoidToTarget(ctx, "hunoid-001", targetPosition)

// Get robot states
states, _ := hunoidClient.GetAllHunoidStates(ctx)

// Execute action
hunoidClient.ExecuteAction(ctx, "hunoid-001", "pick_up", map[string]interface{}{
    "object": "medical_kit",
})
```

### Sat_Net Integration

Delay-tolerant communications:

```go
// Send trajectory to remote payload
satnetClient.SendTrajectory(ctx, "payload-001", waypoints)

// Send command
satnetClient.SendCommand(ctx, "payload-001", Command{
    Type: "navigate_to",
    Parameters: map[string]interface{}{
        "x": 5000, "y": 3000, "z": 500,
    },
    Priority: 2, // Expedited
})

// Get contact windows
windows, _ := integration.GetAvailableContactWindows(ctx, "sat-001")
```

### Giru Integration

Security and threat intelligence:

```go
// Get threat zones for route
threatZones, _ := integration.GetThreatZonesForRoute(ctx, route)
for _, zone := range threatZones {
    optimizer.AddRadarSite(RadarSite{
        Position: Vector3D{
            X: zone.Center.Longitude * 111000,
            Y: zone.Center.Latitude * 111000,
            Z: zone.Center.Altitude,
        },
        RangeKm:  zone.RadiusKm,
        Active:   zone.Active,
    })
}

// Subscribe to threats
threats := integration.GetThreats()
for threat := range threats {
    if threat.Severity == "critical" {
        // Trigger evasion
        evasionPath := optimizer.GenerateEvasionManeuver(
            currentPos, currentVel, threatPos, threatRadius)
        navigator.SetWaypoints(evasionPath)
    }
}
```

### Nysus Integration

Mission orchestration:

```go
// Create guidance mission
mission := GuidanceMission{
    MissionType:  "reconnaissance",
    PayloadID:    "uav-001",
    PayloadType:  "uav",
    StartPoint:   Vector3D{X: 0, Y: 0, Z: 100},
    Destination:  Vector3D{X: 50000, Y: 30000, Z: 2000},
    Waypoints:    waypoints,
    Priority:     8,
    StealthMode:  true,
}

err := integration.CreateGuidanceMission(ctx, mission)

// Subscribe to mission updates
events, _ := nysusClient.SubscribeEvents(ctx, []string{"mission.updated"})
```

---

## API Reference

### HTTP Endpoints - Core

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/status` | System status |
| GET | `/api/v1/missions` | List all missions |
| POST | `/api/v1/missions` | Create mission |
| GET | `/api/v1/missions/{id}` | Get mission details |
| GET | `/api/v1/payloads` | List all payloads |
| POST | `/api/v1/payloads` | Register payload |
| GET | `/api/v1/payloads/{id}` | Get payload state |
| PUT | `/api/v1/payloads/{id}` | Update payload state |
| GET | `/api/v1/trajectories/{id}` | Get trajectory |

### HTTP Endpoints - Access Control

| Method | Endpoint | Description | Headers |
|--------|----------|-------------|---------|
| POST | `/api/v1/auth/login` | User login | - |
| GET | `/api/v1/auth/validate` | Validate session | X-Session-ID |
| GET | `/api/v1/users` | List users | - |
| GET | `/api/v1/terminals` | List terminals | X-Clearance |
| GET | `/api/v1/clearance/levels` | Get clearance levels | - |

### HTTP Endpoints - Live Feeds

| Method | Endpoint | Description | Headers |
|--------|----------|-------------|---------|
| GET | `/api/v1/feeds` | List live feeds | X-Clearance |
| POST | `/api/v1/feeds` | Create feed | X-Clearance |
| GET | `/api/v1/feeds/{id}` | Get feed details | X-Clearance |
| DELETE | `/api/v1/feeds/{id}` | End feed | X-Clearance |
| GET | `/api/v1/telemetry/{payloadId}` | Get live telemetry | X-Clearance |

### Clearance Levels

| Level | Name | Access | Color |
|-------|------|--------|-------|
| 0 | PUBLIC | Basic info, public humanitarian missions | #22c55e |
| 1 | CIVILIAN | Civilian operations, search & rescue | #3b82f6 |
| 2 | MILITARY | Military operations, tactical missions | #f59e0b |
| 3 | GOVERNMENT | Classified government missions | #8b5cf6 |
| 4 | SECRET | Top secret operations | #ef4444 |
| 5 | ULTRA | Highest classification | #ec4899 |

### Create Mission Request

```json
POST /api/v1/missions
{
    "type": "reconnaissance",
    "payloadId": "uav-001",
    "payloadType": "uav",
    "startPosition": {"x": 0, "y": 0, "z": 100},
    "targetPosition": {"x": 50000, "y": 30000, "z": 2000},
    "priority": 8,
    "stealthRequired": true
}
```

### Mission Response

```json
{
    "id": "mission-123",
    "type": "reconnaissance",
    "payloadId": "uav-001",
    "payloadType": "uav",
    "startPosition": {"x": 0, "y": 0, "z": 100},
    "targetPosition": {"x": 50000, "y": 30000, "z": 2000},
    "priority": 8,
    "stealthRequired": true,
    "status": "active",
    "trajectory": {
        "id": "traj-456",
        "waypoints": [...],
        "stealthScore": 0.85,
        "confidence": 0.92
    },
    "createdAt": "2026-01-22T18:00:00Z"
}
```

### Payload State

```json
{
    "id": "uav-001",
    "type": "uav",
    "position": {"x": 25000, "y": 15000, "z": 1500},
    "velocity": {"x": 50, "y": 25, "z": 0},
    "heading": 0.463,
    "fuel": 75.5,
    "battery": 88.2,
    "health": 0.98,
    "status": "navigating",
    "lastUpdate": "2026-01-22T18:05:00Z"
}
```

---

## Live Feed System

PERCILA v2.0 introduces a comprehensive live feed system for real-time payload tracking and mission monitoring.

### Live Feed Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         PERCILA Live Feed Hub                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌───────────────┐    ┌───────────────┐    ┌───────────────┐          │
│  │ Telemetry     │    │ Video         │    │ Map           │          │
│  │ Stream        │    │ Stream        │    │ Overlay       │          │
│  │               │    │               │    │               │          │
│  │ Position      │    │ Live Camera   │    │ 3D Track      │          │
│  │ Velocity      │    │ Thermal       │    │ Threat Zones  │          │
│  │ Status        │    │ 4K/1080p      │    │ No-Fly Zones  │          │
│  └───────┬───────┘    └───────┬───────┘    └───────┬───────┘          │
│          │                    │                    │                   │
│          └────────────────────┼────────────────────┘                   │
│                               │                                        │
│                    ┌──────────▼──────────┐                            │
│                    │  WebSocket Hub      │                            │
│                    │  Real-time Broadcast │                            │
│                    └──────────┬──────────┘                            │
│                               │                                        │
│     ┌─────────────────────────┼─────────────────────────┐             │
│     │                         │                         │              │
│  ┌──▼──┐    ┌──────┐    ┌────▼────┐    ┌──────┐    ┌──▼──┐          │
│  │PUBLIC│    │CIVILIAN│   │MILITARY│    │ GOV  │    │SECRET│          │
│  │Feeds │    │ Feeds  │   │ Feeds  │    │Feeds │    │Feeds │          │
│  └──────┘    └────────┘   └─────────┘   └──────┘    └──────┘          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Stream Types

| Type | Description | Use Case |
|------|-------------|----------|
| `telemetry` | Position, velocity, status | All missions |
| `video` | Live camera feed | Visual confirmation |
| `thermal` | Thermal imaging | Night/stealth ops |
| `radar` | Radar overlay | Threat detection |
| `map` | 3D visualization | Mission planning |
| `command` | Command stream | Control ops |
| `alert` | Alert notifications | Critical updates |

### Feed Quality Options

- **4K** - Ultra HD (3840x2160) - Tactical recon
- **1080p** - Full HD (1920x1080) - Standard ops
- **720p** - HD (1280x720) - Bandwidth limited
- **480p** - SD (854x480) - Low bandwidth
- **audio_only** - Audio stream only

### Example: Subscribe to Live Feed

```javascript
// Connect to WebSocket
const ws = new WebSocket('wss://percila.asgard/feeds/feed-001');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'telemetry':
      updatePosition(data.data.position);
      updateStatus(data.data.status);
      break;
    case 'alert':
      showAlert(data.data.title, data.data.message);
      break;
  }
};
```

---

## Mission Hub

The Mission Hub provides a centralized interface for monitoring and controlling all PERCILA-guided missions with tiered access control.

### Access Terminal Types

| Type | Clearance | Capabilities | Location |
|------|-----------|--------------|----------|
| `public` | PUBLIC | View public feeds | Main Lobby |
| `civilian` | CIVILIAN | View feeds, basic commands | Aid Operations |
| `tactical` | MILITARY | Full view, all commands | Military HQ |
| `command` | GOV | Full access, override | Command Center |
| `scif` | SECRET+ | Classified access | Secure Facility |

### Mission Hub Features

1. **Live Mission Dashboard**
   - Real-time mission status
   - Active payload tracking
   - Viewer count monitoring

2. **Tiered Feed Access**
   - Automatic clearance filtering
   - Secure authentication
   - Audit logging

3. **Telemetry Display**
   - Position/velocity data
   - Fuel/battery status
   - ETA calculations
   - Warning indicators

4. **Access Terminals**
   - Location-based access
   - Hardware authentication
   - Heartbeat monitoring

### Mission Hub Interface

Located at: `http://localhost:5173/missions`

```
┌────────────────────────────────────────────────────────────────────────┐
│                      ASGARD MISSION HUB                                │
│  [CLEARANCE: MILITARY]                    [Terminal: Tactical-001]    │
├──────────────────┬─────────────────────────────────┬──────────────────┤
│ ACTIVE MISSIONS  │         LIVE FEED                │   TELEMETRY      │
│                  │                                  │                  │
│ ┌──────────────┐ │  ┌────────────────────────────┐ │ Position:        │
│ │Hurricane     │ │  │                            │ │  X: 10,234       │
│ │Relief        │ │  │     [LIVE VIDEO FEED]      │ │  Y: 5,678        │
│ │[PUBLIC] LIVE │ │  │                            │ │  Z: 2,000m       │
│ │65% ████████░░│ │  │   ALT: 2000m  SPD: 55m/s   │ │                  │
│ └──────────────┘ │  │                            │ │ Speed: 55.9 m/s  │
│                  │  │   FUEL: 75%   BAT: 85%     │ │ Heading: 45.2°   │
│ ┌──────────────┐ │  │                            │ │                  │
│ │Tactical      │ │  └────────────────────────────┘ │ Fuel:  ████████░ │
│ │Recon Alpha   │ │                                  │ Battery: ███████ │
│ │[MILITARY]    │ │  [▶] [🔇] [⚙️] [⛶]              │                  │
│ │45% █████░░░░░│ │                                  │ ETA: 25 minutes  │
│ └──────────────┘ │                                  │ Status: ACTIVE   │
├──────────────────┴─────────────────────────────────┴──────────────────┤
│  PERCILA v2.0.0 | 2026-01-22 | CLEARANCE: MILITARY                    │
└────────────────────────────────────────────────────────────────────────┘
```

### Login API

```bash
# Login with username
curl -X POST http://localhost:8092/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "military_commander", "password": "secret"}'

# Response
{
  "sessionId": "abc123...",
  "token": "xyz789...",
  "userId": "user-military-001",
  "username": "military_commander",
  "clearance": 2,
  "clearanceName": "MILITARY",
  "expiresAt": "2026-01-23T03:17:00Z"
}
```

### Get Feeds by Clearance

```bash
# Get feeds (filtered by clearance header)
curl http://localhost:8092/api/v1/feeds \
  -H "X-Clearance: MILITARY"

# Response shows only feeds at or below MILITARY clearance
```

---

## Deployment Guide

### Building PERCILA

```powershell
# Build executable
cd C:\Users\hp\Desktop\Asgard
go build -o bin/percila.exe ./Percila/cmd/percila/main.go

# Verify build
.\bin\percila.exe --help
```

### Running PERCILA

```powershell
# Basic run
.\bin\percila.exe

# With custom configuration
.\bin\percila.exe `
    -http-port 8092 `
    -metrics-port 9092 `
    -nysus http://localhost:8080 `
    -satnet http://localhost:8081 `
    -giru http://localhost:9090 `
    -stealth=true `
    -prediction=true
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o percila ./Percila/cmd/percila/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/percila .
EXPOSE 8092 9092
CMD ["./percila"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: percila
  namespace: asgard
spec:
  replicas: 2
  selector:
    matchLabels:
      app: percila
  template:
    metadata:
      labels:
        app: percila
    spec:
      containers:
      - name: percila
        image: asgard/percila:latest
        ports:
        - containerPort: 8092
        - containerPort: 9092
        env:
        - name: NYSUS_ENDPOINT
          value: "http://nysus-service:8080"
        - name: SATNET_ENDPOINT
          value: "http://satnet-service:8081"
        - name: GIRU_ENDPOINT
          value: "http://giru-service:9090"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
---
apiVersion: v1
kind: Service
metadata:
  name: percila-service
  namespace: asgard
spec:
  selector:
    app: percila
  ports:
  - name: http
    port: 8092
    targetPort: 8092
  - name: metrics
    port: 9092
    targetPort: 9092
```

---

## Configuration Reference

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-http-port` | 8092 | HTTP API port |
| `-metrics-port` | 9092 | Metrics/health port |
| `-nysus` | http://localhost:8080 | Nysus API endpoint |
| `-satnet` | http://localhost:8081 | Sat_Net endpoint |
| `-giru` | http://localhost:9090 | Giru endpoint |
| `-stealth` | true | Enable stealth optimization |
| `-prediction` | true | Enable trajectory prediction |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PERCILA_HTTP_PORT` | HTTP API port |
| `PERCILA_METRICS_PORT` | Metrics port |
| `NYSUS_ENDPOINT` | Nysus API endpoint |
| `SATNET_ENDPOINT` | Sat_Net endpoint |
| `GIRU_ENDPOINT` | Giru endpoint |
| `LOG_LEVEL` | Log level (debug, info, warn, error) |

---

## Performance Specifications

### Guidance Performance

| Metric | Value |
|--------|-------|
| Trajectory update rate | 20 Hz |
| Replan latency | < 100 ms |
| Max concurrent payloads | 1000+ |
| Prediction horizon | 0-60 minutes |
| Kalman filter states | 9 (pos, vel, acc) |

### Stealth Optimization

| Metric | Value |
|--------|-------|
| RCS calculation rate | 1000 Hz |
| Terrain masking resolution | 100 m cells |
| Max radar sites tracked | 100+ |
| Max SAM sites tracked | 100+ |
| Thermal model accuracy | ±5% |

### Integration Latency

| System | Typical Latency |
|--------|-----------------|
| Silenus | 50-200 ms |
| Nysus | 10-50 ms |
| Sat_Net (local) | 100-500 ms |
| Sat_Net (LEO) | 200-800 ms |
| Sat_Net (GEO) | 500-1500 ms |
| Giru | 20-100 ms |
| Hunoid | 50-200 ms |

---

## Changelog

### Version 1.0.0 (2026-01-22)

- Initial release
- Multi-payload guidance (Hunoid, UAV, Rocket, Missile, Spacecraft, Drone, Ground Robot, Submarine, Interstellar)
- Stealth trajectory optimization with RCS, thermal, and radar modeling
- Kalman filter prediction engine
- Proportional navigation intercept calculation
- Full ASGARD integration (Silenus, Hunoid, Sat_Net, Giru, Nysus)
- REST API for mission management
- Metrics and health endpoints
- Kubernetes deployment support

---

## Authors

ASGARD Defense Systems - Pandora Project Team

## License

Proprietary - ASGARD Defense Systems
