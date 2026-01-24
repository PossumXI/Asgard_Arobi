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

Note: Local Docker binding uses port 55432 due to a host PostgreSQL 18
service occupying 5432.

| Property | Value |
|----------|-------|
| **Container Name** | `asgard_postgres` |
| **Image** | `postgis/postgis:15-3.3` |
| **Host** | `localhost` |
| **Port** | `55432` |
| **Database** | `asgard` |
| **Username** | `postgres` |
| **Password** | `${POSTGRES_PASSWORD}` (set in .env) |
| **SSL Mode** | `disable` (dev only) |

**Connection String:**
```
postgres://postgres:${POSTGRES_PASSWORD}@localhost:55432/asgard?sslmode=disable
```

**DSN Format (Go):**
```
host=localhost port=55432 user=postgres password=asgard_secure_2026 dbname=asgard sslmode=disable
```

**Source Files:**
- `Data/docker-compose.yml` (lines 6-8)
- `internal/platform/db/config.go` (lines 31-36)
- `Data/init_databases.ps1` (line 66)

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
| **Password** | `${MONGO_PASSWORD}` (set in .env) |
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
| **Password** | `${REDIS_PASSWORD}` (set in .env) |
| **Persistence** | AOF (appendonly yes) |
| **Protected Mode** | Enabled |

**Connection Address:**
```
redis://:asgard_redis_2026@localhost:6379
```

**Redis CLI Connection:**
```powershell
docker exec -it asgard_redis redis-cli -a $REDIS_PASSWORD
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
POSTGRES_PORT=55432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_postgres_password
POSTGRES_DB=asgard
POSTGRES_SSLMODE=disable

# MongoDB
MONGO_HOST=localhost
MONGO_PORT=27017
MONGO_USER=admin
MONGO_PASSWORD=your_secure_mongo_password
MONGO_DB=asgard

# NATS
NATS_HOST=localhost
NATS_PORT=4222

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_secure_redis_password
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
| **Current Value** | `${ASGARD_JWT_SECRET}` (set in .env, min 32 chars) |
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
docker exec -it asgard_mongodb mongosh -u admin -p $MONGO_PASSWORD
```

### View NATS Monitoring
```
http://localhost:8222
```

### Connect to Redis
```powershell
docker exec -it asgard_redis redis-cli -a $REDIS_PASSWORD
```

---

## 10. Security Notes

### Development Environment
- All credentials are hardcoded defaults for local development
- No SSL/TLS enabled
- No authentication on NATS
- Redis: Password enabled, bound to localhost only

### 2.1 Stripe API Configuration

**⚠️ IMPORTANT:** Use environment variables for Stripe keys. Never commit keys to version control.

| Property | Environment Variable | Description |
|----------|---------------------|-------------|
| **API Secret Key** | `STRIPE_SECRET_KEY` | Live or test secret key from Stripe Dashboard |
| **Webhook Secret** | `STRIPE_WEBHOOK_SECRET` | Webhook signing secret from Stripe Dashboard |
| **Success URL** | `STRIPE_SUCCESS_URL` | Redirect URL after successful checkout |
| **Cancel URL** | `STRIPE_CANCEL_URL` | Redirect URL if checkout cancelled |
| **Portal Return URL** | `STRIPE_PORTAL_RETURN_URL` | Return URL for customer portal |

**Setup Instructions:**

1. **Get your Stripe API keys:**
   - Log into [Stripe Dashboard](https://dashboard.stripe.com)
   - Go to Developers → API keys
   - Copy your **Secret key** (starts with `sk_live_` or `sk_test_`)

2. **Set environment variables:**
   ```bash
   # Windows PowerShell
   $env:STRIPE_SECRET_KEY="sk_live_YOUR_KEY_HERE"
   $env:STRIPE_WEBHOOK_SECRET="whsec_YOUR_WEBHOOK_SECRET"
   $env:STRIPE_SUCCESS_URL="https://yourdomain.com/dashboard?success=true"
   $env:STRIPE_CANCEL_URL="https://yourdomain.com/pricing"
   $env:STRIPE_PORTAL_RETURN_URL="https://yourdomain.com/dashboard"
   
   # Linux/Mac
   export STRIPE_SECRET_KEY="sk_live_YOUR_KEY_HERE"
   export STRIPE_WEBHOOK_SECRET="whsec_YOUR_WEBHOOK_SECRET"
   export STRIPE_SUCCESS_URL="https://yourdomain.com/dashboard?success=true"
   export STRIPE_CANCEL_URL="https://yourdomain.com/pricing"
   export STRIPE_PORTAL_RETURN_URL="https://yourdomain.com/dashboard"
   ```

3. **Or use .env file** (ensure it's in .gitignore):
   ```
   STRIPE_SECRET_KEY=sk_live_YOUR_KEY_HERE
   STRIPE_WEBHOOK_SECRET=whsec_YOUR_WEBHOOK_SECRET
   STRIPE_SUCCESS_URL=https://yourdomain.com/dashboard?success=true
   STRIPE_CANCEL_URL=https://yourdomain.com/pricing
   STRIPE_PORTAL_RETURN_URL=https://yourdomain.com/dashboard
   ```

4. **Configure Stripe Price IDs:**
   - In Stripe Dashboard, create Products and Prices
   - Update `PlanPriceMap` in `internal/services/stripe.go`:
     ```go
     var PlanPriceMap = map[string]string{
         "plan_observer":  "price_YOUR_OBSERVER_PRICE_ID",
         "plan_supporter": "price_YOUR_SUPPORTER_PRICE_ID",
         "plan_commander": "price_YOUR_COMMANDER_PRICE_ID",
     }
     ```

5. **Set up Webhooks:**
   - In Stripe Dashboard → Developers → Webhooks
   - Add endpoint: `https://yourdomain.com/api/webhooks/stripe`
   - Select events:
     - `checkout.session.completed`
     - `customer.subscription.updated`
     - `customer.subscription.deleted`
     - `invoice.payment_succeeded`
     - `invoice.payment_failed`
   - Copy the **Signing secret** to `STRIPE_WEBHOOK_SECRET`

**Verification:**
- The service will return an error if `STRIPE_SECRET_KEY` is not set
- Check logs for "stripe is not configured" errors
- Test with Stripe test mode first (`sk_test_...` keys)

---

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
| 2026-01-21 | Docker Monitor | Added Redis password authentication (via env var) |
| 2026-01-21 | Docker Monitor | Bound Redis to localhost only for security |
