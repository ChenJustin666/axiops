-- Desired service revision and runtime overrides per project environment.
CREATE TABLE `environment_service_override` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `project_environment_id` BIGINT NOT NULL,
  `service_id` BIGINT NOT NULL,
  `service_revision_id` BIGINT NOT NULL,
  `image_ref` VARCHAR(512) NOT NULL DEFAULT '',
  `replicas` INT NULL,
  `revision` BIGINT NOT NULL DEFAULT 1 COMMENT 'optimistic-lock revision',
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_environment_service_override` (`project_environment_id`, `service_id`),
  KEY `idx_environment_override_service` (`service_id`),
  KEY `idx_environment_override_revision` (`service_revision_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='desired service version in an environment';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('010_cicd_environment_service_override.sql', '');
