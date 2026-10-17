-- Pull-based host delivery for private and restricted networks.
CREATE TABLE IF NOT EXISTS `host_agent_registration_token` (
  `id` BIGINT NOT NULL AUTO_INCREMENT, `project_id` BIGINT NOT NULL, `host_group_ref` VARCHAR(128) NOT NULL,
  `token_hash` CHAR(64) NOT NULL, `expires_at` TIMESTAMP NOT NULL, `max_uses` INT NOT NULL DEFAULT 1,
  `used_count` INT NOT NULL DEFAULT 0, `created_by` BIGINT NOT NULL DEFAULT 0, `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_host_agent_registration_token` (`token_hash`), KEY `idx_host_agent_registration_expiry` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='One-time host agent registration tokens';

CREATE TABLE IF NOT EXISTS `host_agent` (
  `id` BIGINT NOT NULL AUTO_INCREMENT, `project_id` BIGINT NOT NULL, `host_group_ref` VARCHAR(128) NOT NULL,
  `asset_ref` VARCHAR(128) NOT NULL, `name` VARCHAR(128) NOT NULL, `token_hash` CHAR(64) NOT NULL,
  `version` VARCHAR(32) NOT NULL DEFAULT '', `os` VARCHAR(32) NOT NULL DEFAULT '', `arch` VARCHAR(32) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT 'online', `metadata` JSON NULL, `last_heartbeat` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_host_agent_asset` (`project_id`,`asset_ref`), UNIQUE KEY `uk_host_agent_token` (`token_hash`),
  KEY `idx_host_agent_group` (`project_id`,`host_group_ref`,`status`), KEY `idx_host_agent_heartbeat` (`last_heartbeat`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps pull-based host agents';

CREATE TABLE IF NOT EXISTS `host_deployment_command` (
  `id` BIGINT NOT NULL AUTO_INCREMENT, `workflow_run_id` BIGINT NOT NULL, `task_key` VARCHAR(64) NOT NULL,
  `project_id` BIGINT NOT NULL, `service_id` BIGINT NOT NULL, `service_revision_id` BIGINT NOT NULL, `agent_id` BIGINT NOT NULL,
  `artifact_url` VARCHAR(1024) NOT NULL, `artifact_sha256` CHAR(64) NOT NULL, `deploy_script` MEDIUMTEXT NOT NULL,
  `health_script` TEXT NOT NULL, `rollback_script` MEDIUMTEXT NOT NULL, `timeout_seconds` INT NOT NULL DEFAULT 900,
  `status` VARCHAR(16) NOT NULL DEFAULT 'pending', `attempts` INT NOT NULL DEFAULT 0, `locked_at` TIMESTAMP NULL,
  `lease_until` TIMESTAMP NULL, `log_text` MEDIUMTEXT, `log_truncated` TINYINT(1) NOT NULL DEFAULT 0,
  `error_message` VARCHAR(1024) NOT NULL DEFAULT '', `started_at` TIMESTAMP NULL, `finished_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_host_deploy_target` (`workflow_run_id`,`task_key`,`agent_id`),
  KEY `idx_host_deploy_agent` (`agent_id`,`status`,`id`), KEY `idx_host_deploy_task` (`workflow_run_id`,`task_key`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Durable host deployment commands';

INSERT IGNORE INTO `role_permission` (`role_id`,`resource`,`action`,`effect`)
SELECT r.id,'host_agent',p.action,'allow' FROM `role` r JOIN (
  SELECT 'developer' role_code,'view' action UNION ALL SELECT 'tester','view' UNION ALL SELECT 'viewer','view' UNION ALL
  SELECT 'project_admin','*' UNION ALL SELECT 'platform_admin','*' UNION ALL SELECT 'operator','*'
) p ON p.role_code=r.code;
INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('022_cicd_host_agent.sql', '');
