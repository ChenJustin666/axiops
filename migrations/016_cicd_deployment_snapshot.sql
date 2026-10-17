-- Immutable deployment output used for audit, diff and rollback.
CREATE TABLE `deployment_snapshot` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `workflow_run_id` BIGINT NOT NULL,
  `workflow_task_run_id` BIGINT NOT NULL,
  `node_key` VARCHAR(64) NOT NULL,
  `project_id` BIGINT NOT NULL,
  `environment_id` BIGINT NOT NULL,
  `service_id` BIGINT NOT NULL,
  `service_revision_id` BIGINT NOT NULL,
  `cluster_id` BIGINT NOT NULL,
  `namespace` VARCHAR(128) NOT NULL,
  `provider` VARCHAR(32) NOT NULL,
  `manifest_digest` CHAR(64) NOT NULL,
  `rendered_manifest` MEDIUMTEXT NOT NULL,
  `values_snapshot` JSON NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'prepared',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `applied_at` TIMESTAMP NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_deployment_snapshot_run_node` (`workflow_run_id`, `node_key`, `provider`),
  KEY `idx_deployment_snapshot_task` (`workflow_task_run_id`),
  KEY `idx_deployment_snapshot_environment` (`environment_id`, `service_id`, `created_at`),
  KEY `idx_deployment_snapshot_revision` (`service_revision_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='immutable deployment output for rollback';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('016_cicd_deployment_snapshot.sql', '');
