-- Execution evidence is retained until the worker acknowledges durable logs
-- and results. Additional post-ack retention is an explicit debugging option.
ALTER TABLE build_platform_settings ALTER COLUMN job_retention_seconds SET DEFAULT 0;

-- Upgrade only the untouched bootstrap row created by migration 033.
-- Existing administrator choices remain unchanged by the schema migration.
UPDATE build_platform_settings SET job_retention_seconds=0
WHERE id=1 AND updated_by=0 AND enabled=0 AND cluster_id=0
  AND build_command='' AND job_retention_seconds=86400;
