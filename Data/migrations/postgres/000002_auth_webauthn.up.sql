-- ASGARD Auth Hardening and WebAuthn Support

CREATE TABLE auth_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    user_agent TEXT,
    ip_address INET
);

CREATE UNIQUE INDEX idx_auth_refresh_hash ON auth_refresh_tokens(token_hash);
CREATE INDEX idx_auth_refresh_user ON auth_refresh_tokens(user_id);
CREATE INDEX idx_auth_refresh_status ON auth_refresh_tokens(revoked_at);

CREATE TABLE auth_token_revocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token_id UUID NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_auth_revocations_token ON auth_token_revocations(token_id);
CREATE INDEX idx_auth_revocations_user ON auth_token_revocations(user_id);

CREATE TABLE webauthn_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA UNIQUE NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type TEXT,
    transport TEXT[],
    sign_count BIGINT DEFAULT 0,
    aaguid UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_webauthn_credentials_user ON webauthn_credentials(user_id);

CREATE TABLE webauthn_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_type VARCHAR(20) NOT NULL CHECK (session_type IN ('registration', 'authentication')),
    challenge BYTEA NOT NULL,
    session_data JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_webauthn_sessions_user ON webauthn_sessions(user_id);
CREATE INDEX idx_webauthn_sessions_type ON webauthn_sessions(session_type);
