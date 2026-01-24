# ASGARD Runbooks

## NATS Disconnect
1. Check NATS container or service status.
2. Verify `NATS_HOST` and `NATS_PORT` environment variables.
3. Restart NATS and confirm `/api/realtime/stats` shows `connected: true`.
4. Validate WebSocket clients resume receiving events.

## PostgreSQL Degradation
1. Verify database connectivity using `psql` or `Data\init_databases.ps1`.
2. Check `POSTGRES_HOST` and `POSTGRES_PORT`.
3. Restart Postgres container or service if connection fails.
4. Confirm `/health` reports `postgres: ok`.

## MongoDB Degradation
1. Verify `mongosh` connectivity to `localhost:27017`.
2. Restart MongoDB service if required.
3. Confirm `/health` reports `mongodb: ok`.

## WebSocket Backpressure
1. Check `/api/realtime/stats` for queue saturation.
2. Scale WebSocket consumers (Nysus replicas) or reduce event rate.
3. Validate client reconnections.

## Giru Threat Spike
1. Inspect Giru logs for attack type and source.
2. Validate mitigation actions and NATS security events.
3. If needed, raise threat level and enable additional IP blocks.
