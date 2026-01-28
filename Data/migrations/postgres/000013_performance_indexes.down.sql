-- Drop performance indexes for ASGARD database
-- Created: 2026-01-28

-- Drop Alerts table indexes
DROP INDEX IF EXISTS idx_alerts_satellite_created;
DROP INDEX IF EXISTS idx_alerts_status_created;
DROP INDEX IF EXISTS idx_alerts_unresolved;

-- Drop Streams table indexes
DROP INDEX IF EXISTS idx_streams_status_started;
DROP INDEX IF EXISTS idx_streams_type_status;
DROP INDEX IF EXISTS idx_streams_active;
DROP INDEX IF EXISTS idx_streams_user_id;

-- Drop Missions table indexes
DROP INDEX IF EXISTS idx_missions_status_priority;
DROP INDEX IF EXISTS idx_missions_hunoid;

-- Drop Threats table indexes
DROP INDEX IF EXISTS idx_threats_target_detected;
DROP INDEX IF EXISTS idx_threats_severity;

-- Drop WebAuthn credentials indexes
DROP INDEX IF EXISTS idx_webauthn_user_created;

-- Drop Users table indexes
DROP INDEX IF EXISTS idx_users_email_verified;
DROP INDEX IF EXISTS idx_users_subscription;

-- Drop Audit logs index
DROP INDEX IF EXISTS idx_audit_logs_user_time;

-- Drop Subscriptions index
DROP INDEX IF EXISTS idx_subscriptions_user_status;
