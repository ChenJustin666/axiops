-- Durable official Helm SDK executions and audited release revisions.
ALTER TABLE `deployment_snapshot`
  ADD COLUMN `helm_release_name` VARCHAR(53) NOT NULL DEFAULT '' AFTER `values_snapshot`,
  ADD COLUMN `helm_revision` INT NOT NULL DEFAULT 0 AFTER `helm_release_name`,
  ADD KEY `idx_deployment_snapshot_helm` (`cluster_id`,`namespace`,`helm_release_name`,`helm_revision`);

CREATE TABLE IF NOT EXISTS `helm_execution` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `idempotency_key` VARCHAR(128) NOT NULL,
  `request_hash` CHAR(64) NOT NULL,
  `workflow_run_id` BIGINT NOT NULL DEFAULT 0,
  `node_key` VARCHAR(64) NOT NULL DEFAULT '',
  `operation` VARCHAR(16) NOT NULL COMMENT 'upgrade/rollback',
  `cluster_id` BIGINT NOT NULL,
  `namespace` VARCHAR(63) NOT NULL,
  `release_name` VARCHAR(53) NOT NULL,
  `target_revision` INT NOT NULL DEFAULT 0,
  `source_revision` INT NOT NULL DEFAULT 0,
  `marker` VARCHAR(256) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT 'running',
  `helm_revision` INT NOT NULL DEFAULT 0,
  `error_message` VARCHAR(1024) NOT NULL DEFAULT '',
  `timeout_seconds` INT NOT NULL DEFAULT 600,
  `started_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `finished_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_helm_execution_idempotency` (`idempotency_key`),
  KEY `idx_helm_execution_workflow` (`workflow_run_id`,`node_key`),
  KEY `idx_helm_execution_release` (`cluster_id`,`namespace`,`release_name`,`id`),
  KEY `idx_helm_execution_status` (`status`,`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Durable official Helm SDK executions';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('025_cicd_helm_execution.sql', '');
