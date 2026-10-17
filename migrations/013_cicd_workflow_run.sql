-- Immutable workflow execution snapshots and per-node task state.
CREATE TABLE `workflow_run` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `workflow_id` BIGINT NOT NULL,
  `project_id` BIGINT NOT NULL,
  `environment_id` BIGINT NOT NULL,
  `definition_version` BIGINT NOT NULL,
  `trigger_type` VARCHAR(24) NOT NULL DEFAULT 'manual',
  `trigger_user_id` BIGINT NOT NULL DEFAULT 0,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `context_snapshot` JSON NOT NULL,
  `error_message` VARCHAR(1024) NOT NULL DEFAULT '',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `started_at` TIMESTAMP NULL,
  `finished_at` TIMESTAMP NULL,
  PRIMARY KEY (`id`),
  KEY `idx_workflow_run_project` (`project_id`, `created_at`),
  KEY `idx_workflow_run_workflow` (`workflow_id`, `created_at`),
  KEY `idx_workflow_run_environment` (`environment_id`, `created_at`),
  KEY `idx_workflow_run_status` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='immutable workflow execution snapshot';

CREATE TABLE `workflow_task_run` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `workflow_run_id` BIGINT NOT NULL,
  `node_key` VARCHAR(64) NOT NULL,
  `node_type` VARCHAR(32) NOT NULL,
  `depends_on` JSON NOT NULL,
  `node_snapshot` JSON NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `attempt` INT NOT NULL DEFAULT 0,
  `provider_run_id` VARCHAR(256) NOT NULL DEFAULT '',
  `log_ref` VARCHAR(512) NOT NULL DEFAULT '',
  `error_message` VARCHAR(1024) NOT NULL DEFAULT '',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `started_at` TIMESTAMP NULL,
  `finished_at` TIMESTAMP NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workflow_task_run_node` (`workflow_run_id`, `node_key`),
  KEY `idx_workflow_task_run_status` (`workflow_run_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='workflow node execution state';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('013_cicd_workflow_run.sql', '');
