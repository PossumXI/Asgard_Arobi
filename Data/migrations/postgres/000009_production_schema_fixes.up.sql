-- ASGARD Production Schema Fixes & Enhancements
-- Migration 000009: Fix column mismatches, add geo columns, edge functions
-- PostgreSQL 15+

-- ============================================================================
-- SECTION 1: ADD MISSING COLUMNS FOR API COMPATIBILITY
-- ============================================================================

-- Add latitude/longitude columns to alerts table (derived from detection_location)
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION;
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS altitude DOUBLE PRECISION DEFAULT 0;

-- Add latitude/longitude columns to hunoids table (derived from current_location)
ALTER TABLE hunoids ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION;
ALTER TABLE hunoids ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;
ALTER TABLE hunoids ADD COLUMN IF NOT EXISTS altitude DOUBLE PRECISION DEFAULT 0;

-- Add view-friendly column aliases using generated columns (if supported)
-- For older PostgreSQL, we'll use triggers instead

-- Create a trigger to sync detection_location with lat/lon columns
CREATE OR REPLACE FUNCTION sync_alert_location()
RETURNS TRIGGER AS $$
BEGIN
    -- If detection_location is set, extract lat/lon
    IF NEW.detection_location IS NOT NULL THEN
        NEW.latitude := ST_Y(NEW.detection_location::geometry);
        NEW.longitude := ST_X(NEW.detection_location::geometry);
    -- If lat/lon are set but detection_location is not, create it
    ELSIF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.detection_location := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_sync_alert_location ON alerts;
CREATE TRIGGER trigger_sync_alert_location
    BEFORE INSERT OR UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION sync_alert_location();

-- Create a trigger to sync current_location with lat/lon columns for hunoids
CREATE OR REPLACE FUNCTION sync_hunoid_location()
RETURNS TRIGGER AS $$
BEGIN
    -- If current_location is set, extract lat/lon
    IF NEW.current_location IS NOT NULL THEN
        NEW.latitude := ST_Y(NEW.current_location::geometry);
        NEW.longitude := ST_X(NEW.current_location::geometry);
    -- If lat/lon are set but current_location is not, create it
    ELSIF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.current_location := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_sync_hunoid_location ON hunoids;
CREATE TRIGGER trigger_sync_hunoid_location
    BEFORE INSERT OR UPDATE ON hunoids
    FOR EACH ROW EXECUTE FUNCTION sync_hunoid_location();

-- ============================================================================
-- SECTION 2: CREATE COLUMN ALIASES VIEWS FOR API COMPATIBILITY
-- ============================================================================

-- Create a view for satellites with last_telemetry_at alias
CREATE OR REPLACE VIEW satellites_api AS
SELECT 
    id, norad_id, name, orbital_elements, hardware_config,
    current_battery_percent, status, 
    last_telemetry AS last_telemetry_at,  -- Alias for API compatibility
    last_telemetry,
    firmware_version, created_at, updated_at
FROM satellites;

-- Create a view for hunoids with last_telemetry_at alias and lat/lon
CREATE OR REPLACE VIEW hunoids_api AS
SELECT 
    id, serial_number, 
    latitude, longitude, altitude,
    current_location,
    current_mission_id, hardware_config,
    battery_percent, status, vla_model_version, ethical_score,
    last_telemetry AS last_telemetry_at,  -- Alias for API compatibility
    last_telemetry,
    created_at, updated_at
FROM hunoids;

-- Create a view for alerts with lat/lon
CREATE OR REPLACE VIEW alerts_api AS
SELECT 
    id, satellite_id, alert_type, confidence_score,
    latitude, longitude, altitude,
    detection_location,
    video_segment_url, metadata, status, created_at
FROM alerts;

-- ============================================================================
-- SECTION 3: PRODUCTION EDGE FUNCTIONS
-- ============================================================================

-- Function to calculate distance between two geo points (in meters)
CREATE OR REPLACE FUNCTION calculate_distance_meters(
    lat1 DOUBLE PRECISION, lon1 DOUBLE PRECISION,
    lat2 DOUBLE PRECISION, lon2 DOUBLE PRECISION
) RETURNS DOUBLE PRECISION AS $$
DECLARE
    point1 geography;
    point2 geography;
BEGIN
    point1 := ST_SetSRID(ST_MakePoint(lon1, lat1), 4326)::geography;
    point2 := ST_SetSRID(ST_MakePoint(lon2, lat2), 4326)::geography;
    RETURN ST_Distance(point1, point2);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to find nearby hunoids within a radius (meters)
CREATE OR REPLACE FUNCTION find_nearby_hunoids(
    center_lat DOUBLE PRECISION,
    center_lon DOUBLE PRECISION,
    radius_meters DOUBLE PRECISION
) RETURNS TABLE (
    hunoid_id UUID,
    serial_number VARCHAR,
    distance_meters DOUBLE PRECISION,
    battery_percent DOUBLE PRECISION,
    status VARCHAR
) AS $$
DECLARE
    center_point geography;
BEGIN
    center_point := ST_SetSRID(ST_MakePoint(center_lon, center_lat), 4326)::geography;
    
    RETURN QUERY
    SELECT 
        h.id,
        h.serial_number,
        ST_Distance(h.current_location, center_point) AS distance_meters,
        h.battery_percent,
        h.status
    FROM hunoids h
    WHERE h.current_location IS NOT NULL
      AND ST_DWithin(h.current_location, center_point, radius_meters)
    ORDER BY ST_Distance(h.current_location, center_point);
END;
$$ LANGUAGE plpgsql;

-- Function to find alerts within a geographic region
CREATE OR REPLACE FUNCTION find_alerts_in_region(
    min_lat DOUBLE PRECISION, min_lon DOUBLE PRECISION,
    max_lat DOUBLE PRECISION, max_lon DOUBLE PRECISION,
    since_hours INTEGER DEFAULT 24
) RETURNS TABLE (
    alert_id UUID,
    alert_type VARCHAR,
    confidence_score DOUBLE PRECISION,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    status VARCHAR,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.id,
        a.alert_type,
        a.confidence_score,
        a.latitude,
        a.longitude,
        a.status,
        a.created_at
    FROM alerts a
    WHERE a.latitude BETWEEN min_lat AND max_lat
      AND a.longitude BETWEEN min_lon AND max_lon
      AND a.created_at > NOW() - (since_hours || ' hours')::INTERVAL
    ORDER BY a.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to get active missions with assigned hunoids
CREATE OR REPLACE FUNCTION get_active_missions_with_hunoids()
RETURNS TABLE (
    mission_id UUID,
    mission_type VARCHAR,
    priority INTEGER,
    status VARCHAR,
    hunoid_count INTEGER,
    started_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.mission_type,
        m.priority,
        m.status,
        COALESCE(array_length(m.assigned_hunoid_ids, 1), 0) AS hunoid_count,
        m.started_at
    FROM missions m
    WHERE m.status IN ('pending', 'active')
    ORDER BY m.priority DESC, m.created_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to update satellite telemetry and return battery status
CREATE OR REPLACE FUNCTION update_satellite_telemetry(
    p_satellite_id UUID,
    p_battery_percent DOUBLE PRECISION,
    p_status VARCHAR DEFAULT NULL
) RETURNS TABLE (
    satellite_id UUID,
    name VARCHAR,
    old_battery DOUBLE PRECISION,
    new_battery DOUBLE PRECISION,
    battery_change DOUBLE PRECISION,
    low_battery_warning BOOLEAN
) AS $$
DECLARE
    v_old_battery DOUBLE PRECISION;
    v_name VARCHAR;
BEGIN
    -- Get current battery level
    SELECT s.current_battery_percent, s.name INTO v_old_battery, v_name
    FROM satellites s WHERE s.id = p_satellite_id;
    
    -- Update telemetry
    UPDATE satellites
    SET 
        current_battery_percent = p_battery_percent,
        last_telemetry = NOW(),
        status = COALESCE(p_status, status)
    WHERE id = p_satellite_id;
    
    -- Return results
    RETURN QUERY SELECT 
        p_satellite_id,
        v_name,
        v_old_battery,
        p_battery_percent,
        p_battery_percent - COALESCE(v_old_battery, p_battery_percent),
        p_battery_percent < 20;
END;
$$ LANGUAGE plpgsql;

-- Function to update hunoid telemetry with location
CREATE OR REPLACE FUNCTION update_hunoid_telemetry(
    p_hunoid_id UUID,
    p_latitude DOUBLE PRECISION,
    p_longitude DOUBLE PRECISION,
    p_altitude DOUBLE PRECISION DEFAULT 0,
    p_battery_percent DOUBLE PRECISION DEFAULT NULL,
    p_status VARCHAR DEFAULT NULL
) RETURNS TABLE (
    hunoid_id UUID,
    serial_number VARCHAR,
    distance_moved_meters DOUBLE PRECISION,
    battery_percent DOUBLE PRECISION,
    status VARCHAR
) AS $$
DECLARE
    v_old_location geography;
    v_serial VARCHAR;
    v_distance DOUBLE PRECISION;
    v_new_location geography;
BEGIN
    -- Get current location
    SELECT h.current_location, h.serial_number INTO v_old_location, v_serial
    FROM hunoids h WHERE h.id = p_hunoid_id;
    
    -- Create new location
    v_new_location := ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326)::geography;
    
    -- Calculate distance moved
    IF v_old_location IS NOT NULL THEN
        v_distance := ST_Distance(v_old_location, v_new_location);
    ELSE
        v_distance := 0;
    END IF;
    
    -- Update telemetry
    UPDATE hunoids
    SET 
        current_location = v_new_location,
        latitude = p_latitude,
        longitude = p_longitude,
        altitude = p_altitude,
        battery_percent = COALESCE(p_battery_percent, battery_percent),
        last_telemetry = NOW(),
        status = COALESCE(p_status, status)
    WHERE id = p_hunoid_id;
    
    -- Return results
    RETURN QUERY SELECT 
        p_hunoid_id,
        v_serial,
        v_distance,
        (SELECT h.battery_percent FROM hunoids h WHERE h.id = p_hunoid_id),
        (SELECT h.status FROM hunoids h WHERE h.id = p_hunoid_id);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 4: AUTOMATED ALERT GENERATION FUNCTION
-- ============================================================================

-- Function to create an alert with automatic location handling
CREATE OR REPLACE FUNCTION create_alert(
    p_satellite_id UUID,
    p_alert_type VARCHAR,
    p_confidence_score DOUBLE PRECISION,
    p_latitude DOUBLE PRECISION,
    p_longitude DOUBLE PRECISION,
    p_altitude DOUBLE PRECISION DEFAULT 0,
    p_video_url TEXT DEFAULT NULL,
    p_metadata JSONB DEFAULT '{}'::JSONB
) RETURNS UUID AS $$
DECLARE
    v_alert_id UUID;
    v_location geography;
BEGIN
    -- Create geography point
    v_location := ST_SetSRID(ST_MakePoint(p_longitude, p_latitude), 4326)::geography;
    
    -- Insert alert
    INSERT INTO alerts (
        satellite_id, alert_type, confidence_score,
        detection_location, latitude, longitude, altitude,
        video_segment_url, metadata, status
    ) VALUES (
        p_satellite_id, p_alert_type, p_confidence_score,
        v_location, p_latitude, p_longitude, p_altitude,
        p_video_url, p_metadata, 'new'
    )
    RETURNING id INTO v_alert_id;
    
    -- Log to audit
    INSERT INTO audit_logs (component, action, metadata)
    VALUES (
        'silenus',
        'alert_created',
        jsonb_build_object(
            'alert_id', v_alert_id,
            'alert_type', p_alert_type,
            'confidence', p_confidence_score,
            'location', jsonb_build_object('lat', p_latitude, 'lon', p_longitude)
        )
    );
    
    RETURN v_alert_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 5: STATISTICS AND MONITORING FUNCTIONS
-- ============================================================================

-- Function to get system health statistics
CREATE OR REPLACE FUNCTION get_system_health_stats()
RETURNS TABLE (
    total_satellites INTEGER,
    operational_satellites INTEGER,
    low_battery_satellites INTEGER,
    total_hunoids INTEGER,
    active_hunoids INTEGER,
    charging_hunoids INTEGER,
    pending_missions INTEGER,
    active_missions INTEGER,
    new_alerts_24h INTEGER,
    high_confidence_alerts_24h INTEGER,
    threats_detected_24h INTEGER,
    critical_threats_24h INTEGER
) AS $$
BEGIN
    RETURN QUERY SELECT
        (SELECT COUNT(*)::INTEGER FROM satellites),
        (SELECT COUNT(*)::INTEGER FROM satellites WHERE status = 'operational'),
        (SELECT COUNT(*)::INTEGER FROM satellites WHERE current_battery_percent < 20),
        (SELECT COUNT(*)::INTEGER FROM hunoids),
        (SELECT COUNT(*)::INTEGER FROM hunoids WHERE status = 'active'),
        (SELECT COUNT(*)::INTEGER FROM hunoids WHERE status = 'charging'),
        (SELECT COUNT(*)::INTEGER FROM missions WHERE status = 'pending'),
        (SELECT COUNT(*)::INTEGER FROM missions WHERE status = 'active'),
        (SELECT COUNT(*)::INTEGER FROM alerts WHERE created_at > NOW() - INTERVAL '24 hours'),
        (SELECT COUNT(*)::INTEGER FROM alerts WHERE created_at > NOW() - INTERVAL '24 hours' AND confidence_score > 0.8),
        (SELECT COUNT(*)::INTEGER FROM threats WHERE detected_at > NOW() - INTERVAL '24 hours'),
        (SELECT COUNT(*)::INTEGER FROM threats WHERE detected_at > NOW() - INTERVAL '24 hours' AND severity = 'critical');
END;
$$ LANGUAGE plpgsql;

-- Function to get real-time dashboard data
CREATE OR REPLACE FUNCTION get_dashboard_data(p_user_id UUID DEFAULT NULL)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB;
BEGIN
    SELECT jsonb_build_object(
        'satellites', (
            SELECT COALESCE(jsonb_agg(sat_row), '[]'::jsonb) FROM (
                SELECT jsonb_build_object(
                    'id', s.id,
                    'name', s.name,
                    'status', s.status,
                    'battery', s.current_battery_percent,
                    'last_telemetry', s.last_telemetry
                ) AS sat_row
                FROM satellites s
                WHERE s.status = 'operational'
                ORDER BY s.name
                LIMIT 50
            ) sub
        ),
        'hunoids', (
            SELECT COALESCE(jsonb_agg(hun_row), '[]'::jsonb) FROM (
                SELECT jsonb_build_object(
                    'id', h.id,
                    'serial_number', h.serial_number,
                    'status', h.status,
                    'battery', h.battery_percent,
                    'location', CASE
                        WHEN h.latitude IS NOT NULL THEN
                            jsonb_build_object('lat', h.latitude, 'lon', h.longitude)
                        ELSE NULL
                    END,
                    'last_telemetry', h.last_telemetry
                ) AS hun_row
                FROM hunoids h
                WHERE h.status IN ('active', 'idle')
                ORDER BY h.serial_number
                LIMIT 50
            ) sub
        ),
        'recent_alerts', (
            SELECT COALESCE(jsonb_agg(alert_row), '[]'::jsonb) FROM (
                SELECT jsonb_build_object(
                    'id', a.id,
                    'type', a.alert_type,
                    'confidence', a.confidence_score,
                    'location', jsonb_build_object('lat', a.latitude, 'lon', a.longitude),
                    'status', a.status,
                    'created_at', a.created_at
                ) AS alert_row
                FROM alerts a
                WHERE a.created_at > NOW() - INTERVAL '24 hours'
                ORDER BY a.created_at DESC
                LIMIT 20
            ) sub
        ),
        'active_missions', (
            SELECT COALESCE(jsonb_agg(mission_row), '[]'::jsonb) FROM (
                SELECT jsonb_build_object(
                    'id', m.id,
                    'type', m.mission_type,
                    'priority', m.priority,
                    'status', m.status,
                    'started_at', m.started_at
                ) AS mission_row
                FROM missions m
                WHERE m.status IN ('pending', 'active')
                ORDER BY m.priority DESC
                LIMIT 10
            ) sub
        ),
        'stats', (SELECT row_to_json(s.*) FROM get_system_health_stats() s)
    ) INTO v_result;

    RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SECTION 6: PERCILA INTEGRATION TABLES
-- ============================================================================

-- PERCILA missions table
CREATE TABLE IF NOT EXISTS percila_missions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mission_type VARCHAR(50) NOT NULL,
    payload_type VARCHAR(50) NOT NULL,
    target_id VARCHAR(255),
    target_type VARCHAR(50),
    start_position JSONB NOT NULL,
    target_position JSONB NOT NULL,
    current_position JSONB,
    status VARCHAR(50) DEFAULT 'planning' CHECK (status IN ('planning', 'active', 'completed', 'aborted', 'failed')),
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    guidance_law VARCHAR(50) DEFAULT 'aug_pronav',
    constraints JSONB DEFAULT '{}',
    telemetry JSONB DEFAULT '[]',
    metrics JSONB DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_percila_missions_status ON percila_missions(status);
CREATE INDEX IF NOT EXISTS idx_percila_missions_type ON percila_missions(mission_type);

-- PERCILA payloads table
CREATE TABLE IF NOT EXISTS percila_payloads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payload_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    serial_number VARCHAR(100),
    current_position JSONB,
    current_velocity JSONB,
    status VARCHAR(50) DEFAULT 'ready' CHECK (status IN ('ready', 'deployed', 'in_flight', 'terminal', 'impact', 'lost')),
    specs JSONB DEFAULT '{}',
    mission_id UUID REFERENCES percila_missions(id),
    last_telemetry TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_percila_payloads_status ON percila_payloads(status);
CREATE INDEX IF NOT EXISTS idx_percila_payloads_type ON percila_payloads(payload_type);

-- PERCILA trajectory waypoints
CREATE TABLE IF NOT EXISTS percila_waypoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mission_id UUID REFERENCES percila_missions(id) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    position JSONB NOT NULL,
    velocity JSONB,
    expected_time TIMESTAMPTZ,
    actual_time TIMESTAMPTZ,
    delta_v DOUBLE PRECISION DEFAULT 0,
    purpose VARCHAR(50) DEFAULT 'waypoint',
    reached BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_percila_waypoints_mission ON percila_waypoints(mission_id);

-- Trigger for PERCILA missions updated_at
CREATE TRIGGER update_percila_missions_updated_at 
    BEFORE UPDATE ON percila_missions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_percila_payloads_updated_at 
    BEFORE UPDATE ON percila_payloads
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SECTION 7: CONTROL PLANE TABLES
-- ============================================================================

-- Control commands table (for government/admin)
CREATE TABLE IF NOT EXISTS control_commands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    command_type VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL, -- 'hunoid', 'satellite', 'mission', 'system'
    target_id UUID,
    payload JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'acknowledged', 'executed', 'failed', 'cancelled')),
    priority INTEGER DEFAULT 5,
    issued_by UUID REFERENCES users(id),
    executed_at TIMESTAMPTZ,
    result JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_control_commands_status ON control_commands(status);
CREATE INDEX IF NOT EXISTS idx_control_commands_target ON control_commands(target_type, target_id);

-- System configuration table
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- SECTION 8: BACKFILL EXISTING DATA
-- ============================================================================

-- Backfill lat/lon for existing alerts with detection_location
UPDATE alerts 
SET 
    latitude = ST_Y(detection_location::geometry),
    longitude = ST_X(detection_location::geometry)
WHERE detection_location IS NOT NULL AND latitude IS NULL;

-- Backfill lat/lon for existing hunoids with current_location
UPDATE hunoids 
SET 
    latitude = ST_Y(current_location::geometry),
    longitude = ST_X(current_location::geometry)
WHERE current_location IS NOT NULL AND latitude IS NULL;

-- ============================================================================
-- SECTION 9: GRANTS AND PERMISSIONS (for production)
-- ============================================================================

-- Create read-only role for monitoring
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'asgard_readonly') THEN
        CREATE ROLE asgard_readonly;
    END IF;
END
$$;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO asgard_readonly;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO asgard_readonly;

-- Create application role
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'asgard_app') THEN
        CREATE ROLE asgard_app;
    END IF;
END
$$;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO asgard_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO asgard_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO asgard_app;

-- ============================================================================
-- SECTION 10: COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON FUNCTION calculate_distance_meters IS 'Calculate distance between two lat/lon points in meters using PostGIS';
COMMENT ON FUNCTION find_nearby_hunoids IS 'Find hunoid robots within a specified radius of a point';
COMMENT ON FUNCTION find_alerts_in_region IS 'Find alerts within a geographic bounding box';
COMMENT ON FUNCTION create_alert IS 'Create a new alert with automatic location handling and audit logging';
COMMENT ON FUNCTION get_system_health_stats IS 'Get aggregated system health statistics';
COMMENT ON FUNCTION get_dashboard_data IS 'Get comprehensive dashboard data as JSON';
COMMENT ON FUNCTION update_satellite_telemetry IS 'Update satellite telemetry and return battery status';
COMMENT ON FUNCTION update_hunoid_telemetry IS 'Update hunoid telemetry with location tracking';

COMMENT ON TABLE percila_missions IS 'PERCILA guidance system missions';
COMMENT ON TABLE percila_payloads IS 'PERCILA tracked payloads';
COMMENT ON TABLE percila_waypoints IS 'PERCILA mission trajectory waypoints';
COMMENT ON TABLE control_commands IS 'Government/admin control commands';
COMMENT ON TABLE system_config IS 'System-wide configuration key-value store';
