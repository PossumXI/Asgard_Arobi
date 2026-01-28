-- Performance indexes for ASGARD database
-- Created: 2026-01-28

-- Alerts table indexes
CREATE INDEX IF NOT EXISTS idx_alerts_satellite_created 
ON alerts(satellite_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_status_created 
ON alerts(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_unresolved 
ON alerts(created_at DESC) 
WHERE status IN ('new', 'acknowledged');

-- Streams table indexes
CREATE INDEX IF NOT EXISTS idx_streams_status_started 
ON streams(status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_streams_type_status 
ON streams(type, status);

CREATE INDEX IF NOT EXISTS idx_streams_active 
ON streams(started_at DESC) 
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_streams_user_id 
ON streams(user_id);

-- Missions table indexes
CREATE INDEX IF NOT EXISTS idx_missions_status_priority 
ON missions(status, priority DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_missions_hunoid 
ON missions USING GIN(assigned_hunoid_ids);

-- Threats table indexes
CREATE INDEX IF NOT EXISTS idx_threats_target_detected 
ON threats(target_component, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_threats_severity 
ON threats(severity, detected_at DESC);

-- WebAuthn credentials indexes
CREATE INDEX IF NOT EXISTS idx_webauthn_user_created 
ON webauthn_credentials(user_id, created_at DESC);

-- Users table indexes
CREATE INDEX IF NOT EXISTS idx_users_email_verified 
ON users(email) WHERE email_verified = true;

CREATE INDEX IF NOT EXISTS idx_users_subscription 
ON users(subscription_tier);

-- Audit logs index
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_time 
ON audit_logs(user_id, created_at DESC);

-- Subscriptions index
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_status 
ON subscriptions(user_id, status);
