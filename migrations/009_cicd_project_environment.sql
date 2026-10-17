-- Project-scoped runtime environments and environment variable sets.
CREATE TABLE `project_environment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `code` VARCHAR(32) NOT NULL,
  `environment_type` VARCHAR(24) NOT NULL COMMENT 'k8s_namespace/host_group',
  `cluster_id` BIGINT NOT NULL DEFAULT 0,
  `namespace` VARCHAR(63) NOT NULL DEFAULT '',
  `host_group_ref` VARCHAR(128) NOT NULL DEFAULT '',
  `binding_status` VARCHAR(24) NOT NULL DEFAULT 'validated' COMMENT 'validated/pending_validation/invalid',
  `is_production` TINYINT(1) NOT NULL DEFAULT 0,
  `deploy_approval` TINYINT(1) NOT NULL DEFAULT 0,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_environment_code` (`project_id`, `code`),
  KEY `idx_project_environment_cluster` (`cluster_id`, `namespace`),
  KEY `idx_project_environment_host_group` (`host_group_ref`),
  KEY `idx_project_environment_status` (`project_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='project runtime environment';

CREATE TABLE `variable_set` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT NOT NULL,
  `environment_id` BIGINT NOT NULL DEFAULT 0,
  `service_id` BIGINT NOT NULL DEFAULT 0,
  `name` VARCHAR(64) NOT NULL,
  `scope` VARCHAR(16) NOT NULL COMMENT 'project/environment/service',
  `revision` BIGINT NOT NULL DEFAULT 1,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_variable_set_scope_name` (`scope`, `project_id`, `environment_id`, `service_id`, `name`),
  KEY `idx_variable_set_environment` (`environment_id`),
  KEY `idx_variable_set_service` (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='layered variable set';

CREATE TABLE `variable_item` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `variable_set_id` BIGINT NOT NULL,
  `variable_key` VARCHAR(128) NOT NULL,
  `value_text` MEDIUMTEXT NULL,
  `credential_id` BIGINT NOT NULL DEFAULT 0,
  `is_secret` TINYINT(1) NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_variable_item_key` (`variable_set_id`, `variable_key`),
  KEY `idx_variable_item_credential` (`credential_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='non-secret value or credential reference';

INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, p.resource, p.action, 'allow'
FROM `role` r
JOIN (
  SELECT 'developer' role_code, 'environment' resource, 'view' action UNION ALL
  SELECT 'tester', 'environment', 'view' UNION ALL
  SELECT 'viewer', 'environment', 'view' UNION ALL
  SELECT 'project_admin', 'environment', 'view' UNION ALL
  SELECT 'project_admin', 'environment', 'create' UNION ALL
  SELECT 'project_admin', 'environment', 'update' UNION ALL
  SELECT 'project_admin', 'environment', 'delete' UNION ALL
  SELECT 'platform_admin', 'environment', '*' UNION ALL
  SELECT 'operator', 'environment', '*'
) p ON p.role_code = r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('009_cicd_project_environment.sql', '');
