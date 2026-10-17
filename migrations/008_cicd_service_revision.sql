-- CI/CD service delivery profile and immutable definition revisions.
ALTER TABLE `app_service`
  ADD COLUMN `delivery_type` VARCHAR(16) NOT NULL DEFAULT 'unclassified' COMMENT 'unclassified/k8s_yaml/helm/host' AFTER `deploy_type`,
  ADD COLUMN `source_type` VARCHAR(16) NOT NULL DEFAULT 'git' COMMENT 'git/manual/template' AFTER `delivery_type`,
  ADD COLUMN `source_ref` JSON NULL COMMENT 'non-secret source coordinates' AFTER `source_type`,
  ADD COLUMN `current_revision_id` BIGINT NOT NULL DEFAULT 0 AFTER `source_ref`,
  ADD COLUMN `component_count` INT NOT NULL DEFAULT 0 AFTER `current_revision_id`;

CREATE INDEX `idx_service_delivery_type` ON `app_service` (`project_id`, `delivery_type`, `status`);

CREATE TABLE `service_revision_sequence` (
  `service_id` BIGINT NOT NULL,
  `next_revision` BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='service revision number allocator';

CREATE TABLE `service_revision` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `service_id` BIGINT NOT NULL,
  `revision_number` BIGINT NOT NULL,
  `source_commit` VARCHAR(128) NOT NULL DEFAULT '',
  `definition` LONGTEXT NOT NULL,
  `definition_format` VARCHAR(16) NOT NULL COMMENT 'k8s_yaml/helm_chart/host_spec',
  `content_hash` CHAR(64) NOT NULL,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_service_revision_number` (`service_id`, `revision_number`),
  KEY `idx_service_revision_hash` (`service_id`, `content_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='immutable service definition revision';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('008_cicd_service_revision.sql', '');
