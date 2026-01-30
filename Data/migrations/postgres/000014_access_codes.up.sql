-- Access codes for clearance-controlled portals and APIs
CREATE TABLE access_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code_hash TEXT UNIQUE NOT NULL,
    code_last4 VARCHAR(4) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    clearance_level VARCHAR(50) NOT NULL DEFAULT 'civilian'
        CHECK (clearance_level IN ('public', 'civilian', 'military', 'interstellar', 'government', 'admin')),
    scope VARCHAR(50) NOT NULL DEFAULT 'portal'
        CHECK (scope IN ('portal', 'api', 'hubs', 'electron', 'all')),
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count INTEGER NOT NULL DEFAULT 0,
    max_uses INTEGER,
    rotation_interval_hours INTEGER NOT NULL DEFAULT 24,
    next_rotation_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    note TEXT
);

CREATE INDEX idx_access_codes_user ON access_codes(user_id);
CREATE INDEX idx_access_codes_active ON access_codes(expires_at, revoked_at);
CREATE INDEX idx_access_codes_rotation ON access_codes(next_rotation_at) WHERE revoked_at IS NULL;
