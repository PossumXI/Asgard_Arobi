-- Stream API compatibility columns for Nysus
ALTER TABLE streams ADD COLUMN IF NOT EXISTS stream_type TEXT;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS latency_ms INTEGER;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS featured BOOLEAN DEFAULT false;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION;
ALTER TABLE streams ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION;

-- Backfill compatibility columns from existing schema
UPDATE streams
SET stream_type = COALESCE(stream_type, type),
    latency_ms = COALESCE(latency_ms, latency),
    description = COALESCE(description, title),
    latitude = COALESCE(latitude, geo_lat),
    longitude = COALESCE(longitude, geo_lon)
WHERE stream_type IS NULL
   OR latency_ms IS NULL
   OR description IS NULL
   OR latitude IS NULL
   OR longitude IS NULL;
