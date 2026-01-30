# ASGARD API Reference

**Complete REST and WebSocket API Documentation**

*Version: 1.0.0*  
*Last Updated: January 24, 2026*

---

## Base URL

```
Production: https://api.aura-genesis.org
Development: http://localhost:8080
```

## Authentication

All authenticated endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

### Access Levels

| Level | Description | Endpoints |
|-------|-------------|-----------|
| Public | No auth required | `/health`, `/api/streams` (public) |
| Civilian | Basic subscription | Dashboard, civilian alerts |
| Military | Military clearance | Tactical data, hunoid status |
| Interstellar | Commander tier | Mars/Lunar feeds |
| Government | FIDO2 + email verified | Security findings, threats |
| Admin | Full system access | All endpoints |

---

## REST API Endpoints

### Health & Status

#### GET /health

Check API health status.

**Authentication**: None

**Response**:
```json
{
  "status": "ok",
  "service": "nysus",
  "version": "1.0.0",
  "timestamp": "2026-01-24T08:30:00Z",
  "database": "connected",
  "nats": "connected"
}
```

#### GET /metrics

Prometheus metrics endpoint.

**Authentication**: None

**Response**: Prometheus text format

---

### Authentication

#### POST /api/auth/signup

Create a new user account.

**Authentication**: None

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "full_name": "John Doe"
}
```

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "usr_abc123",
      "email": "user@example.com",
      "full_name": "John Doe",
      "created_at": "2026-01-24T08:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

**Errors**:
- 400: Invalid email format, password too short, missing fields
- 409: Email already exists

---

#### POST /api/auth/signin

Authenticate existing user.

**Authentication**: None

**Request Body**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "usr_abc123",
      "email": "user@example.com",
      "role": "civilian",
      "subscription_tier": "supporter"
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

**Errors**:
- 401: Invalid credentials
- 423: Account locked

---

#### POST /api/auth/fido2/register/start

Start FIDO2/WebAuthn registration (for government users).

**Authentication**: Required

**Response**:
```json
{
  "success": true,
  "data": {
    "publicKey": {
      "challenge": "base64-encoded-challenge",
      "rp": { "name": "ASGARD Portal", "id": "aura-genesis.org" },
      "user": { "id": "...", "name": "user@example.com" },
      "pubKeyCredParams": [...]
    }
  }
}
```

---

#### POST /api/auth/fido2/auth/start

Start FIDO2 authentication.

**Authentication**: None

**Request Body**:
```json
{
  "email": "gov.user@example.com"
}
```

**Response**: WebAuthn challenge object

---

### Dashboard

#### GET /api/dashboard/stats

Get aggregate dashboard statistics.

**Authentication**: Required (Civilian+)

**Response**:
```json
{
  "success": true,
  "data": {
    "satellites": {
      "total": 12,
      "operational": 10,
      "maintenance": 2
    },
    "hunoids": {
      "total": 50,
      "active": 42,
      "idle": 8
    },
    "alerts": {
      "total": 156,
      "critical": 3,
      "unacknowledged": 12
    },
    "missions": {
      "active": 5,
      "completed_today": 23
    }
  }
}
```

---

#### GET /api/alerts

List alerts with filtering.

**Authentication**: Required (Civilian+)

**Query Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| type | string | Filter by alert type (fire, smoke, threat) |
| severity | string | Filter by severity (low, medium, high, critical) |
| status | string | Filter by status (new, acknowledged, resolved) |
| limit | int | Max results (default: 20, max: 100) |
| offset | int | Pagination offset |

**Response**:
```json
{
  "success": true,
  "data": {
    "alerts": [
      {
        "id": "alt_xyz789",
        "type": "fire",
        "severity": "high",
        "confidence": 0.92,
        "latitude": 34.0522,
        "longitude": -118.2437,
        "satellite_id": "sat_001",
        "status": "new",
        "created_at": "2026-01-24T08:15:00Z"
      }
    ]
  },
  "meta": {
    "total": 156,
    "limit": 20,
    "offset": 0
  }
}
```

---

#### GET /api/missions

List active and recent missions.

**Authentication**: Required (Military+)

**Response**:
```json
{
  "success": true,
  "data": {
    "missions": [
      {
        "id": "msn_abc123",
        "name": "Evacuation Support Alpha",
        "type": "humanitarian",
        "status": "active",
        "hunoids_assigned": ["hnd_001", "hnd_002"],
        "start_time": "2026-01-24T07:00:00Z",
        "location": {
          "latitude": 34.05,
          "longitude": -118.25
        }
      }
    ]
  }
}
```

---

#### GET /api/satellites

List satellite fleet status.

**Authentication**: Required (Civilian+)

**Response**:
```json
{
  "success": true,
  "data": {
    "satellites": [
      {
        "id": "sat_001",
        "name": "ASGARD-LEO-001",
        "norad_id": 99001,
        "status": "operational",
        "battery_percent": 87,
        "latitude": 42.3601,
        "longitude": -71.0589,
        "altitude_km": 415.2,
        "last_telemetry": "2026-01-24T08:28:00Z"
      }
    ]
  }
}
```

---

#### GET /api/hunoids

List humanoid robot fleet.

**Authentication**: Required (Military+)

**Response**:
```json
{
  "success": true,
  "data": {
    "hunoids": [
      {
        "id": "hnd_001",
        "name": "HUNOID-ALPHA-001",
        "status": "active",
        "battery_percent": 72,
        "current_mission": "msn_abc123",
        "latitude": 34.05,
        "longitude": -118.25,
        "last_command": "navigate to checkpoint-B"
      }
    ]
  }
}
```

---

### Streaming

#### GET /api/streams

List available video streams.

**Authentication**: Optional (access level determines visibility)

**Query Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| category | string | civilian, military, interstellar |
| status | string | live, offline, scheduled |
| featured | bool | Only featured streams |

**Response**:
```json
{
  "success": true,
  "data": {
    "streams": [
      {
        "id": "str_001",
        "title": "ISS Earth View",
        "category": "civilian",
        "status": "live",
        "viewer_count": 1247,
        "thumbnail_url": "https://...",
        "access_level": "public"
      }
    ]
  }
}
```

---

#### GET /api/streams/:id

Get stream details.

**Authentication**: Depends on stream access level

**Response**:
```json
{
  "success": true,
  "data": {
    "stream": {
      "id": "str_001",
      "title": "ISS Earth View",
      "description": "Live view from ISS cupola",
      "category": "civilian",
      "status": "live",
      "started_at": "2026-01-24T00:00:00Z",
      "viewer_count": 1247,
      "playback_url": "wss://..."
    }
  }
}
```

---

#### POST /api/streams/:id/session

Create WebRTC session for stream viewing.

**Authentication**: Depends on stream access level

**Response**:
```json
{
  "success": true,
  "data": {
    "session_id": "sess_xyz789",
    "signaling_url": "wss://api.aura-genesis.org/ws/signaling",
    "ice_servers": [
      { "urls": "stun:stun.aura-genesis.org:3478" }
    ]
  }
}
```

---

### Subscriptions

#### GET /api/subscriptions/plans

List available subscription plans.

**Authentication**: None

**Response**:
```json
{
  "success": true,
  "data": {
    "plans": [
      {
        "id": "plan_observer",
        "name": "Observer",
        "tier": "observer",
        "price": 9.99,
        "interval": "month",
        "features": [
          "Live civilian feeds",
          "Basic alerts",
          "Dashboard access"
        ]
      },
      {
        "id": "plan_supporter",
        "name": "Supporter",
        "tier": "supporter",
        "price": 29.99,
        "interval": "month",
        "features": [
          "All Observer features",
          "Military feeds",
          "Priority alerts"
        ]
      },
      {
        "id": "plan_commander",
        "name": "Commander",
        "tier": "commander",
        "price": 99.99,
        "interval": "month",
        "features": [
          "All Supporter features",
          "Interstellar feeds",
          "Mission requests",
          "API access"
        ]
      }
    ]
  }
}
```

---

#### POST /api/subscriptions/checkout

Create Stripe checkout session.

**Authentication**: Required

**Request Body**:
```json
{
  "plan_id": "plan_supporter"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "checkout_url": "https://checkout.stripe.com/..."
  }
}
```

---

### Satellite Tracking

#### GET /api/satellites/position

Get propagated satellite position.

**Authentication**: Required (Civilian+)

**Query Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| norad_id | int | NORAD catalog ID |

**Response**:
```json
{
  "success": true,
  "data": {
    "satellite_id": 25544,
    "name": "ISS (ZARYA)",
    "latitude": -12.0256,
    "longitude": 47.7762,
    "altitude_km": 415.34,
    "velocity_km_s": 7.66,
    "timestamp": "2026-01-24T08:30:00Z"
  }
}
```

---

#### GET /api/satellites/groundtrack

Generate orbital ground track.

**Query Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| norad_id | int | NORAD catalog ID |
| duration | int | Minutes to project (max: 180) |

**Response**:
```json
{
  "success": true,
  "data": {
    "satellite_id": 25544,
    "track": [
      { "lat": -12.02, "lon": 47.77, "alt": 415.3, "time": "..." },
      { "lat": -10.15, "lon": 50.12, "alt": 415.5, "time": "..." }
    ]
  }
}
```

---

## WebSocket API

### WS /ws/realtime

Real-time event stream with access control.

**Connection**:
```
ws://localhost:8080/ws/realtime?access=civilian&token=<jwt>
```

**Incoming Messages**:
```json
{
  "type": "alert",
  "data": {
    "id": "alt_xyz789",
    "type": "fire",
    "severity": "high"
  }
}
```

**Message Types**:
- `alert` - New alert
- `telemetry` - Satellite telemetry update
- `mission_update` - Mission status change
- `threat` - Security threat (Government+)

**Outgoing Messages**:
```json
{
  "type": "ping"
}
```

---

### WS /ws/signaling

WebRTC signaling for video streams.

**Connection**:
```
ws://localhost:8080/ws/signaling
```

**Join Stream**:
```json
{
  "type": "join",
  "session_id": "sess_xyz789",
  "stream_id": "str_001"
}
```

**SDP Offer/Answer**:
```json
{
  "type": "offer",
  "sdp": "v=0\r\no=..."
}
```

**ICE Candidate**:
```json
{
  "type": "candidate",
  "candidate": "candidate:..."
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| AUTH_REQUIRED | 401 | Authentication required |
| INVALID_TOKEN | 401 | JWT token invalid or expired |
| ACCESS_DENIED | 403 | Insufficient access level |
| NOT_FOUND | 404 | Resource not found |
| VALIDATION_ERROR | 400 | Request validation failed |
| RATE_LIMITED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Server error |

---

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| `/api/auth/*` | 10 requests/minute |
| `/api/dashboard/*` | 60 requests/minute |
| `/api/streams/*` | 30 requests/minute |
| WebSocket | 100 messages/minute |

---

## SDK Examples

### JavaScript/TypeScript

```typescript
import { AsgardClient } from '@asgard/sdk';

const client = new AsgardClient({
  baseUrl: 'https://api.aura-genesis.org',
  token: 'your-jwt-token'
});

// Get alerts
const alerts = await client.alerts.list({ severity: 'high' });

// Connect to real-time events
client.realtime.connect('civilian', (event) => {
  console.log('Event:', event);
});
```

### Go

```go
import "github.com/asgard/pandora/pkg/client"

client := client.New("https://api.aura-genesis.org", "your-jwt-token")

alerts, err := client.Alerts.List(ctx, client.AlertFilter{
    Severity: "high",
})
```

### cURL

```bash
# Get health
curl https://api.aura-genesis.org/health

# Authenticate
curl -X POST https://api.aura-genesis.org/api/auth/signin \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"..."}'

# Get alerts (authenticated)
curl https://api.aura-genesis.org/api/alerts \
  -H "Authorization: Bearer <token>"
```

---

*This API reference covers all public endpoints of the ASGARD platform.*
