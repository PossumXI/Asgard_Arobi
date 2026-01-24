-- DTN Bundle storage for delay-tolerant networking transport layer
CREATE TABLE IF NOT EXISTS dtn_bundles (
    id TEXT PRIMARY KEY,
    source_eid TEXT NOT NULL,
    destination_eid TEXT NOT NULL,
    payload BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    priority INTEGER NOT NULL DEFAULT 1,
    hop_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    CONSTRAINT valid_status CHECK (status IN ('pending', 'in_transit', 'delivered', 'failed', 'expired'))
);

CREATE INDEX idx_dtn_bundles_destination ON dtn_bundles(destination_eid);
CREATE INDEX idx_dtn_bundles_status ON dtn_bundles(status);
CREATE INDEX idx_dtn_bundles_expires ON dtn_bundles(expires_at);
