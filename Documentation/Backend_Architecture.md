# ASGARD Backend Architecture

## Overview

The ASGARD backend is built as a monolithic Go service (`cmd/nysus`) that serves as the central orchestration server. It provides REST APIs for the Websites and Hubs frontends, WebSocket support for real-time events, and WebRTC signaling for video streaming.

## Architecture Layers

### 1. HTTP Server (`cmd/nysus/main.go`)
- Main entry point for the Nysus server
- Initializes database connections (PostgreSQL, MongoDB)
- Sets up repositories, services, and handlers
- Starts HTTP server on port 8080 (configurable)
- Graceful shutdown handling

### 2. API Router (`internal/api/router.go`)
- Chi router for HTTP routing
- CORS middleware for frontend access
- Request logging and recovery middleware
- Route definitions for all API endpoints

### 3. Middleware (`internal/api/middleware/`)
- Request ID generation
- Real IP extraction
- Structured logging
- Panic recovery
- Request timeout (30s)
- Compression

### 4. Handlers (`internal/api/handlers/`)
- **AuthHandler**: Authentication endpoints (signin, signup, signout, refresh, password reset, FIDO2)
- **UserHandler**: User profile and subscription endpoints
- **SubscriptionHandler**: Stripe subscription management
- **DashboardHandler**: Dashboard stats, alerts, missions, satellites, hunoids
- **StreamHandler**: Stream listing, search, stats, WebRTC session creation

### 5. Services (`internal/services/`)
- **AuthService**: JWT token generation/validation, password hashing (Argon2id)
- **UserService**: User profile management
- **SubscriptionService**: Subscription plan management, Stripe integration (mock)
- **DashboardService**: Aggregates data from multiple repositories
- **StreamService**: Stream management and WebRTC session creation

### 6. Repositories (`internal/repositories/`)
- **UserRepository**: User CRUD operations
- **SatelliteRepository**: Satellite data access
- **HunoidRepository**: Hunoid robot data access
- **MissionRepository**: Mission data access
- **AlertRepository**: Alert data access
- **ThreatRepository**: Threat data access
- **SubscriptionRepository**: Subscription data access
- **StreamRepository**: Stream metadata and telemetry

### 7. Real-time Services
- **Broadcaster** (`internal/api/realtime/`): WebSocket event broadcasting
- **Signaling Server** (`internal/api/signaling/`): WebRTC signaling for Hubs streaming

### 8. Database Layer (`internal/platform/db/`)
- PostgreSQL connection pooling
- MongoDB connection management
- Configuration loading from environment variables

## API Endpoints

### Authentication (`/api/auth`)
- `POST /api/auth/signin` - User sign in
- `POST /api/auth/signup` - User registration
- `POST /api/auth/signout` - Sign out
- `POST /api/auth/refresh` - Refresh JWT token
- `POST /api/auth/password-reset/request` - Request password reset
- `POST /api/auth/password-reset/confirm` - Confirm password reset
- `POST /api/auth/verify-email` - Verify email address
- `POST /api/auth/fido2/register/start` - Start FIDO2 registration
- `POST /api/auth/fido2/register/complete` - Complete FIDO2 registration
- `POST /api/auth/fido2/auth/start` - Start FIDO2 authentication
- `POST /api/auth/fido2/auth/complete` - Complete FIDO2 authentication

### User (`/api/user`)
- `GET /api/user/profile` - Get user profile (protected)
- `PATCH /api/user/profile` - Update user profile (protected)
- `GET /api/user/subscription` - Get user subscription (protected)
- `PATCH /api/user/notifications` - Update notification settings (protected)

### Subscriptions (`/api/subscriptions`)
- `GET /api/subscriptions/plans` - Get available plans (protected)
- `POST /api/subscriptions/checkout` - Create Stripe checkout session (protected)
- `POST /api/subscriptions/portal` - Create billing portal session (protected)
- `POST /api/subscriptions/cancel` - Cancel subscription (protected)
- `POST /api/subscriptions/reactivate` - Reactivate subscription (protected)

### Dashboard (`/api/dashboard`)
- `GET /api/dashboard/stats` - Get dashboard statistics (protected)

### Entities (`/api/alerts`, `/api/missions`, `/api/satellites`, `/api/hunoids`)
- `GET /api/alerts` - List alerts (protected)
- `GET /api/alerts/{id}` - Get alert by ID (protected)
- `GET /api/missions` - List missions (protected)
- `GET /api/missions/{id}` - Get mission by ID (protected)
- `GET /api/satellites` - List satellites (protected)
- `GET /api/satellites/{id}` - Get satellite by ID (protected)
- `GET /api/hunoids` - List hunoids (protected)
- `GET /api/hunoids/{id}` - Get hunoid by ID (protected)

### Streams (`/api/streams`)
- `GET /api/streams` - List streams (public, supports filters)
- `GET /api/streams/{id}` - Get stream by ID (public)
- `GET /api/streams/stats` - Get stream statistics (public)
- `GET /api/streams/featured` - Get featured streams (public)
- `GET /api/streams/search?q=query` - Search streams (public)
- `POST /api/streams/{id}/session` - Create WebRTC session (optional auth)

### WebSocket (`/ws`)
- `WS /ws/realtime` - Real-time event stream
- `WS /ws/signaling` - WebRTC signaling for streams

## Authentication

- JWT tokens with 24-hour expiry
- Bearer token authentication via `Authorization` header
- Password hashing using Argon2id
- Protected routes use `RequireAuth` middleware
- Optional authentication for some public routes

## Database Schema

### PostgreSQL (Metadata)
- `users` - User accounts
- `satellites` - Satellite fleet
- `hunoids` - Humanoid robots
- `missions` - Mission assignments
- `alerts` - Alert detections
- `threats` - Security threats
- `subscriptions` - User subscriptions
- `audit_logs` - System audit trail
- `ethical_decisions` - Hunoid ethical decisions

### MongoDB (Time-series)
- `satellite_telemetry` - Satellite telemetry data
- `hunoid_telemetry` - Hunoid telemetry data
- `network_flows` - Network flow logs
- `security_events` - Security event logs

## Configuration

Environment variables:
- `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- `MONGO_HOST`, `MONGO_PORT`, `MONGO_USER`, `MONGO_PASSWORD`, `MONGO_DB`
- `NATS_HOST`, `NATS_PORT`
- `REDIS_HOST`, `REDIS_PORT`

## Dependencies

- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/go-chi/cors` - CORS middleware
- `github.com/golang-jwt/jwt/v5` - JWT tokens
- `golang.org/x/crypto/argon2` - Password hashing
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/lib/pq` - PostgreSQL driver
- `go.mongodb.org/mongo-driver` - MongoDB driver
- `github.com/google/uuid` - UUID generation

## Running the Server

```bash
# Build
go build -o bin/nysus.exe cmd/nysus/main.go

# Run
./bin/nysus.exe -port 8080

# Or with custom config
./bin/nysus.exe -port 8080 -config config.yaml
```

## Next Steps

1. **Stripe Integration**: Replace mock subscription endpoints with real Stripe API calls
2. **FIDO2 Implementation**: Complete WebAuthn/FIDO2 authentication flow
3. **Email Service**: Implement email sending for password reset and verification
4. **WebRTC Media Server**: Integrate with pion/webrtc for actual video streaming
5. **NATS Integration**: Connect real-time broadcaster to NATS for distributed events
6. **Caching**: Add Redis caching for frequently accessed data
7. **Rate Limiting**: Add rate limiting middleware
8. **Metrics**: Add Prometheus metrics endpoint
9. **API Documentation**: Generate OpenAPI/Swagger documentation
10. **Testing**: Add unit and integration tests

## Status

✅ Core HTTP server structure
✅ Database connection layer
✅ Repository pattern implementation
✅ Service layer with business logic
✅ HTTP handlers for all endpoints
✅ JWT authentication
✅ WebSocket real-time broadcasting
✅ WebRTC signaling server skeleton
✅ CORS and middleware setup

🚧 Stripe integration (mock)
🚧 FIDO2 implementation (placeholder)
🚧 Email service (placeholder)
🚧 WebRTC media server (placeholder)
