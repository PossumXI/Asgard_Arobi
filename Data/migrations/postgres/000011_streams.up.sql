-- Streams and stream sessions
CREATE TABLE streams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    source TEXT,
    source_type TEXT,
    source_id TEXT,
    location TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    viewers INTEGER DEFAULT 0,
    latency INTEGER,
    resolution TEXT,
    bitrate INTEGER,
    started_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    geo_lat DOUBLE PRECISION,
    geo_lon DOUBLE PRECISION,
    geo_alt DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_streams_type ON streams(type);
CREATE INDEX idx_streams_status ON streams(status);
CREATE INDEX idx_streams_started_at ON streams(started_at);
CREATE INDEX idx_streams_viewers ON streams(viewers);

CREATE TABLE stream_sessions (
    id UUID PRIMARY KEY,
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ice_servers JSONB NOT NULL,
    signaling_url TEXT NOT NULL,
    auth_token TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_stream_sessions_stream ON stream_sessions(stream_id);
CREATE INDEX idx_stream_sessions_user ON stream_sessions(user_id);
CREATE INDEX idx_stream_sessions_expires ON stream_sessions(expires_at);
