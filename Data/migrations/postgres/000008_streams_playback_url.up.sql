-- Playback URL for external streams (HLS/MP4)
ALTER TABLE streams ADD COLUMN IF NOT EXISTS playback_url TEXT;
