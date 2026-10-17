-- Environment-owned Kubernetes resources. These are layered on top of a service
-- revision and are rendered only for the selected environment.
CREATE TABLE IF NOT EXISTS `environment_resource` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `environment_id` BIGINT NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `kind` VARCHAR(32) NOT NULL COMMENT 'Ingress/ConfigMap/Secret/PersistentVolumeClaim',
  `manifest` MEDIUMTEXT NOT NULL,
  `revision` BIGINT NOT NULL DEFAULT 1,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_environment_resource_name` (`environment_id`, `name`),
  KEY `idx_environment_resource_environment` (`environment_id`, `kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='environment-specific Kubernetes resources';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('030_cicd_environment_resources.sql', '');
