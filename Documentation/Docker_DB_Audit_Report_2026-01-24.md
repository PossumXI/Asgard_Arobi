## Docker + Database Audit Report (2026-01-24)

### Scope
- Docker container health and recent logs
- PostgreSQL, MongoDB, Redis, NATS operational status
- Error analysis and fixes applied

### Container Health Snapshot

| Container | Status | Healthcheck |
|-----------|--------|-------------|
| `asgard_postgres` | Running | healthy |
| `asgard_mongodb` | Running | healthy |
| `asgard_redis` | Running | healthy |
| `asgard_nats` | Running | no healthcheck |

### PostgreSQL (postgis/postgis:15-3.3)

**Findings**
- Repeated errors observed in logs referencing missing column `last_telemetry_at` on `satellites` and `hunoids`.
- Single error for an unterminated quoted string (manual query likely).
- Single syntax error from attempted inline function/trigger creation (missing `$$` delimiter).
- Routine checkpoint logs otherwise normal.

**Root Cause**
- Legacy queries reference `last_telemetry_at` directly on base tables.
- Compatibility columns/triggers not present when those queries executed.

**Fix Applied**
- Ran `.\bin\db_migrate.exe` to apply `000010_fix_column_compatibility.up.sql`:
  - Adds `last_telemetry_at` columns to `satellites` and `hunoids`
  - Adds triggers to sync `last_telemetry_at` with `last_telemetry`
  - Backfills existing data + adds indexes

**Status**
- Compatibility columns now present; legacy queries should succeed going forward.

### MongoDB (mongo:7)

**Findings**
- Repeated `Connection not authenticating` entries.

**Assessment**
- Expected with unauthenticated `mongosh` healthcheck probes in Docker logs.
- No action required unless strict auth logging is desired.

### Redis (redis:7-alpine)

**Findings**
- Normal startup logs and graceful shutdown entries.
- No errors observed.

### NATS (nats:latest)

**Findings**
- Normal JetStream startup logs.
- Healthcheck disabled by design (minimal image).

**Assessment**
- Healthy per container uptime; use `http://localhost:8222/healthz` if needed.

### Actions Taken
- Applied database compatibility migration (`000010_fix_column_compatibility`) via `.\bin\db_migrate.exe`.

### Remaining Recommendations
- Identify and update any legacy query sources still selecting `last_telemetry_at` directly from base tables.
- If desired, replace unauthenticated MongoDB probes with authenticated healthcheck to reduce log noise.
- Consider adding a lightweight NATS health probe in monitoring (HTTP `/healthz`).
