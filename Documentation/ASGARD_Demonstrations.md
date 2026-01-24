# ASGARD Demonstration Guide

**Live Demonstrations and Use Case Scenarios**

*Last Updated: January 24, 2026*

---

## Current Demonstrables

The following ASGARD capabilities are ready for live demonstration:

### 1. Nysus - Central Orchestrator ✅

**What You Can Show**:
- REST API health checks and dashboard stats
- Real-time WebSocket event streaming
- NATS message routing and access control
- WebRTC signaling for video streams
- Multi-level access control in action

**Demo Command**:
```powershell
# Start Nysus
.\bin\nysus.exe

# In another terminal, test health
Invoke-WebRequest http://localhost:8080/health | ConvertFrom-Json

# Test WebSocket (PowerShell)
$ws = New-Object System.Net.WebSockets.ClientWebSocket
$ws.ConnectAsync("ws://localhost:8080/ws/realtime?access=public", [Threading.CancellationToken]::None).Wait()
```

**Expected Output**:
```json
{
  "status": "ok",
  "service": "nysus",
  "version": "1.0.0",
  "timestamp": "2026-01-24T08:30:00Z"
}
```

---

### 2. Giru - Live Security Scanning ✅

**What You Can Show**:
- Real-time packet capture from network interface
- Threat detection (high entropy, large packets)
- Automated mitigation actions
- NATS event publishing
- Prometheus metrics

**Demo Command**:
```powershell
# Get your interface GUID
Get-NetAdapter | ForEach-Object { Write-Host "$($_.Name) -> \Device\NPF_$($_.InterfaceGuid)" }

# Start Giru with your Wi-Fi interface
.\bin\giru.exe -interface "\Device\NPF_{YOUR-GUID-HERE}"
```

**Expected Output**:
```
=== ASGARD Giru - Security System ===
Real-time packet capture initialized
THREAT DETECTED: suspicious_payload (severity: medium, confidence: 0.65)
[Giru Publisher] Published alert to asgard.security.alerts
Mitigation action executed: rate_limit (success: true)
```

**Live Demo Tips**:
- Open a browser and navigate to various websites
- Watch Giru detect encrypted TLS traffic (high entropy)
- Show the metrics endpoint at `http://localhost:9091/metrics`

---

### 3. Integration Test Suite ✅

**What You Can Show**:
- 68 automated tests covering all subsystems
- Fast execution (< 1 second)
- Comprehensive coverage report

**Demo Command**:
```powershell
# Run all integration tests
go test ./test/integration/... -v

# Run with coverage
go test ./test/integration/... -cover
```

**Expected Output**:
```
=== RUN   TestHealthHandler
--- PASS: TestHealthHandler (0.00s)
=== RUN   TestBundleCreation
--- PASS: TestBundleCreation (0.00s)
...
PASS
ok      github.com/asgard/pandora/test/integration    0.251s
```

---

### 4. Load Testing ✅

**What You Can Show**:
- 50 concurrent WebSocket connections
- 25 concurrent WebRTC signaling sessions
- System stability under load

**Demo Command**:
```powershell
# Start Nysus first
Start-Process .\bin\nysus.exe

# Run WebSocket load test
.\scripts\load_test_realtime.ps1 -Connections 50 -Url "ws://localhost:8080/ws/realtime?access=public"

# Run signaling load test
.\scripts\load_test_signaling.ps1 -Connections 25 -Url "ws://localhost:8080/ws/signaling"
```

**Expected Output**:
```
Opening 50 WebSocket connections to ws://localhost:8080/ws/realtime
Holding connections for 5 seconds...
Realtime load test complete.
```

---

### 5. Satellite Tracking ✅

**What You Can Show**:
- Real-time ISS position tracking
- SGP4 orbit propagation
- Ground track generation

**Demo Command**:
```powershell
# Track ISS
.\bin\satellite_tracker.exe -norad 25544

# With N2YO API key for real-time comparison
$env:N2YO_API_KEY = "your-key"
.\bin\satellite_tracker.exe -norad 25544 -duration 90
```

**Expected Output**:
```
Fetching TLE for NORAD ID 25544...
Satellite: ISS (ZARYA)
Current Position (propagated):
  Latitude:  -12.0256°
  Longitude: 47.7762°
  Altitude:  415.34 km
```

---

### 6. DTN Bundle Protocol ✅

**What You Can Show**:
- Bundle creation and validation
- Routing algorithm selection
- Store-and-forward operation

**Demo Command**:
```powershell
# Verify DTN routing
.\bin\satnet_verify.exe
```

---

### 7. PERCILA Guidance System ✅

**What You Can Show**:
- Trajectory planning for various payloads
- Physics-informed neural network calculations
- Multi-agent reinforcement learning decisions

**Demo Command**:
```powershell
# Run PERCILA demo
.\scripts\percila_demo.ps1

# Or start the service directly
.\bin\percila.exe -http-port 8089
```

---

## Use Case Demonstrations

### Use Case 1: Wildfire Detection Pipeline

**Scenario**: A satellite detects a wildfire and alerts emergency services.

**Components**: Silenus → Nysus → WebSocket Clients

**Simulated Demo**:
```powershell
# Start Nysus
.\bin\nysus.exe

# In another terminal, connect as WebSocket client
# Then publish a simulated alert via NATS

# Using NATS CLI (if installed)
nats pub asgard.alerts.fire '{"type":"fire","confidence":0.92,"lat":34.05,"lon":-118.25}'
```

**Expected Flow**:
1. NATS receives fire alert
2. Nysus event bus routes to handlers
3. Alert stored in PostgreSQL
4. WebSocket clients receive real-time update
5. Dashboard shows new alert

---

### Use Case 2: Network Intrusion Response

**Scenario**: An attacker performs a port scan; Giru detects and mitigates.

**Live Demo**:
```powershell
# Start Giru
.\bin\giru.exe -interface "\Device\NPF_{YOUR-GUID}"

# In another terminal, perform a scan (against your own test system)
# Giru will detect and log the activity
```

**Expected Output**:
```
THREAT DETECTED: port_scan (severity: high)
Mitigation action executed: block_ip (success: true)
```

---

### Use Case 3: Ethical Robot Decision

**Scenario**: A Hunoid robot receives a potentially harmful command.

**Code Demo**:
```go
// In test/integration/ethics_kernel_test.go
func TestDangerousCommand() {
    kernel := ethics.NewEthicalKernel()
    
    action := &vla.Action{
        Type:       vla.ActionPickUp,
        Confidence: 0.5,  // Low confidence
        Parameters: map[string]interface{}{
            "force": 500,  // High force
        },
    }
    
    decision, _ := kernel.Evaluate(ctx, action)
    // decision.Decision = "escalated" or "rejected"
}
```

**Run Demo**:
```powershell
go test ./test/integration/... -run TestEthicsKernel -v
```

---

### Use Case 4: Interplanetary Message Routing

**Scenario**: A message needs to travel from Mars rover to Earth control.

**Simulated Demo**:
```go
// Create bundle for Mars→Earth transmission
bundle := bundle.NewBundle(
    "dtn://mars/curiosity",
    "dtn://earth/jpl",
    []byte("Telemetry data: battery=85%, temp=42C"),
)
bundle.SetPriority(bundle.PriorityExpedited)

// Route through DTN network
router := dtn.NewContactGraphRouter("dtn://mars/relay")
nextHop, _ := router.SelectNextHop(ctx, bundle, neighbors)
// nextHop = "earth-relay" or "lunar-relay"
```

---

### Use Case 5: Precision Payload Delivery

**Scenario**: Guide a supply drone to a remote location.

**PERCILA Demo**:
```powershell
# Start PERCILA
.\bin\percila.exe

# Create mission via API
$mission = @{
    payload_type = "drone"
    start = @{ lat = 40.7128; lon = -74.0060; alt = 100 }
    target = @{ lat = 40.7580; lon = -73.9855; alt = 50 }
}
Invoke-RestMethod -Uri "http://localhost:8089/api/v1/missions" -Method Post -Body ($mission | ConvertTo-Json) -ContentType "application/json"
```

---

## Demo Environment Setup

### Quick Start Script

```powershell
# demo_setup.ps1

Write-Host "Starting ASGARD Demo Environment..." -ForegroundColor Cyan

# 1. Start databases
Write-Host "Starting databases..."
Set-Location Data
docker-compose up -d
Set-Location ..
Start-Sleep -Seconds 10

# 2. Run migrations
Write-Host "Running migrations..."
.\bin\db_migrate.exe

# 3. Start Nysus
Write-Host "Starting Nysus..."
Start-Process -FilePath ".\bin\nysus.exe" -NoNewWindow

# 4. Wait for startup
Start-Sleep -Seconds 5

# 5. Verify health
$health = Invoke-RestMethod http://localhost:8080/health
Write-Host "Nysus Status: $($health.status)" -ForegroundColor Green

Write-Host "`nDemo environment ready!" -ForegroundColor Green
Write-Host "- Nysus API: http://localhost:8080"
Write-Host "- PostgreSQL: localhost:55432"
Write-Host "- MongoDB: localhost:27017"
Write-Host "- NATS: localhost:4222"
```

### Stopping the Demo

```powershell
# Stop all services
Stop-Process -Name nysus -Force -ErrorAction SilentlyContinue
Stop-Process -Name giru -Force -ErrorAction SilentlyContinue
Stop-Process -Name percila -Force -ErrorAction SilentlyContinue

# Stop databases
Set-Location Data
docker-compose down
Set-Location ..
```

---

## Demonstration Checklist

Before any demo, verify:

- [ ] Docker Desktop running
- [ ] Database containers healthy (`docker ps`)
- [ ] Npcap installed (for Giru)
- [ ] N2YO API key set (optional, for satellite tracking)
- [ ] All binaries compiled (`.\bin\*.exe`)
- [ ] Integration tests passing

---

## Investor Demo Script

### 5-Minute Executive Demo

1. **Show System Overview** (1 min)
   - Display architecture diagram
   - Explain core value proposition

2. **Live Health Check** (30 sec)
   ```powershell
   Invoke-WebRequest http://localhost:8080/health
   ```

3. **Security Scanner Demo** (2 min)
   - Start Giru
   - Show real-time threat detection
   - Demonstrate automated mitigation

4. **Integration Tests** (1 min)
   ```powershell
   go test ./test/integration/... -v
   ```
   - Show 68 tests passing

5. **Load Test** (30 sec)
   - Run 50-connection WebSocket test
   - Show system stability

### Key Talking Points

- "Production-ready with 68 integration tests"
- "Real-time threat detection and mitigation"
- "Ethical AI with 4-rule safety kernel"
- "Interplanetary networking capability"
- "Sub-meter precision guidance system"

---

## Troubleshooting

### Common Issues

**Giru fails to start**:
```
Error: couldn't load wpcap.dll
Solution: Install Npcap from https://npcap.com
```

**Nysus port already in use**:
```powershell
# Find process using port 8080
Get-NetTCPConnection -LocalPort 8080 | Select-Object OwningProcess
# Kill the process or use different port
.\bin\nysus.exe -addr :8081
```

**Database connection refused**:
```powershell
# Check Docker containers
docker ps
# Restart if needed
cd Data && docker-compose restart
```

---

*This guide enables live demonstration of all ASGARD capabilities.*
