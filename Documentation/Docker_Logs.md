# ASGARD Docker Container Logs

This file is automatically maintained by the Docker log monitor script.
It tracks container status, errors, and warnings across all ASGARD services.

## Monitored Containers

| Container | Service | Port(s) | Purpose |
|-----------|---------|---------|---------|
| asgard_postgres | PostgreSQL/PostGIS 15 | 5432 | Primary relational database with geospatial support |
| asgard_mongodb | MongoDB 7 | 27017 | Document store for telemetry and unstructured data |
| asgard_nats | NATS JetStream | 4222, 8222, 6222 | Message broker for event-driven architecture |
| asgard_redis | Redis 7 | 6379 | Cache and session storage |

## Quick Reference

### Start All Services
```powershell
cd C:\Users\hp\Desktop\Asgard\Data
docker compose up -d
```

### View Live Logs
```powershell
# All containers
docker compose logs -f

# Specific container
docker logs -f asgard_postgres
docker logs -f asgard_mongodb
docker logs -f asgard_nats
docker logs -f asgard_redis
```

### Check Health Status
```powershell
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

### Restart Unhealthy Container
```powershell
docker restart asgard_nats
```

## Known Issues & Fixes

### NATS Healthcheck (RESOLVED 2026-01-21)
**Symptom**: NATS container shows as "unhealthy" despite working correctly.

**Cause**: NATS image is minimal/distroless - no shell or wget available for healthcheck commands.

**Resolution Applied**: Removed healthcheck from docker-compose.yml. Monitor NATS health externally via:
```bash
curl http://localhost:8222/healthz
```

### Redis Security Warning (RESOLVED 2026-01-21)
**Symptom**: Log shows "Possible SECURITY ATTACK detected. Cross Protocol Scripting"

**Cause**: HTTP requests reaching Redis port when exposed externally.

**Resolution Applied**:
1. Bound Redis to localhost only: `127.0.0.1:6379:6379`
2. Added password authentication: `asgard_redis_2026`
3. Enabled protected mode

**Current Configuration** (docker-compose.yml):
```yaml
redis:
  ports:
    - "127.0.0.1:6379:6379"
  command: redis-server --appendonly yes --requirepass asgard_redis_2026 --protected-mode yes
```

---

## Activity Log

[2026-01-21 04:45:00] [INFO] Docker log monitoring initialized
[2026-01-21 04:45:00] [INFO] Monitoring 4 ASGARD containers

---

## Status Report: 2026-01-21 04:45:00 (Initial Scan)

### Container Health

| Container | Status | Health | Notes |
|-----------|--------|--------|-------|
| asgard_postgres | running | healthy | PostgreSQL ready, accepting connections |
| asgard_mongodb | running | healthy | MongoDB ready, healthcheck passing |
| asgard_nats | running | healthy | Server working (healthcheck removed - minimal image) |
| asgard_redis | running | healthy | Running with AOF persistence, password protected |

### Issues Detected & Resolved

#### RESOLVED - asgard_redis Security
- **Issue**: `Possible SECURITY ATTACK detected. Cross Protocol Scripting`
- **Fix Applied**: Bound to localhost only + password authentication enabled

#### RESOLVED - asgard_nats Healthcheck
- **Issue**: Container marked unhealthy due to missing shell in minimal image
- **Fix Applied**: Healthcheck removed, monitor externally via HTTP endpoint

---
[2026-01-20 23:56:34] [INFO] Starting Docker log scan at 2026-01-20 23:56:34
[2026-01-20 23:56:37] [INFO] Containers: 4 running, 0 unhealthy

---

## Status Report: 2026-01-20 23:56:39

### Container Health

| Container | Status | Health | Uptime |
|-----------|--------|--------|--------|| asgard_redis | running | OK | 0h 6m |
| asgard_mongodb | running | OK | 0h 6m |
| asgard_nats | running | - | 0h 1m |
| asgard_postgres | running | OK | 0h 6m |

### Status: All Clear

No critical errors or warnings detected in this monitoring interval.

[2026-01-20 23:56:39] [SUCCESS] SUMMARY: All containers healthy, no issues detected
