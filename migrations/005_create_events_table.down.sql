DROP INDEX IF EXISTS idx_snapshots_version;
DROP INDEX IF EXISTS idx_snapshots_aggregate_type;
DROP TABLE IF EXISTS snapshots;

DROP INDEX IF EXISTS idx_events_aggregate_version;
DROP INDEX IF EXISTS idx_events_version;
DROP INDEX IF EXISTS idx_events_created_at;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_aggregate_id;
DROP TABLE IF EXISTS events;
