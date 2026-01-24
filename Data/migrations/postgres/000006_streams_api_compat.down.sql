-- Revert stream API compatibility columns
ALTER TABLE streams DROP COLUMN IF EXISTS longitude;
ALTER TABLE streams DROP COLUMN IF EXISTS latitude;
ALTER TABLE streams DROP COLUMN IF EXISTS featured;
ALTER TABLE streams DROP COLUMN IF EXISTS description;
ALTER TABLE streams DROP COLUMN IF EXISTS latency_ms;
ALTER TABLE streams DROP COLUMN IF EXISTS stream_type;
