# ASGARD Quick Start Guide

**Get ASGARD Running in 5 Minutes**

*Last Updated: January 24, 2026*

---

## Prerequisites

- **Go 1.24+**: https://go.dev/dl/
- **Docker Desktop**: https://docker.com/products/docker-desktop
- **Npcap** (for Giru): https://npcap.com/#download
- **PowerShell 7+** (recommended)

---

## Step 1: Clone and Build

```powershell
# Clone repository
git clone https://github.com/asgard/pandora.git
cd pandora

# Build all binaries
go build -o bin/nysus.exe ./cmd/nysus
go build -o bin/giru.exe ./cmd/giru
go build -o bin/percila.exe ./Percila/cmd/percila
go build -o bin/db_migrate.exe ./cmd/db_migrate

# Verify builds
Get-ChildItem bin/*.exe
```

---

## Step 2: Start Databases

```powershell
cd Data
docker-compose up -d

# Wait for containers to be healthy
docker ps

# Expected output:
# asgard_postgres   Up (healthy)
# asgard_mongodb    Up (healthy)
# asgard_nats       Up
# asgard_redis      Up (healthy)
```

---

## Step 3: Run Migrations

```powershell
cd ..
.\bin\db_migrate.exe

# Expected output:
# Connected to PostgreSQL
# Running migrations...
# Migration complete
```

---

## Step 4: Start Nysus

```powershell
.\bin\nysus.exe

# Expected output:
# === ASGARD Nysus - Central Nervous System ===
# PostgreSQL connected successfully
# MongoDB connected successfully
# Nysus is ready and accepting connections
```

---

## Step 5: Verify Health

Open a new terminal:

```powershell
# Check API health
Invoke-WebRequest http://localhost:8080/health | ConvertFrom-Json

# Expected response:
# status    : ok
# service   : nysus
# version   : 1.0.0
```

---

## Step 6: Run Integration Tests

```powershell
go test ./test/integration/... -v

# Expected output:
# PASS
# ok      github.com/asgard/pandora/test/integration    0.251s
```

---

## Optional: Start Giru (Security Scanner)

```powershell
# Find your network interface
Get-NetAdapter | ForEach-Object { 
    Write-Host "$($_.Name) -> \Device\NPF_$($_.InterfaceGuid)" 
}

# Start Giru with your interface
.\bin\giru.exe -interface "\Device\NPF_{YOUR-GUID-HERE}"
```

---

## Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| Nysus API | 8080 | HTTP/WS |
| Giru Metrics | 9091 | HTTP |
| PERCILA API | 8089 | HTTP |
| PostgreSQL | 55432 | TCP |
| MongoDB | 27017 | TCP |
| NATS | 4222 | TCP |
| Redis | 6379 | TCP |

---

## Stopping Services

```powershell
# Stop Go services
Stop-Process -Name nysus,giru,percila -Force -ErrorAction SilentlyContinue

# Stop databases
cd Data
docker-compose down
```

---

## Next Steps

- Read [System Overview](ASGARD_System_Overview.md) for architecture understanding
- Check [API Reference](ASGARD_API_Reference.md) for endpoint details
- Try [Demonstrations](ASGARD_Demonstrations.md) for live demos
- Review [Integration Report](ASGARD_Integration_Report.md) for test results

---

## Troubleshooting

### Port 8080 already in use

```powershell
# Find process using port
Get-NetTCPConnection -LocalPort 8080 | Select-Object OwningProcess

# Use different port
.\bin\nysus.exe -addr :8081
```

### Database connection refused

```powershell
# Check Docker containers
docker ps

# Restart if needed
cd Data && docker-compose restart
```

### Giru can't capture packets

```
Error: couldn't load wpcap.dll
```

Install Npcap from https://npcap.com with "WinPcap API-compatible Mode" enabled.

---

*You're now ready to explore ASGARD!*
