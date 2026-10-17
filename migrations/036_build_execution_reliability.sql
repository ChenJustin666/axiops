ALTER TABLE build_platform_settings ADD COLUMN execution_policy JSON NULL;
ALTER TABLE build_run
  ADD COLUMN terminal_observed_at DATETIME NULL,
  ADD COLUMN cleanup_pending TINYINT(1) NOT NULL DEFAULT 0,
  ADD COLUMN cleanup_retry_at DATETIME NULL,
  ADD INDEX idx_build_cleanup (cleanup_pending, cleanup_retry_at);
ALTER TABLE build_log ADD COLUMN log_expired TINYINT(1) NOT NULL DEFAULT 0;
