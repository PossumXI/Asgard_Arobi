-- Rollback migration 000010

-- Drop indexes
DROP INDEX IF EXISTS idx_hunoids_last_telemetry_at;
DROP INDEX IF EXISTS idx_satellites_last_telemetry_at;

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_sync_hunoid_telemetry_at ON hunoids;
DROP TRIGGER IF EXISTS trigger_sync_satellite_telemetry_at ON satellites;

-- Drop functions
DROP FUNCTION IF EXISTS sync_hunoid_telemetry_at();
DROP FUNCTION IF EXISTS sync_satellite_telemetry_at();

-- Drop columns
ALTER TABLE hunoids DROP COLUMN IF EXISTS last_telemetry_at;
ALTER TABLE satellites DROP COLUMN IF EXISTS last_telemetry_at;
