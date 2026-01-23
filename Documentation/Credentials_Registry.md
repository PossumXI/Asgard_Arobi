# ASGARD Credentials Registry

> **SECURITY NOTICE**: This document contains sensitive credentials for development environments.
> DO NOT commit to public repositories. Add to `.gitignore` in production.

---

## Document Control

| Field | Value |
|-------|-------|
| Created | 2026-01-21 |
| Created By | Audit Agent |
| Last Updated | 2026-01-21 |
| Classification | DEVELOPMENT ONLY |

---

## 1. Database Credentials

### 1.1 PostgreSQL (PostGIS)

| Property | Value |
|----------|-------|
| **Container Name** | `asgard_postgres` |
| **Image** | `postgis/postgis:15-3.3` |
| **Host** | `localhost` |
| **Port** | `5432` |
| **Database** | `asgard` |
| **Username** | `postgres` |
| **Password** | `asgard_secure_2026` |
| **SSL Mode** | `disable` (dev only) |

**Connection String:**
```
postgres://postgres:asgard_secure_2026@localhost:5432/asgard?sslmode=disable
```

**DSN Format (Go):**
```
host=localhost port=5432 user=postgres password=asgard_secure_2026 dbname=asgard sslmode=disable
```

**Source Files:**
- `Data/docker-compose.yml` (lines 6-8)
- `internal/platform/db/config.go` (lines 31-36)
- `Data/init_databases.ps1` (line 64)

---

### 1.2 MongoDB

| Property | Value |
|----------|-------|
| **Container Name** | `asgard_mongodb` |
| **Image** | `mongo:7` |
| **Host** | `localhost` |
| **Port** | `27017` |
| **Database** | `asgard` |
| **Username** | `admin` |
| **Password** | `asgard_mongo_2026` |
| **Auth DB** | `admin` |

**Connection URI:**
```
mongodb://admin:asgard_mongo_2026@localhost:27017
```

**Source Files:**
- `Data/docker-compose.yml` (lines 24-25)
- `internal/platform/db/config.go` (lines 38-42)
- `Data/init_databases.ps1` (line 69)

---

### 1.3 NATS (Message Queue)

| Property | Value |
|----------|-------|
| **Container Name** | `asgard_nats` |
| **Image** | `nats:latest` |
| **Host** | `localhost` |
| **Client Port** | `4222` |
| **HTTP Monitoring** | `8222` |
| **Cluster Routing** | `6222` |
| **JetStream** | Enabled |
| **Authentication** | None (dev) |

**Connection URI:**
```
nats://localhost:4222
```

**Health Check URL:**
```
http://localhost:8222/healthz
```

**Source Files:**
- `Data/docker-compose.yml` (lines 37-53)
- `internal/platform/db/config.go` (lines 44-45)

---

### 1.4 Redis (Cache)

| Property | Value |
|----------|-------|
| **Container Name** | `asgard_redis` |
| **Image** | `redis:7-alpine` |
| **Host** | `localhost` (bound to 127.0.0.1 only) |
| **Port** | `6379` |
| **Password** | `asgard_redis_2026` |
| **Persistence** | AOF (appendonly yes) |
| **Protected Mode** | Enabled |

**Connection Address:**
```
redis://:asgard_redis_2026@localhost:6379
```

**Redis CLI Connection:**
```powershell
docker exec -it asgard_redis redis-cli -a asgard_redis_2026
```

**Source Files:**
- `Data/docker-compose.yml` (lines 55-67)
- `internal/platform/db/config.go` (lines 47-49)

---

## 2. API Endpoints Configuration

### 2.1 Backend API Server (Planned)

| Property | Value |
|----------|-------|
| **Host** | `localhost` |
| **Port** | `8080` |
| **Base URL** | `http://localhost:8080/api` |
| **WebSocket URL** | `ws://localhost:8080/ws` |

**Source Files:**
- `Websites/vite.config.ts` (line 16)
- `Hubs/vite.config.ts` (lines 16, 20)

### 2.2 Frontend Development Servers

| App | Port | URL |
|-----|------|-----|
| Websites | 5173 (default) | `http://localhost:5173` |
| Hubs | 5174 (default) | `http://localhost:5174` |

---

## 3. Environment Variables Reference

### 3.1 Go Backend Environment Variables

```bash
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=asgard_secure_2026
POSTGRES_DB=asgard
POSTGRES_SSLMODE=disable

# MongoDB
MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_USER=admin
MONGO_PASSWORD=asgard_mongo_2026
MONGO_DB=asgard

# NATS
NATS_HOST=localhost
NATS_PORT=4222

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=asgard_redis_2026
```

### 3.2 Frontend Environment Variables

```bash
# Websites (.env)
VITE_API_URL=http://localhost:8080/api

# Hubs (.env)
VITE_API_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws
```

---

## 4. Docker Network

| Property | Value |
|----------|-------|
| **Network Name** | `asgard_network` |
| **Driver** | `bridge` (default) |

**Container Hostnames (internal):**
- `postgres` → PostgreSQL
- `mongodb` → MongoDB
- `nats` → NATS
- `redis` → Redis

---

## 5. Data Volumes

| Volume Name | Purpose | Mount Path |
|-------------|---------|------------|
| `postgres_data` | PostgreSQL data | `/var/lib/postgresql/data` |
| `mongo_data` | MongoDB data | `/data/db` |
| `nats_data` | NATS JetStream | `/data` |
| `redis_data` | Redis AOF | `/data` |

---

## 6. Application Secrets

### 6.1 JWT Secret (Authentication)

| Property | Value |
|----------|-------|
| **Location** | `internal/services/auth.go` line 36 |
| **Current Value** | `asgard_jwt_secret_change_in_production_2026` |
| **Token Expiry** | 24 hours |
| **Algorithm** | HS256 |

**SECURITY NOTE**: This secret is hardcoded for development. In production:
1. Generate a secure 256-bit random secret
2. Store in environment variable `JWT_SECRET`
3. Never commit to version control

### 6.2 Password Hashing

| Property | Value |
|----------|-------|
| **Algorithm** | Argon2id |
| **Memory** | 64 KB |
| **Iterations** | 1 |
| **Parallelism** | 4 |
| **Salt Length** | 16 bytes |
| **Hash Length** | 32 bytes |

---

## 7. Agent-Created Accounts

### 7.1 Database Service Accounts

| Service | Account Type | Username | Created By | Date |
|---------|--------------|----------|------------|------|
| PostgreSQL | Superuser | `postgres` | Initial Agent | 2026-01-20 |
| MongoDB | Root Admin | `admin` | Initial Agent | 2026-01-20 |

### 7.2 Application Users (To Be Created)

| User Type | Purpose | Status |
|-----------|---------|--------|
| API Service Account | Backend API to DB | PENDING |
| Stripe Webhook | Payment processing | PENDING |
| FIDO2 Service | WebAuthn for Gov Portal | PENDING |

---

## 8. Credential Rotation Schedule

| Credential | Environment | Rotation |
|------------|-------------|----------|
| PostgreSQL password | Development | Never (local) |
| PostgreSQL password | Production | 90 days |
| MongoDB password | Development | Never (local) |
| MongoDB password | Production | 90 days |
| API tokens | All | 30 days |
| JWT secrets | All | On deployment |

---

## 8. Quick Start Commands

### Start All Services
```powershell
cd C:\Users\hp\Desktop\Asgard\Data
docker-compose up -d
```

### Check Service Status
```powershell
docker-compose ps
```

### Connect to PostgreSQL
```powershell
docker exec -it asgard_postgres psql -U postgres -d asgard
```

### Connect to MongoDB
```powershell
docker exec -it asgard_mongodb mongosh -u admin -p asgard_mongo_2026
```

### View NATS Monitoring
```
http://localhost:8222
```

### Connect to Redis
```powershell
docker exec -it asgard_redis redis-cli -a asgard_redis_2026
```

---

## 10. Security Notes

### Development Environment
- All credentials are hardcoded defaults for local development
- No SSL/TLS enabled
- No authentication on NATS
- Redis: Password enabled, bound to localhost only

### Production Requirements (TODO)
- [ ] Generate unique passwords for each environment
- [ ] Enable SSL/TLS on all connections
- [ ] Configure NATS authentication
- [x] Configure Redis password (completed 2026-01-21)
- [ ] Use secrets management (Vault, AWS Secrets Manager)
- [ ] Implement credential rotation
- [ ] Set up audit logging for all access

---

## Changelog

| Date | Author | Change |
|------|--------|--------|
| 2026-01-21 | Audit Agent | Initial document creation |
| 2026-01-21 | Audit Agent | Added all database credentials from agent work |
| 2026-01-21 | Docker Monitor | Added Redis password authentication (asgard_redis_2026) |
| 2026-01-21 | Docker Monitor | Bound Redis to localhost only for security |
