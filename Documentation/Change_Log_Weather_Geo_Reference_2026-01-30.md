# Valkyrie Weather Geo Reference Activation (2026-01-30)

## Summary
- Activated geo-referenced weather lookups for Valkyrie using N2YO real-time satellite positions when configured.
- Preserved fusion-state fallback with local meters -> lat/lon conversion for offline use.
- Added configuration defaults, Nysus API wiring, and documentation updates.

## Configuration
Update these values to match your operational settings:

```yaml
ai:
  geo_reference_enabled: true
  geo_reference_latitude: 37.7749
  geo_reference_longitude: -122.4194
  geo_reference_source: n2yo
  geo_reference_norad_id: 25544
```

N2YO access is provided by Nysus using `N2YO_API_KEY` from `.env`.

## Files Updated
- `Valkyrie/internal/ai/decision_engine.go` (N2YO-first coordinate sourcing)
- `Valkyrie/internal/integration/asgard.go` (Nysus satellite lookup client)
- `Valkyrie/configs/config.yaml` (default geo reference values)
- `Valkyrie/setup-valkyrie.ps1` (config template)
- `Valkyrie/deployment/k8s/valkyrie-deployment.yaml` (ConfigMap values)
- `Valkyrie/readME.md` (configuration snippet)
- `internal/nysus/api/server.go` (registered satellite tracking routes)
- `Documentation/Build_Log.md` (build entry)
- `Documentation/README.md` (index entry)
