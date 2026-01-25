# Hunoid - Autonomous Humanoid Unit

## Overview
Hunoid is the physical effector of ASGARD, providing autonomous humanitarian aid and emergency response capabilities through a sophisticated mission execution system with ethical oversight and multi-robot coordination.

## Architecture
- **Reflexive Layer**: Local Go control loops for robot movement and manipulation
- **Cognitive Layer**: VLA (Vision-Language-Action) model integration for high-level planning
- **Ethical Kernel**: Runtime action validation against ethical constraints
- **Mission Runtime**: Full mission planning, intervention control, and audit logging
- **Operator Console**: Web-based UI for real-time human oversight
- **Swarm Coordinator**: Multi-robot coordination for team operations

## Directory Structure
```
Hunoid/
├── README.md                    # This file
cmd/hunoid/
├── main.go                      # Main entry point (~1700 lines)
internal/robotics/
├── control/
│   ├── interfaces.go            # Controller interfaces
│   ├── hunoid_controller.go     # Hunoid robot controller
│   ├── remote_hunoid.go         # Remote Hunoid via HTTP
│   ├── manipulator_controller.go # Gripper/arm control
│   └── remote_manipulator.go    # Remote manipulator
├── ethics/
│   └── kernel.go                # Ethical evaluation kernel
├── vla/
│   ├── interface.go             # VLA model interface
│   ├── http_vla.go              # HTTP-based VLA client
│   └── openvla.go               # OpenVLA model support
└── coordination/
    └── swarm.go                 # Multi-robot swarm coordination
```

## Features Implemented

### Mission System
- **Mission Planner**: Builds scenarios from templates
- **Mission Executor**: Step-by-step execution with decision gates
- **Step Results**: Tracking of completed, blocked, failed steps
- **Injected Commands**: Runtime command injection by operators

### Pre-Built Scenarios
| Scenario | Risk Level | Description |
|----------|------------|-------------|
| `medical_aid` | Medium | Deliver medical kit with hazard assessment |
| `perimeter_check` | Low | Validate perimeter safety and report anomalies |
| `hazard_response` | High | Investigate and mitigate hazards |

### Multi-Robot Coordination (NEW)
Swarm coordination for team operations:

**Swarm States**
- Idle, Forming, Operating, Disbanding, Emergency

**Formation Types**
| Formation | Description |
|-----------|-------------|
| Line | Robots arranged in a horizontal line |
| Column | Robots arranged in a vertical column |
| Wedge | V-shaped attack/advance formation |
| Circle | Defensive circular formation |
| Grid | Square grid pattern |
| Scatter | Random distribution within radius |

**Capabilities**
- Leader election with failover
- Heartbeat monitoring for robot health
- Automatic formation adjustment
- Coordinated mission assignment
- Emergency stop for entire swarm
- Real-time telemetry aggregation

### VLA Integration
- **Action Types**: Navigate, PickUp, PutDown, Open, Close, Inspect, Wait
- **Confidence Scoring**: Actions below threshold require approval
- **HTTP Client**: Connects to external VLA inference servers

### Ethical Kernel
- **Decision Types**: Approved, Rejected, Escalated
- **Scoring System**: Numeric ethical score with reasoning
- **Constraint Validation**: Pre-flight checks before action execution

### Safety Policy Engine
- **Battery Checks**: Blocks navigation when battery too low
- **Hazard Levels**: Higher hazard requires operator oversight
- **Consent Requirements**: Some steps require explicit approval

### Intervention System
- **Intervention Actions**: Proceed, Hold, Abort
- **Operator Timeout**: Configurable approval wait time
- **Auto-Approval**: Optional automatic approval for low-risk steps

### Operator Console (Web UI)
- **Live Status**: Mission name, objective, current step, action, confidence
- **Decision Display**: Ethics, policy, and intervention decisions
- **Controls**: Pause, Resume, Abort mission
- **Step Approval**: Manual approval for held steps
- **Command Injection**: Add new steps during mission
- **Event Log**: Recent mission events

## Build Status
**Phase: OPERATIONAL** (Full functionality including Multi-Robot Coordination)

## Usage

### Single Robot Mission
```powershell
$env:HUNOID_ENDPOINT = "http://localhost:8091"
$env:VLA_ENDPOINT = "http://localhost:8092"

go run ./cmd/hunoid/main.go -scenario medical_aid -operator-mode auto
```

### Access Operator UI
Open `http://localhost:8090` for live mission control.

### Swarm Operations
The swarm coordinator starts automatically. Additional robots can be registered via the coordinator API:
```go
swarmCoordinator.RegisterRobot("hunoid-002", Vector3{X: 5, Y: 0, Z: 0})
swarmCoordinator.SetFormation(FormationWedge)
```

### Command-Line Flags
| Flag | Default | Description |
|------|---------|-------------|
| `-id` | hunoid001 | Hunoid identifier |
| `-serial` | HND-2026-001 | Serial number |
| `-scenario` | medical_aid | Scenario to run |
| `-operator-mode` | auto | Mode: auto, manual, disabled |
| `-auto-approve-delay` | 3s | Auto-approval wait time |
| `-operator-ui` | true | Enable web UI |
| `-operator-ui-addr` | :8090 | UI server address |
| `-audit-log` | Documentation/Hunoid_Audit_Log.jsonl | Audit log path |
| `-report` | Documentation/Hunoid_Mission_Report.md | Report output |
| `-telemetry-interval` | 5s | Telemetry interval |
| `-metrics-addr` | :9092 | Metrics server address |

### Environment Variables
| Variable | Description |
|----------|-------------|
| `HUNOID_ENDPOINT` | Robot control server URL |
| `VLA_ENDPOINT` | VLA inference server URL |

### Operator Console Commands (CLI)
- `help` - Show available commands
- `status` - Show operator status
- `pause` - Pause mission execution
- `resume` - Resume paused mission
- `abort` - Abort mission immediately
- `approve <step-id>` - Approve a held step
- `inject <command>` - Add new command to mission

## Swarm Coordinator API

### Status Query
```go
status := swarmCoordinator.GetSwarmStatus()
// Returns: swarmState, formation, leaderID, robots, activeMissions
```

### Formation Control
```go
// Change formation
swarmCoordinator.SetFormation(FormationCircle)

// Get robot's formation position
pos, _ := swarmCoordinator.GetFormationPosition("hunoid-001")
```

### Mission Coordination
```go
// Create coordinated mission
mission, _ := swarmCoordinator.CreateMission(
    "Search and Rescue",
    "search",
    Area{Center: Vector3{X: 100, Y: 50, Z: 0}, Radius: 50},
    FormationGrid,
)

// Assign robots
swarmCoordinator.AssignMission(mission.ID, []string{"hunoid-001", "hunoid-002", "hunoid-003"})

// Start mission
swarmCoordinator.StartMission(mission.ID)
```

### Emergency Control
```go
// Stop all robots immediately
swarmCoordinator.EmergencyStop()
```

## Dependencies
- Go 1.21+
- VLA inference server (external)
- Robot control server (external, for real hardware)

## Integration Points
- **Nysus**: Receives mission dispatches, sends telemetry
- **Giru**: Receives threat zone data for hazard avoidance
- **Silenus**: Can receive satellite alerts for mission planning
- **Other Hunoids**: Coordinated via Swarm Coordinator
