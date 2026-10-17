-- Worker lease fields prevent duplicate provider execution across workers.
ALTER TABLE `workflow_task_run`
  ADD COLUMN `locked_by` VARCHAR(128) NOT NULL DEFAULT '' AFTER `attempt`,
  ADD COLUMN `locked_at` TIMESTAMP NULL AFTER `locked_by`,
  ADD COLUMN `lease_until` TIMESTAMP NULL AFTER `locked_at`;

CREATE INDEX `idx_workflow_task_run_lease` ON `workflow_task_run` (`status`, `lease_until`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('015_cicd_workflow_task_lease.sql', '');
