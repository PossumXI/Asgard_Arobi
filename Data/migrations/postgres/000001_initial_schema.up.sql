-- ASGARD Core Metadata Schema
-- PostgreSQL 15+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- Users table (for Websites authentication)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(255),
    subscription_tier VARCHAR(50) DEFAULT 'observer' CHECK (subscription_tier IN ('observer', 'supporter', 'commander')),
    is_government BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription ON users(subscription_tier);

-- Satellites table (Silenus fleet)
CREATE TABLE satellites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    norad_id INTEGER UNIQUE,
    name VARCHAR(255) NOT NULL,
    orbital_elements JSONB NOT NULL, -- TLE data
    hardware_config JSONB, -- Camera specs, sensors
    current_battery_percent FLOAT CHECK (current_battery_percent >= 0 AND current_battery_percent <= 100),
    status VARCHAR(50) DEFAULT 'operational' CHECK (status IN ('operational', 'eclipse', 'maintenance', 'decommissioned')),
    last_telemetry TIMESTAMP WITH TIME ZONE,
    firmware_version VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_satellites_status ON satellites(status);
CREATE INDEX idx_satellites_battery ON satellites(current_battery_percent);

-- Hunoid robots table
CREATE TABLE hunoids (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    serial_number VARCHAR(100) UNIQUE NOT NULL,
    current_location GEOGRAPHY(POINT), -- PostGIS for lat/lon
    current_mission_id UUID, -- FK to missions table
    hardware_config JSONB, -- Actuators, sensors, compute
    battery_percent FLOAT CHECK (battery_percent >= 0 AND battery_percent <= 100),
    status VARCHAR(50) DEFAULT 'idle' CHECK (status IN ('idle', 'active', 'charging', 'maintenance', 'emergency')),
    vla_model_version VARCHAR(50),
    ethical_score FLOAT DEFAULT 1.0 CHECK (ethical_score >= 0 AND ethical_score <= 1),
    last_telemetry TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_hunoids_status ON hunoids(status);
CREATE INDEX idx_hunoids_location ON hunoids USING GIST(current_location);

-- Missions table
CREATE TABLE missions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    mission_type VARCHAR(100) NOT NULL, -- 'search_rescue', 'aid_delivery', 'reconnaissance'
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed', 'aborted')),
    assigned_hunoid_ids UUID[], -- Array of hunoid IDs
    target_location GEOGRAPHY(POINT),
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_missions_status ON missions(status);
CREATE INDEX idx_missions_priority ON missions(priority);

-- Alerts table (from Silenus detections)
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    satellite_id UUID REFERENCES satellites(id),
    alert_type VARCHAR(100) NOT NULL, -- 'tsunami', 'fire', 'troop_movement', 'missile_launch'
    confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
    detection_location GEOGRAPHY(POINT),
    video_segment_url TEXT, -- S3 URL or local path
    metadata JSONB, -- Detection bounding boxes, etc.
    status VARCHAR(50) DEFAULT 'new' CHECK (status IN ('new', 'acknowledged', 'dispatched', 'resolved')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alerts_type ON alerts(alert_type);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created ON alerts(created_at);

-- Threat incidents (Giru detections)
CREATE TABLE threats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    threat_type VARCHAR(100) NOT NULL, -- 'ddos', 'intrusion', 'malware'
    severity VARCHAR(50) DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    source_ip INET,
    target_component VARCHAR(100), -- 'nysus', 'sat_net', 'websites'
    attack_vector TEXT,
    mitigation_action TEXT,
    status VARCHAR(50) DEFAULT 'detected' CHECK (status IN ('detected', 'mitigated', 'resolved')),
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_threats_severity ON threats(severity);
CREATE INDEX idx_threats_status ON threats(status);

-- Subscriptions table (for Stripe integration)
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id VARCHAR(255),
    tier VARCHAR(50) CHECK (tier IN ('observer', 'supporter', 'commander')),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired')),
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);

-- System audit log
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    component VARCHAR(100) NOT NULL, -- 'nysus', 'giru', 'hunoid', etc.
    action VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_component ON audit_logs(component);
CREATE INDEX idx_audit_created ON audit_logs(created_at);

-- Ethical decision log (for Hunoid actions)
CREATE TABLE ethical_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hunoid_id UUID REFERENCES hunoids(id),
    proposed_action TEXT NOT NULL,
    ethical_assessment JSONB, -- Rules checked, scores
    decision VARCHAR(50) CHECK (decision IN ('approved', 'rejected', 'escalated')),
    reasoning TEXT,
    human_override BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_ethical_hunoid ON ethical_decisions(hunoid_id);
CREATE INDEX idx_ethical_decision ON ethical_decisions(decision);

-- Update triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_satellites_updated_at BEFORE UPDATE ON satellites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hunoids_updated_at BEFORE UPDATE ON hunoids
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
