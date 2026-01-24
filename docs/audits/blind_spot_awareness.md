| # | Status | Impact | Blind Spot | Effect | Confidence | Link to Section |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Open | High | Tokens accepted in query params | Failure Rate +6% | 80% | [BS-01](#bs-01-tokens-in-query-params) |
| 2 | Open | High | WebSocket origin checks disabled | Failure Rate +7% | 75% | [BS-02](#bs-02-websocket-origin-checks-disabled) |
| 3 | Open | High | "free" tier blocked by DB constraint | Activation -12% | 70% | [BS-03](#bs-03-free-tier-blocked-by-db-constraint) |
| 4 | Open | Med-High | Unbounded stream list pagination | Cost +9% | 65% | [BS-04](#bs-04-unbounded-stream-pagination) |
| 5 | Open | Medium | Auth writes not atomic | Support Load +6% | 60% | [BS-05](#bs-05-auth-writes-not-atomic) |
| 6 | Open | Medium | WebSocket backpressure drops messages | Retention -4% | 60% | [BS-06](#bs-06-websocket-backpressure-drops-messages) |
| 7 | Open | Medium | Logs lack correlation IDs/stack traces | Support Load +5% | 70% | [BS-07](#bs-07-missing-log-correlation-and-panic-context) |
| 8 | Open | Medium | Email verification hits DB per request | Cost +5% | 60% | [BS-08](#bs-08-email-verification-db-hit-per-request) |
| 9 | Open | Medium | No React error boundaries | Retention -3% | 65% | [BS-09](#bs-09-no-react-error-boundaries) |
| 10 | Open | Medium | Silent UI failures in chat/WS | Retention -3% | 60% | [BS-10](#bs-10-silent-ui-failures-in-chat-and-websocket) |
| 11 | Open | High | No retry or token refresh handling | Time-to-Value +8% | 70% | [BS-11](#bs-11-no-retry-or-token-refresh-handling) |
| 12 | Open | High | No data retention or backups for time-series/audit | Cost +10% | 70% | [BS-12](#bs-12-missing-data-retention-and-backups) |

# Prioritized Remediation Backlog

| Rank | Ticket ID | Focus Area | Impact | Effort | Target Sprint | Depends On |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | ASG-BS-03 | Align tier constraints | High | S | Next | None |
| 2 | ASG-BS-01 | Remove query-string tokens | High | M | Next | None |
| 3 | ASG-BS-02 | Enforce WS origin allowlist | High | M | Next | ASG-BS-01 |
| 4 | ASG-BS-11 | Add retry + token refresh | High | M | Next | ASG-BS-01 |
| 5 | ASG-BS-12 | Add retention + backups | High | L | Next+1 | None |
| 6 | ASG-BS-04 | Cap stream pagination | Med-High | S | Next+1 | None |
| 7 | ASG-BS-07 | Correlated logs + stack traces | Medium | M | Next+1 | None |
| 8 | ASG-BS-06 | WebSocket backpressure handling | Medium | M | Next+1 | ASG-BS-02 |
| 9 | ASG-BS-05 | Auth transactional writes | Medium | L | Next+2 | ASG-BS-03 |
| 10 | ASG-BS-09 | Add React error boundaries | Medium | S | Next+1 | None |
| 11 | ASG-BS-10 | Surface UI failures | Medium | S | Next+1 | None |
| 12 | ASG-BS-08 | Cache email verification | Medium | M | Next+2 | None |

# Tickets

## ASG-BS-01 Remove query-string tokens
**Summary:** Disallow URL token usage and update WebSocket/auth flows.  
**Priority:** P0  
**Owner:** Backend + Frontend  
**Scope:** `internal/api/middleware/auth.go`, `internal/api/signaling/server.go`, `Websites/src/pages/dashboard/CommandHub.tsx`  
**Acceptance Criteria:** Query-string tokens are rejected; header-based auth works for HTTP and WebSocket; tests cover both cases.  
**Prompt Link:** [Token in query params](#prompt-token-in-query-params)

## ASG-BS-02 Enforce WebSocket origin allowlist
**Summary:** Require origin validation on all WebSocket upgrades.  
**Priority:** P0  
**Owner:** Backend  
**Scope:** `internal/platform/realtime/websocket.go`, `internal/api/realtime/broadcaster.go`, `internal/api/signaling/server.go`  
**Acceptance Criteria:** Allowed origins succeed, disallowed origins fail with clear errors; configurable via env.  
**Prompt Link:** [WebSocket origin checks disabled](#prompt-websocket-origin-checks-disabled)

## ASG-BS-03 Align tier constraints
**Summary:** Align DB subscription tier constraint with app default tier.  
**Priority:** P0  
**Owner:** Backend + DB  
**Scope:** `Data/migrations/postgres/000001_initial_schema.up.sql`, `internal/services/auth.go`  
**Acceptance Criteria:** `free` tier inserts succeed; sign-up returns success; migration adds tier safely.  
**Prompt Link:** ["free" tier blocked by DB constraint](#prompt-free-tier-blocked-by-db-constraint)

## ASG-BS-04 Cap stream pagination
**Summary:** Enforce max page size for stream lists.  
**Priority:** P1  
**Owner:** Backend  
**Scope:** `internal/api/handlers/stream.go`, shared pagination helper  
**Acceptance Criteria:** Oversized `limit` returns 400; valid limits work; tests added.  
**Prompt Link:** [Unbounded stream pagination](#prompt-unbounded-stream-pagination)

## ASG-BS-05 Auth transactional writes
**Summary:** Make sign-up/sign-in state changes atomic.  
**Priority:** P1  
**Owner:** Backend  
**Scope:** `internal/services/auth.go`, repository transaction support  
**Acceptance Criteria:** No partial writes on token/email failures; tests simulate failures.  
**Prompt Link:** [Auth writes not atomic](#prompt-auth-writes-not-atomic)

## ASG-BS-06 WebSocket backpressure handling
**Summary:** Handle slow clients with disconnects or retries plus metrics.  
**Priority:** P1  
**Owner:** Backend  
**Scope:** `internal/platform/realtime/websocket.go`, `internal/api/realtime/broadcaster.go`  
**Acceptance Criteria:** Backpressure triggers explicit behavior; metrics emitted; tests cover slow client.  
**Prompt Link:** [WebSocket backpressure drops messages](#prompt-websocket-backpressure-drops-messages)

## ASG-BS-07 Correlated logs + stack traces
**Summary:** Add request IDs and panic stacks to logs.  
**Priority:** P1  
**Owner:** Backend  
**Scope:** `internal/api/middleware/logging.go`, `internal/api/middleware/recovery.go`  
**Acceptance Criteria:** Logs include request ID and user context; panic logs include stack trace.  
**Prompt Link:** [Missing log correlation and panic context](#prompt-missing-log-correlation-and-panic-context)

## ASG-BS-08 Cache email verification
**Summary:** Avoid DB hit on each request for email verification.  
**Priority:** P2  
**Owner:** Backend  
**Scope:** `internal/api/middleware/verification.go`  
**Acceptance Criteria:** Cache or claims-based verification reduces DB calls; tests verify cache hits.  
**Prompt Link:** [Email verification DB hit per request](#prompt-email-verification-db-hit-per-request)

## ASG-BS-09 Add React error boundaries
**Summary:** Provide a global error boundary in both apps.  
**Priority:** P1  
**Owner:** Frontend  
**Scope:** `Hubs/src/App.tsx`, `Websites/src/App.tsx`  
**Acceptance Criteria:** Render errors show fallback UI; test covers thrown error.  
**Prompt Link:** [No React error boundaries](#prompt-no-react-error-boundaries)

## ASG-BS-10 Surface UI failures
**Summary:** Replace silent UI failures with user-facing feedback.  
**Priority:** P1  
**Owner:** Frontend  
**Scope:** `Hubs/src/pages/StreamView.tsx`, `Websites/src/pages/dashboard/CommandHub.tsx`  
**Acceptance Criteria:** Failed chat send and WS parse errors show non-blocking alerts.  
**Prompt Link:** [Silent UI failures in chat and WebSocket](#prompt-silent-ui-failures-in-chat-and-websocket)

## ASG-BS-11 Add retry + token refresh
**Summary:** Add request retry and token refresh before logout.  
**Priority:** P0  
**Owner:** Frontend  
**Scope:** `Websites/src/lib/api.ts`, `Websites/src/providers/AuthProvider.tsx`  
**Acceptance Criteria:** Retries on transient failures; 401 triggers refresh; tests confirm no forced logout.  
**Prompt Link:** [No retry or token refresh handling](#prompt-no-retry-or-token-refresh-handling)

## ASG-BS-12 Add retention + backups
**Summary:** Implement data retention and backup/restore workflows.  
**Priority:** P0  
**Owner:** Platform/DB  
**Scope:** `Data/migrations/mongo/001_create_collections.js`, `Data/docker-compose.yml`, docs  
**Acceptance Criteria:** TTL indexes exist; backups scheduled; restore steps documented and verified.  
**Prompt Link:** [Missing data retention and backups](#prompt-missing-data-retention-and-backups)

# Project Reality Check

## Product vision and primary user loop (inferred)
- ASGARD is an autonomous defense and space operations platform coordinating satellites, robotics, and security systems, with real-time dashboards and streaming interfaces for civilian, military, and government users.
- Primary user loop appears to be: detect (Silenus) -> alert (Nysus) -> respond (Hunoid) -> monitor (Websites/Hubs dashboards) -> iterate via real-time events and control plane commands.

## Current assumptions inferred from code/config/docs
- Auth tokens can be passed in query strings for HTTP and WebSocket access.
- WebSocket origins are trusted (origin validation is disabled in multiple servers).
- The "free" subscription tier is valid in code even though the DB schema does not allow it.
- Localhost-only and placeholder credentials are acceptable defaults for runtime config.
- A single 30-second timeout is enough for all HTTP requests, including stream/session flows.
- Email verification status is evaluated per request with a fresh DB read.
- WebRTC signaling accepts sessions using query-string tokens.
- No explicit retention policy is required for time-series telemetry and audit logs.
- User-facing failures can be represented as generic error messages without in-app recovery guidance.

# Blind Spot Discovery

## BS-01 Tokens in query params
**Category:** Permission Surprise, Data Safety  
**Evidence:** `internal/api/middleware/auth.go` extracts tokens from `r.URL.Query()`, and `Websites/src/pages/dashboard/CommandHub.tsx` appends the token into the WebSocket URL query string.  
**Why it is a blind spot:** URL tokens leak to logs, browser history, proxies, and referer headers, creating a silent credential exfiltration surface.  
**Testable hypothesis:** Access logs or browser history will show bearer tokens when users access `/ws/events` or other WebSocket endpoints.

## BS-02 WebSocket origin checks disabled
**Category:** Permission Surprise, Security  
**Evidence:** `internal/platform/realtime/websocket.go`, `internal/api/realtime/broadcaster.go`, and `internal/api/signaling/server.go` all return `true` in `CheckOrigin`.  
**Why it is a blind spot:** Any origin can initiate WebSocket connections, enabling cross-site connection hijacking or CSRF-like WS abuse.  
**Testable hypothesis:** A WebSocket connection from an untrusted domain succeeds without any origin rejection.

## BS-03 "free" tier blocked by DB constraint
**Category:** Correctness Drift, Activation  
**Evidence:** `internal/services/auth.go` sets `SubscriptionTier: "free"` on sign-up, while `Data/migrations/postgres/000001_initial_schema.up.sql` restricts `subscription_tier` to `observer/supporter/commander`.  
**Why it is a blind spot:** New user creation can fail silently or throw a generic error, creating a hidden activation drop.  
**Testable hypothesis:** A sign-up request fails with a DB constraint error or returns a 500 when attempting to insert a "free" tier.

## BS-04 Unbounded stream pagination
**Category:** Cost Landmines, Reliability  
**Evidence:** `internal/api/handlers/stream.go` allows `limit` from query params without any max, while `internal/repositories/stream.go` builds `LIMIT` dynamically.  
**Why it is a blind spot:** A single request can trigger large DB scans and large JSON payloads, increasing latency and cost.  
**Testable hypothesis:** `GET /api/streams?limit=100000` results in heavy query execution and large response size.

## BS-05 Auth writes not atomic
**Category:** Correctness Drift, Data Safety  
**Evidence:** `internal/services/auth.go` updates `last_login` before token generation and creates users + email tokens + async email without transactions.  
**Why it is a blind spot:** Partial writes yield inconsistent user states (login recorded without valid token, user created without verification).  
**Testable hypothesis:** Simulated token generation failure still updates `last_login`, or email send failures leave users without verification paths.

## BS-06 WebSocket backpressure drops messages
**Category:** Timing & Concurrency, Correctness Drift  
**Evidence:** `internal/platform/realtime/websocket.go` drops outbound events when client buffers are full, and `internal/api/realtime/broadcaster.go` drops broadcasts on channel overflow.  
**Why it is a blind spot:** Users silently miss updates, causing dashboards to show stale or inconsistent data without any error.  
**Testable hypothesis:** Under load, clients receive fewer events than published with no error emitted.

## BS-07 Missing log correlation and panic context
**Category:** Observability Gaps  
**Evidence:** `internal/api/middleware/logging.go` logs without request IDs, and `internal/api/middleware/recovery.go` does not capture stack traces.  
**Why it is a blind spot:** Failures cannot be correlated across services or traced to specific requests, increasing MTTR.  
**Testable hypothesis:** A panic yields a log line with no request ID or stack trace.

## BS-08 Email verification DB hit per request
**Category:** Cost Landmines, Timing  
**Evidence:** `internal/api/middleware/verification.go` calls `userRepo.GetByID` on every request that uses the middleware.  
**Why it is a blind spot:** High-traffic endpoints force repeated DB lookups for the same user, spiking DB load with no caching.  
**Testable hypothesis:** Repeated requests from the same user show linear increases in DB query volume.

## BS-09 No React error boundaries
**Category:** UX Failure Modes  
**Evidence:** `Hubs/src/App.tsx` and `Websites/src/App.tsx` render routes without any error boundary wrapper.  
**Why it is a blind spot:** Any render exception crashes the full React tree and shows a blank screen, with no recovery UX.  
**Testable hypothesis:** Throwing an error in a child component results in a blank page with no fallback UI.

## BS-10 Silent UI failures in chat and WebSocket
**Category:** UX Failure Modes  
**Evidence:** `Hubs/src/pages/StreamView.tsx` swallows chat send errors; `Websites/src/pages/dashboard/CommandHub.tsx` swallows WS parse errors.  
**Why it is a blind spot:** Users assume actions succeeded while the system dropped them, driving invisible failure and support tickets.  
**Testable hypothesis:** Disconnecting the backend yields no error indication when sending chat or receiving invalid frames.

## BS-11 No retry or token refresh handling
**Category:** Timing & Concurrency, UX Failure Modes  
**Evidence:** `Websites/src/lib/api.ts` has no retry on network or 5xx failures; `Websites/src/providers/AuthProvider.tsx` logs users out on refresh errors.  
**Why it is a blind spot:** Transient failures become permanent sign-outs and incomplete actions, especially on mobile or flaky networks.  
**Testable hypothesis:** A single network blip during refresh logs out the user and forces a manual sign-in.

## BS-12 Missing data retention and backups
**Category:** Data Safety, Cost Landmines  
**Evidence:** `Data/migrations/mongo/001_create_collections.js` creates time-series collections with no TTL indexes, and `Data/docker-compose.yml` defines volumes without backup or retention automation. `audit_logs` has no retention policy in schema.  
**Why it is a blind spot:** Storage grows unbounded, increasing cost and recovery time; lack of backups increases data loss risk.  
**Testable hypothesis:** DB storage grows without upper bounds and there is no verified restore procedure.

# Implementation Prompts

## Prompt: Token in query params

### The Problem
Tokens passed in URLs are exposed via logs, browser history, and referrers, enabling silent credential leakage and unauthorized access.

### The Current State
`internal/api/middleware/auth.go` accepts tokens from `r.URL.Query()` and the frontend builds WebSocket URLs with `token` query params in `Websites/src/pages/dashboard/CommandHub.tsx`. WebRTC signaling uses query-string tokens in `internal/api/signaling/server.go`.

### The Goal State
Authentication tokens are only accepted via Authorization headers or secure cookies. WebSocket and signaling endpoints must require header-based or negotiated tokens, and URL tokens are rejected with explicit errors.

### A Unit Test (or UI test) to validate behavior
1. Issue a request with `?token=...` and assert a 401 response.  
2. Issue the same request with `Authorization: Bearer ...` and assert success.  
3. Establish a WebSocket connection without headers and assert it is rejected.

### The Implementation Prompt
Update the auth middleware and WebSocket handlers to disallow query-string tokens. Require tokens via `Authorization` headers or secure cookies, update frontend WebSocket clients to send headers (or switch to a signed short-lived token exchange), and add tests that ensure query tokens are rejected.

## Prompt: WebSocket origin checks disabled

### The Problem
Any origin can open WebSocket connections, enabling cross-site connection hijacking and data leakage.

### The Current State
`CheckOrigin` returns `true` in `internal/platform/realtime/websocket.go`, `internal/api/realtime/broadcaster.go`, and `internal/api/signaling/server.go`.

### The Goal State
WebSocket origins are validated against a configurable allowlist (e.g., `FRONTEND_URL` and admin portals), with explicit rejection of unknown origins.

### A Unit Test (or UI test) to validate behavior
1. Attempt a WS connection with an allowed Origin and assert success.  
2. Attempt a WS connection with a disallowed Origin and assert a 403 or upgrade failure.

### The Implementation Prompt
Add origin validation for all WebSocket upgraders using a shared helper that checks against config-driven allowlists. Ensure signaling, realtime, and broadcaster servers enforce the same logic, and add tests that validate allowed vs disallowed origins.

## Prompt: "free" tier blocked by DB constraint

### The Problem
The app sets new users to the "free" tier, but the DB schema rejects it, causing sign-up failures or silent activation drops.

### The Current State
`internal/services/auth.go` sets `SubscriptionTier: "free"` while `Data/migrations/postgres/000001_initial_schema.up.sql` allows only `observer/supporter/commander`.

### The Goal State
The DB and application agree on the allowed subscription tiers, and sign-up always succeeds with the intended default tier.

### A Unit Test (or UI test) to validate behavior
Create a user in the DB layer with `subscription_tier = 'free'` and assert the insert succeeds. Ensure sign-up returns 201 with a valid user and token.

### The Implementation Prompt
Align the database constraint and application default tier. Update the migration (or add a new migration) to allow `free`, and verify sign-up uses the same tier. Add a regression test in the auth service.

## Prompt: Unbounded stream pagination

### The Problem
Stream listing accepts unbounded `limit`, which can trigger expensive DB queries and large responses.

### The Current State
`internal/api/handlers/stream.go` sets `limit` from query params without enforcing a maximum; `internal/repositories/stream.go` builds `LIMIT` dynamically.

### The Goal State
All list endpoints enforce a strict maximum page size and return an explicit error for invalid limits.

### A Unit Test (or UI test) to validate behavior
Call `GET /api/streams?limit=100000` and assert a 400 response. Call `GET /api/streams?limit=100` and assert success with `<= 100` results.

### The Implementation Prompt
Introduce a shared pagination validator with a max limit constant. Apply it to all list endpoints, return a structured error on invalid input, and update tests to ensure oversized limits are rejected.

## Prompt: Auth writes not atomic

### The Problem
User creation, verification token storage, and login timestamps are written separately, producing partial state when downstream operations fail.

### The Current State
`internal/services/auth.go` updates `last_login` before token generation and creates users before storing email tokens with no transaction support.

### The Goal State
Auth operations are transactional: either all writes succeed or none are applied, and verification email logic is durable (e.g., outbox pattern).

### A Unit Test (or UI test) to validate behavior
Force token generation failure and assert `last_login` is unchanged. Force email token storage failure and assert the user is not persisted (or a retry is queued).

### The Implementation Prompt
Add transaction support to repositories and refactor sign-in/sign-up flows to use a single transaction. Use an outbox or retryable job for verification emails so user state and tokens are consistent.

## Prompt: WebSocket backpressure drops messages

### The Problem
Events are dropped silently when client buffers fill, causing user dashboards to become stale without any visible error.

### The Current State
`internal/platform/realtime/websocket.go` drops messages when `client.send` is full, and `internal/api/realtime/broadcaster.go` drops when the broadcast channel is full.

### The Goal State
Backpressure is handled explicitly: either slow clients are disconnected with a reason, or events are queued with bounded retries and metrics.

### A Unit Test (or UI test) to validate behavior
Simulate a slow WebSocket client and publish more messages than the buffer size. Assert that the client is disconnected with a specific close code or that a retry metric increments.

### The Implementation Prompt
Implement backpressure handling by disconnecting slow clients and emitting metrics, or by adding bounded retries with explicit error signaling. Add tests that verify behavior under load.

## Prompt: Missing log correlation and panic context

### The Problem
Logs cannot be correlated to specific requests, and panics lack stack traces, making production debugging slow and expensive.

### The Current State
`internal/api/middleware/logging.go` logs only method/path/remote/status and `internal/api/middleware/recovery.go` logs only the panic value.

### The Goal State
Every request log includes a request ID, user ID when available, and latency. Panic logs include stack traces and request context.

### A Unit Test (or UI test) to validate behavior
Trigger a handler panic and assert the log output includes a request ID and stack trace. Issue a normal request and assert the request ID is logged.

### The Implementation Prompt
Enhance logging middleware to include `middleware.RequestID` values and user IDs when present. Update recovery middleware to log stack traces and request metadata. Add tests around log output formatting.

## Prompt: Email verification DB hit per request

### The Problem
Every protected request triggers a DB read for email verification, increasing DB load and latency as traffic scales.

### The Current State
`internal/api/middleware/verification.go` fetches the user on every request to check `EmailVerified`.

### The Goal State
Email verification status is cached or embedded in token claims, and re-validated only when needed or on a TTL.

### A Unit Test (or UI test) to validate behavior
Make 100 requests with the same user and assert only one DB fetch happens within the cache TTL.

### The Implementation Prompt
Add a cache layer or embed email verification status in token claims with a short TTL. Update middleware to use cached data and fall back to DB when necessary. Add tests for cache hits.

## Prompt: No React error boundaries

### The Problem
Render-time errors in React crash the whole app without a recovery UI, causing blank screens and abandonment.

### The Current State
`Hubs/src/App.tsx` and `Websites/src/App.tsx` do not wrap routes with error boundaries.

### The Goal State
Both apps have a global error boundary that shows a fallback UI and offers a retry or navigation option.

### A Unit Test (or UI test) to validate behavior
Render a component that throws and assert the fallback UI appears instead of a blank screen.

### The Implementation Prompt
Create a shared ErrorBoundary component and wrap both app route trees. Provide a fallback UI that allows recovery and logs the error to telemetry.

## Prompt: Silent UI failures in chat and WebSocket

### The Problem
User actions fail silently in chat and command streaming, leading to lost messages and incorrect user expectations.

### The Current State
`Hubs/src/pages/StreamView.tsx` swallows chat send errors and `Websites/src/pages/dashboard/CommandHub.tsx` swallows WS parse errors.

### The Goal State
User-facing errors are surfaced with actionable messages, and retry options are provided for failed sends or corrupted frames.

### A Unit Test (or UI test) to validate behavior
Simulate a failed chat send and assert an error toast is shown. Simulate malformed WS frames and assert a non-blocking error message is displayed.

### The Implementation Prompt
Add error handling that surfaces failures in the UI (toast or inline banner) and provides a retry path. Log errors to a centralized error reporting pipeline.

## Prompt: No retry or token refresh handling

### The Problem
Transient network failures or expired tokens lead to immediate sign-outs and failed requests, degrading onboarding and retention.

### The Current State
`Websites/src/lib/api.ts` performs a single fetch with no retry, and `Websites/src/providers/AuthProvider.tsx` clears tokens on refresh errors.

### The Goal State
HTTP requests retry on transient failures, and 401 responses trigger token refresh flows before logging out users.

### A Unit Test (or UI test) to validate behavior
Mock a transient 503 on `getProfile` followed by 200 and assert the user stays signed in. Mock a 401 and assert token refresh runs before logout.

### The Implementation Prompt
Add a retry/backoff layer to the API client and implement a token refresh flow that replays the failed request. Update AuthProvider to only log out after refresh failure.

## Prompt: Missing data retention and backups

### The Problem
Time-series telemetry and audit logs grow without bounds and there is no automated backup/restore plan, risking runaway costs and data loss.

### The Current State
Mongo time-series collections are created without TTL indexes in `Data/migrations/mongo/001_create_collections.js`, and `Data/docker-compose.yml` defines volumes but no backup jobs.

### The Goal State
All time-series collections have TTL indexes or retention policies, and a documented backup/restore workflow exists with scheduled jobs and verification.

### A Unit Test (or UI test) to validate behavior
Create a document older than the TTL and assert it is removed after the TTL window. Verify backup scripts can restore to a fresh instance in a test environment.

### The Implementation Prompt
Add TTL indexes for time-series collections and add backup scripts (with schedules) for Postgres, Mongo, NATS, and Redis. Document restore steps and add a verification checklist.
