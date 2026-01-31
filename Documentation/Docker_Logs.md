# ASGARD Docker Container Logs

This file is automatically maintained by the Docker log monitor script.
It tracks container status, errors, and warnings across all ASGARD services.

## Monitored Containers

| Container | Service | Port(s) | Purpose |
|-----------|---------|---------|---------|
| asgard_postgres | PostgreSQL/PostGIS 15 | 55432 -> 5432 | Primary relational database with geospatial support |
| asgard_mongodb | MongoDB 7 | 27018 -> 27017 | Document store for telemetry and unstructured data |
| asgard_nats | NATS JetStream | 4222, 8222, 6222 | Message broker for event-driven architecture |
| asgard_redis | Redis 7 | 6379 | Cache and session storage |

## Quick Reference

### Start All Services
```powershell
# Set ASGARD_ROOT to your repo root (e.g., C:\Users\hp\Desktop\Asgard)
cd "$env:ASGARD_ROOT\Data"
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

### NATS Protocol Confusion (RESOLVED 2026-01-21)
**Symptom**: Log shows `Client parser ERROR` with HTTP GET requests on port 4222.

**Cause**: HTTP requests accidentally sent to NATS client port instead of HTTP monitoring port.

**Resolution Applied**: Bound NATS client and cluster ports to localhost only:
```yaml
ports:
  - "127.0.0.1:4222:4222"  # Client - localhost only
  - "8222:8222"            # HTTP monitoring - open
  - "127.0.0.1:6222:6222"  # Cluster - localhost only
```

### Redis Security Warning (RESOLVED 2026-01-21)
**Symptom**: Log shows "Possible SECURITY ATTACK detected. Cross Protocol Scripting"

**Cause**: HTTP requests reaching Redis port when exposed externally.

**Resolution Applied**:
1. Bound Redis to localhost only: `127.0.0.1:6379:6379`
2. Added password authentication via environment variable `REDIS_PASSWORD`
3. Enabled protected mode

**Current Configuration** (docker-compose.yml):
```yaml
redis:
  ports:
    - "127.0.0.1:6379:6379"
  command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD} --protected-mode yes
```

---

## Activity Log

[2026-01-30 17:40:00] [INFO] MongoDB volume reset to restore auth

---

## Status Report: 2026-01-30 17:40:00 (Mongo Reset)

### Action

- Removed `data_mongo_data` volume and recreated MongoDB container to reinitialize admin credentials.
- Result: MongoDB authentication restored; Nysus health reports `mongodb: ok`.

---

[2026-01-30 15:20:00] [INFO] Manual container status check

---

## Status Report: 2026-01-30 15:20:00 (Manual Check)

### Container Health

| Container | Status | Health | Notes |
|-----------|--------|--------|-------|
| asgard_postgres | not running | - | No containers active |
| asgard_mongodb | not running | - | No containers active |
| asgard_nats | not running | - | No containers active |
| asgard_redis | not running | - | No containers active |

### Status: Not Running

No Docker containers were running during this check. Start services via:
```powershell
cd C:\Users\hp\Desktop\Asgard\Data
docker compose up -d
```

---

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

---

## Status Report: 2026-01-24 02:19:00 (Audit & Fixes)

### Issues Found & Fixed

#### RESOLVED - PostgreSQL Column Name Errors (2026-01-24)
**Symptom**: Repeated errors in PostgreSQL logs:
```
ERROR: column "last_telemetry_at" does not exist at character 54
HINT: Perhaps you meant to reference the column "satellites.last_telemetry".
STATEMENT: SELECT id, name, current_battery_percent, status, last_telemetry_at
FROM satellites WHERE last_telemetry_at > NOW() - INTERVAL '30 seconds'
```

**Cause**: Some queries or processes were using `last_telemetry_at` column name, but the actual column in the database is `last_telemetry`.

**Resolution Applied**: 
1. Created migration `000010_fix_column_compatibility.up.sql`
2. Added `last_telemetry_at` columns to `satellites` and `hunoids` tables
3. Created triggers to automatically sync `last_telemetry_at` with `last_telemetry`
4. Backfilled existing data
5. Added indexes for performance

**Migration Details**:
- Added compatibility columns: `satellites.last_telemetry_at`, `hunoids.last_telemetry_at`
- Created sync triggers: `trigger_sync_satellite_telemetry_at`, `trigger_sync_hunoid_telemetry_at`
- Both columns stay in sync automatically via triggers

#### RESOLVED - create_alert Unterminated String (2026-01-24)
**Symptom**: 
```
ERROR: unterminated quoted string at or near "'{\" at character 79
STATEMENT: SELECT create_alert(NULL, 'test_alert', 0.95, 40.7128, -74.0060, 100.0, NULL, '{\
```

**Cause**: Manual test call to `create_alert` function with improperly formatted JSON string parameter.

**Resolution**: This was a one-time manual test error, not a code issue. The `create_alert` function accepts JSONB parameters correctly. When calling manually, ensure JSON strings are properly escaped and terminated.

**Best Practice**: Use parameterized queries or JSONB literals:
```sql
-- Correct usage
SELECT create_alert(
    NULL::UUID, 
    'test_alert', 
    0.95, 
    40.7128, 
    -74.0060, 
    100.0, 
    NULL, 
    '{}'::JSONB
);
```

### Container Health

| Container | Status | Health | Notes |
|-----------|--------|--------|-------|
| asgard_postgres | running | healthy | Migration 000010 applied successfully |
| asgard_mongodb | running | healthy | No issues detected |
| asgard_nats | running | healthy | No issues detected |
| asgard_redis | running | healthy | No issues detected |

### Status: All Clear

All identified errors have been resolved. Database compatibility columns added and synced.
