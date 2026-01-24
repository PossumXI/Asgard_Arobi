# ASGARD Release Procedure

## Pre-Release
1. Run `.\scripts\integration_test.ps1`.
2. Run `.\scripts\security_scan.ps1` and review outputs in `Documentation/`.
3. Verify build artifacts by compiling core binaries:
   - `go build -o bin\nysus.exe cmd\nysus\main.go`
   - `go build -o bin\giru.exe cmd\giru\main.go`
   - `go build -o bin\hunoid.exe cmd\hunoid\main.go`
   - `go build -o bin\silenus.exe cmd\silenus\main.go`

## Staging Deployment (Local Kubernetes)
1. Apply manifests: `cd Control_net; .\deploy.ps1`.
2. Verify pods are Ready and services are reachable.
3. Confirm `/health`, `/metrics`, and `/api/realtime/stats` endpoints respond.

## Production Readiness
1. Confirm environment variables are set for secrets and WebAuthn RP settings.
2. Validate Grafana dashboards load with Prometheus datasource.
3. Run smoke tests against API and WebSocket endpoints.

## Rollback
1. Redeploy the previous manifest version from `Control_net/kubernetes`.
2. Verify service health and traffic recovery.
