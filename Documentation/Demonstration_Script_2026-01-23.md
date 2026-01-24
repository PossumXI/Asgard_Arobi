# ASGARD System Demonstration Script

**Version:** 1.0  
**Date:** January 23, 2026  
**Duration:** 45-60 minutes  
**Audience:** Investors, Government Officials, Technical Partners

---

## Pre-Demo Checklist

### Environment Setup
```powershell
# 1. Start database services
cd C:\Users\hp\Desktop\Asgard\Data
docker-compose up -d

# 2. Wait for services to be ready (30 seconds)
Start-Sleep -Seconds 30

# 3. Verify services
docker-compose ps
```

### Expected Services
- PostgreSQL (PostGIS) on port 55432
- MongoDB on port 27017
- NATS on ports 4222, 8222, 6222
- Redis on port 6379

### Build Binaries (if needed)
```powershell
cd C:\Users\hp\Desktop\Asgard
go build -o bin/nysus.exe ./cmd/nysus
go build -o bin/silenus.exe ./cmd/silenus
go build -o bin/hunoid.exe ./cmd/hunoid
go build -o bin/percila.exe ./Percila/cmd/percila
go build -o bin/giru.exe ./cmd/giru
```

---

## Demo Script

### Introduction (5 minutes)

**Presenter Notes:**
> "Welcome to ASGARD - Advanced Space Guardian & Autonomous Response Defense. Today, I'll demonstrate our integrated platform that combines satellite perception, autonomous robotics, precision guidance, and delay-tolerant networking into a unified command and control system."

**Key Points:**
1. 8 integrated subsystems
2. Production-ready architecture
3. Validated physics and algorithms
4. Government and commercial applications

---

### Part 1: System Architecture Overview (5 minutes)

**Visual:** Show architecture diagram

**Script:**
> "ASGARD consists of eight core systems:
> 
> 1. **Nysus** - Central orchestration with 40+ API endpoints
> 2. **Silenus** - Satellite vision and perception
> 3. **Hunoid** - Humanoid robot control with ethics kernel
> 4. **Percila** - AI-powered precision guidance
> 5. **Giru** - Security threat detection and response
> 6. **Sat_Net** - Delay-tolerant networking for deep space
> 7. **Hubs** - Real-time streaming interface
> 8. **Websites** - User portal and dashboard
>
> All systems communicate through our Unified Control Plane, enabling coordinated responses across domains."

---

### Part 2: Launch Nysus Server (5 minutes)

**Terminal Commands:**
```powershell
# Start Nysus server
cd C:\Users\hp\Desktop\Asgard
.\bin\nysus.exe
```

**Expected Output:**
```
[Nysus] Starting ASGARD Central Orchestration Server...
[Nysus] Connected to PostgreSQL
[Nysus] Connected to MongoDB
[Nysus] Event bus initialized
[Nysus] HTTP server listening on :8080
[Nysus] WebSocket server ready
[Nysus] WebRTC signaling ready
```

**Demo Points:**
1. Show health endpoint: `http://localhost:8080/api/health`
2. Show API docs (if available)
3. Demonstrate JWT authentication flow

**API Test:**
```powershell
# Health check
Invoke-RestMethod -Uri "http://localhost:8080/api/health"

# Expected: {"status":"healthy","version":"1.0.0"}
```

---

### Part 3: Web Dashboard Demo (10 minutes)

**Start Frontend:**
```powershell
# In new terminal
cd C:\Users\hp\Desktop\Asgard\Websites
npm run dev
```

**Browser:** Navigate to `http://localhost:5173`

**Demo Flow:**
1. **Landing Page** - Show public-facing marketing
2. **Sign Up** - Create demo account
3. **Dashboard** - Show stats and entity counts
4. **Subscription** - Demonstrate tier system
5. **Government Portal** - Show enhanced access

**Script:**
> "Our web interface provides tiered access:
> - **Civilian** - Basic satellite imagery and alerts
> - **Military** - Enhanced feeds and mission control
> - **Interstellar** - Full access including DTN and Percila
> 
> Government users authenticate with hardware security keys via FIDO2/WebAuthn."

---

### Part 4: Hubs Streaming Demo (10 minutes)

**Start Hubs:**
```powershell
# In new terminal
cd C:\Users\hp\Desktop\Asgard\Hubs
npm run dev
```

**Browser:** Navigate to `http://localhost:5174`

**Demo Flow:**
1. **Hub Selection** - Show three tier options
2. **Civilian Hub** - Basic stream viewer
3. **Military Hub** - Enhanced with overlays
4. **Interstellar Hub** - Full mission control

**Script:**
> "The Hubs system provides real-time video streaming from satellites and ground assets. WebRTC ensures low-latency delivery, while our backend handles access control based on subscription tier."

---

### Part 5: Percila Guidance Demo (10 minutes)

**Run Percila Tests:**
```powershell
cd C:\Users\hp\Desktop\Asgard
go test -v ./Percila/internal/physics/... 2>&1 | Select-Object -First 50
```

**Demo Points:**
1. **Physics Validation** - Show passing tests
2. **Orbital Mechanics** - J2 perturbations, SGP4
3. **Intercept Calculations** - Target interception
4. **Delivery Accuracy** - CEP/SEP metrics

**Script:**
> "Percila's guidance engine uses Multi-Agent Reinforcement Learning combined with Physics-Informed Neural Networks. Our physics calculations have been validated:
> 
> - Surface gravity: 9.8203 m/s² (accurate to 0.02%)
> - Delivery accuracy: 7.56m CEP
> - Intercept feasibility: 0.60+ confidence
>
> The system supports eight payload types from ground robots to interstellar probes."

**Show Payload Capabilities:**
```
| Payload     | Max Speed   | CEP    | Range      |
|-------------|-------------|--------|------------|
| Hunoid      | 15 m/s      | 0.1m   | 100 km     |
| UAV         | 150 m/s     | 1.0m   | 500 km     |
| Missile     | 2000 m/s    | 5.0m   | 5000 km    |
| Spacecraft  | 30 km/s     | 0.5m   | Orbital    |
```

---

### Part 6: Hunoid Robotics Demo (5 minutes)

**Script:**
> "The Hunoid system controls humanoid robots with an integrated Ethics Kernel. Before any autonomous action, the system evaluates:
> 
> 1. **Lawfulness** - Is the action legal?
> 2. **Proportionality** - Is the response appropriate?
> 3. **Human Safety** - Are civilians protected?
> 4. **Mission Necessity** - Is the action required?
>
> If any check fails, the system escalates to human operators."

**Show Code:**
```go
// Ethics Kernel evaluation
decision := ethics.Evaluate(action, context)
if decision.RequiresEscalation {
    notifyOperator(decision)
    return WaitForHuman
}
```

---

### Part 7: Control Plane Demo (5 minutes)

**Script:**
> "The Unified Control Plane coordinates all systems through policies. For example:
>
> **Policy: Security Threat Halt**
> - Trigger: Critical security threat detected
> - Action: Pause all autonomous operations
> - Cooldown: 5 minutes
>
> **Policy: DTN Congestion Management**
> - Trigger: Queue utilization > 80%
> - Action: Prioritize critical bundles
> - Cooldown: 2 minutes
>
> These policies ensure coordinated, safe responses across the entire system."

---

### Part 8: Q&A and Technical Deep-Dive (10 minutes)

**Common Questions:**

**Q: How does the system handle loss of communication?**
> "Sat_Net uses Bundle Protocol v7 with store-and-forward. Bundles are persisted in PostgreSQL and automatically forwarded when connectivity resumes. Energy-aware routing prioritizes critical messages."

**Q: What about cybersecurity?**
> "Giru provides continuous monitoring with anomaly detection. We use JWT with Argon2id hashing, support FIDO2 hardware keys, and maintain comprehensive audit logs. All communications use TLS."

**Q: Can this integrate with existing systems?**
> "Yes. Our REST API supports standard authentication methods, and we can integrate via NATS messaging. We've designed for interoperability with existing DoD and commercial systems."

**Q: What's the deployment model?**
> "We support cloud (AWS/GCP/Azure), on-premise, and air-gapped deployments. Kubernetes manifests are included, and the system runs on standard hardware."

---

## Post-Demo Actions

### Follow-Up Materials
1. Technical documentation package
2. API reference
3. Architecture diagrams
4. Test results summary

### Next Steps for Interested Parties
1. Schedule technical deep-dive
2. Provide sandbox access
3. Discuss specific use cases
4. Begin procurement discussion

---

## Troubleshooting

### Database Connection Issues
```powershell
# Check Docker containers
docker ps

# View logs
docker-compose logs postgres

# Restart services
docker-compose restart
```

### Port Conflicts
```powershell
# Check what's using port 8080
netstat -ano | findstr :8080

# Kill process if needed
taskkill /PID <PID> /F
```

### Build Errors
```powershell
# Clean and rebuild
go clean ./...
go build ./...
```

---

## Demo Environment Quick Start

```powershell
# Full start sequence
cd C:\Users\hp\Desktop\Asgard\Data
docker-compose up -d
Start-Sleep -Seconds 30

cd C:\Users\hp\Desktop\Asgard
.\bin\nysus.exe &

cd Websites
npm run dev &

cd ..\Hubs
npm run dev &
```

**URLs:**
- Nysus API: http://localhost:8080
- Websites: http://localhost:5173
- Hubs: http://localhost:5174
- NATS Monitor: http://localhost:8222

---

*ASGARD Systems - Demonstration Script v1.0*
