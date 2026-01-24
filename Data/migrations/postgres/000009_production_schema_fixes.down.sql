-- Rollback for Migration 000009

-- Drop views
DROP VIEW IF EXISTS alerts_api;
DROP VIEW IF EXISTS hunoids_api;
DROP VIEW IF EXISTS satellites_api;

-- Drop PERCILA tables
DROP TABLE IF EXISTS percila_waypoints;
DROP TABLE IF EXISTS percila_payloads;
DROP TABLE IF EXISTS percila_missions;

-- Drop control plane tables
DROP TABLE IF EXISTS control_commands;
DROP TABLE IF EXISTS system_config;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_sync_alert_location ON alerts;
DROP TRIGGER IF EXISTS trigger_sync_hunoid_location ON hunoids;

-- Drop functions
DROP FUNCTION IF EXISTS sync_alert_location();
DROP FUNCTION IF EXISTS sync_hunoid_location();
DROP FUNCTION IF EXISTS calculate_distance_meters(DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION);
DROP FUNCTION IF EXISTS find_nearby_hunoids(DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION);
DROP FUNCTION IF EXISTS find_alerts_in_region(DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, INTEGER);
DROP FUNCTION IF EXISTS get_active_missions_with_hunoids();
DROP FUNCTION IF EXISTS update_satellite_telemetry(UUID, DOUBLE PRECISION, VARCHAR);
DROP FUNCTION IF EXISTS update_hunoid_telemetry(UUID, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, VARCHAR);
DROP FUNCTION IF EXISTS create_alert(UUID, VARCHAR, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, DOUBLE PRECISION, TEXT, JSONB);
DROP FUNCTION IF EXISTS get_system_health_stats();
DROP FUNCTION IF EXISTS get_dashboard_data(UUID);

-- Remove added columns (optional - may want to keep data)
-- ALTER TABLE alerts DROP COLUMN IF EXISTS latitude;
-- ALTER TABLE alerts DROP COLUMN IF EXISTS longitude;
-- ALTER TABLE alerts DROP COLUMN IF EXISTS altitude;
-- ALTER TABLE hunoids DROP COLUMN IF EXISTS latitude;
-- ALTER TABLE hunoids DROP COLUMN IF EXISTS longitude;
-- ALTER TABLE hunoids DROP COLUMN IF EXISTS altitude;

-- Drop roles (be careful in production)
-- DROP ROLE IF EXISTS asgard_readonly;
-- DROP ROLE IF EXISTS asgard_app;
