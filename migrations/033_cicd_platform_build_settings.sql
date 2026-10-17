-- Platform-wide CI runner defaults. Values are read by workflow-worker at startup;
-- project/service settings may override them where explicitly configured.
CREATE TABLE IF NOT EXISTS `build_platform_settings` (
  `id` TINYINT NOT NULL,
  `enabled` TINYINT(1) NOT NULL DEFAULT 0,
  `cluster_id` BIGINT NOT NULL DEFAULT 0,
  `namespace` VARCHAR(63) NOT NULL DEFAULT 'axiops-cicd',
  `builder_image` VARCHAR(512) NOT NULL DEFAULT '',
  `build_command` MEDIUMTEXT NOT NULL,
  `timeout_seconds` INT NOT NULL DEFAULT 1800,
  `resource_cpu` VARCHAR(32) NOT NULL DEFAULT '2',
  `resource_memory` VARCHAR(32) NOT NULL DEFAULT '4Gi',
  `cache_pvc_name` VARCHAR(253) NOT NULL DEFAULT '',
  `cache_dir` VARCHAR(256) NOT NULL DEFAULT '/workspace/.axiops-cache',
  `job_retention_seconds` INT NOT NULL DEFAULT 86400,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Platform CI runner defaults';

INSERT INTO `build_platform_settings` (`id`, `build_command`)
VALUES (1, '')
ON DUPLICATE KEY UPDATE `id`=VALUES(`id`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('033_cicd_platform_build_settings.sql', '');
