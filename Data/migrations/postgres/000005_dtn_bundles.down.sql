-- Rollback DTN Bundle storage
DROP INDEX IF EXISTS idx_dtn_bundles_expires;
DROP INDEX IF EXISTS idx_dtn_bundles_status;
DROP INDEX IF EXISTS idx_dtn_bundles_destination;
DROP TABLE IF EXISTS dtn_bundles;
