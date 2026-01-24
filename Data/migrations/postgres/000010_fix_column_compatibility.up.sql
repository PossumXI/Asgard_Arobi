-- ASGARD Column Compatibility Fix
-- Migration 000010: Add last_telemetry_at column aliases for backward compatibility
-- PostgreSQL 15+

-- ============================================================================
-- SECTION 1: ADD last_telemetry_at COLUMNS FOR COMPATIBILITY
-- ============================================================================

-- Add last_telemetry_at column to satellites table (synced with last_telemetry)
ALTER TABLE satellites ADD COLUMN IF NOT EXISTS last_telemetry_at TIMESTAMPTZ;

-- Add last_telemetry_at column to hunoids table (synced with last_telemetry)
ALTER TABLE hunoids ADD COLUMN IF NOT EXISTS last_telemetry_at TIMESTAMPTZ;

-- ============================================================================
-- SECTION 2: CREATE TRIGGERS TO SYNC last_telemetry_at WITH last_telemetry
-- ============================================================================

-- Function to sync last_telemetry_at with last_telemetry for satellites
CREATE OR REPLACE FUNCTION sync_satellite_telemetry_at()
RETURNS TRIGGER AS $$
BEGIN
    -- Sync last_telemetry_at with last_telemetry
    IF NEW.last_telemetry IS NOT NULL THEN
        NEW.last_telemetry_at := NEW.last_telemetry;
    ELSIF NEW.last_telemetry_at IS NOT NULL AND NEW.last_telemetry IS NULL THEN
        NEW.last_telemetry := NEW.last_telemetry_at;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to sync last_telemetry_at with last_telemetry for hunoids
CREATE OR REPLACE FUNCTION sync_hunoid_telemetry_at()
RETURNS TRIGGER AS $$
BEGIN
    -- Sync last_telemetry_at with last_telemetry
    IF NEW.last_telemetry IS NOT NULL THEN
        NEW.last_telemetry_at := NEW.last_telemetry;
    ELSIF NEW.last_telemetry_at IS NOT NULL AND NEW.last_telemetry IS NULL THEN
        NEW.last_telemetry := NEW.last_telemetry_at;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
DROP TRIGGER IF EXISTS trigger_sync_satellite_telemetry_at ON satellites;
CREATE TRIGGER trigger_sync_satellite_telemetry_at
    BEFORE INSERT OR UPDATE ON satellites
    FOR EACH ROW EXECUTE FUNCTION sync_satellite_telemetry_at();

DROP TRIGGER IF EXISTS trigger_sync_hunoid_telemetry_at ON hunoids;
CREATE TRIGGER trigger_sync_hunoid_telemetry_at
    BEFORE INSERT OR UPDATE ON hunoids
    FOR EACH ROW EXECUTE FUNCTION sync_hunoid_telemetry_at();

-- ============================================================================
-- SECTION 3: BACKFILL EXISTING DATA
-- ============================================================================

-- Backfill last_telemetry_at for existing satellites
UPDATE satellites 
SET last_telemetry_at = last_telemetry
WHERE last_telemetry IS NOT NULL AND last_telemetry_at IS NULL;

-- Backfill last_telemetry_at for existing hunoids
UPDATE hunoids 
SET last_telemetry_at = last_telemetry
WHERE last_telemetry IS NOT NULL AND last_telemetry_at IS NULL;

-- ============================================================================
-- SECTION 4: CREATE INDEXES FOR PERFORMANCE
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_satellites_last_telemetry_at ON satellites(last_telemetry_at);
CREATE INDEX IF NOT EXISTS idx_hunoids_last_telemetry_at ON hunoids(last_telemetry_at);

-- ============================================================================
-- SECTION 5: COMMENTS
-- ============================================================================

COMMENT ON COLUMN satellites.last_telemetry_at IS 'Alias for last_telemetry column - maintained for backward compatibility';
COMMENT ON COLUMN hunoids.last_telemetry_at IS 'Alias for last_telemetry column - maintained for backward compatibility';
