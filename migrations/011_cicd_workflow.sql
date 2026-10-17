-- Declarative workflow definitions. Runtime executions are added separately
-- so definition changes never mutate historical runs.
CREATE TABLE `workflow` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `code` VARCHAR(64) NOT NULL,
  `description` VARCHAR(512) NOT NULL DEFAULT '',
  `delivery_type` VARCHAR(24) NOT NULL COMMENT 'k8s_yaml/helm/host',
  `definition_version` BIGINT NOT NULL DEFAULT 1,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workflow_project_code` (`project_id`, `code`),
  KEY `idx_workflow_project_status` (`project_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='declarative CI/CD workflow';

CREATE TABLE `workflow_node` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `workflow_id` BIGINT NOT NULL,
  `node_key` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `node_type` VARCHAR(32) NOT NULL,
  `config_json` JSON NOT NULL,
  `depends_on` JSON NOT NULL,
  `timeout_seconds` INT NOT NULL DEFAULT 0,
  `retry_limit` INT NOT NULL DEFAULT 0,
  `ordinal` INT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_workflow_node_key` (`workflow_id`, `node_key`),
  KEY `idx_workflow_node_workflow` (`workflow_id`, `ordinal`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='workflow DAG node';

INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, p.resource, p.action, 'allow'
FROM `role` r
JOIN (
  SELECT 'developer' role_code, 'workflow' resource, 'view' action UNION ALL
  SELECT 'tester', 'workflow', 'view' UNION ALL
  SELECT 'project_admin', 'workflow', 'view' UNION ALL
  SELECT 'project_admin', 'workflow', 'create' UNION ALL
  SELECT 'project_admin', 'workflow', 'update' UNION ALL
  SELECT 'project_admin', 'workflow', 'run' UNION ALL
  SELECT 'platform_admin', 'workflow', '*' UNION ALL
  SELECT 'operator', 'workflow', '*'
) p ON p.role_code = r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('011_cicd_workflow.sql', '');
