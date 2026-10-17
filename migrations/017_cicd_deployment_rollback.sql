-- Audited and idempotent system rollback runs.
ALTER TABLE `workflow_run`
  ADD COLUMN `operation_type` VARCHAR(24) NOT NULL DEFAULT 'workflow' AFTER `trigger_type`,
  ADD COLUMN `source_snapshot_id` BIGINT NULL AFTER `operation_type`,
  ADD COLUMN `request_key` VARCHAR(128) NULL AFTER `source_snapshot_id`;

CREATE UNIQUE INDEX `uk_workflow_run_request_key` ON `workflow_run` (`request_key`);
CREATE INDEX `idx_workflow_run_source_snapshot` ON `workflow_run` (`source_snapshot_id`, `created_at`);

-- Built-in rollback permission. Developer remains read-only by default.
INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, 'deployment', 'rollback', 'allow'
FROM `role` r
WHERE r.code IN ('project_admin', 'platform_admin', 'operator');

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('017_cicd_deployment_rollback.sql', '');
