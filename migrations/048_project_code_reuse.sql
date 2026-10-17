-- Deleted history retains its original code but no longer reserves it.
-- One ALTER atomically replaces the old uniqueness rule. Guard retries after
-- a successful ALTER but an interrupted migration-history write.
SET @project_code_reuse_sql = IF(
  EXISTS(SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='project' AND INDEX_NAME='uk_project_active_code'),
  'SELECT 1',
  'ALTER TABLE project DROP INDEX uk_project_code,
     ADD COLUMN active_code VARCHAR(64) GENERATED ALWAYS AS (CASE WHEN status <> ''deleted'' THEN code ELSE NULL END) STORED,
     ADD UNIQUE KEY uk_project_active_code (active_code)'
);
PREPARE project_code_reuse_stmt FROM @project_code_reuse_sql;
EXECUTE project_code_reuse_stmt;
DEALLOCATE PREPARE project_code_reuse_stmt;
