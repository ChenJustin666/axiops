-- Idempotent build log snapshots and provider result metadata.
ALTER TABLE `build_log`
  ADD COLUMN `truncated` TINYINT(1) NOT NULL DEFAULT 0 AFTER `line_count`,
  ADD COLUMN `log_key` VARCHAR(128) NULL AFTER `truncated`;

-- Existing rows remain NULL and need no destructive deduplication. New worker
-- snapshots use a stable key and can be safely upserted.
CREATE UNIQUE INDEX `uk_build_log_key` ON `build_log` (`log_key`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('018_cicd_build_observability.sql', '');
