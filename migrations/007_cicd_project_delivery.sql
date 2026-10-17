-- CI/CD project delivery type compatibility migration.
-- Existing projects are deliberately not inferred from legacy deployment data.
CREATE TABLE IF NOT EXISTS `schema_migration` (
  `version` VARCHAR(128) NOT NULL,
  `checksum` CHAR(64) NOT NULL DEFAULT '',
  `applied_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps database migration history';

ALTER TABLE `project`
  ADD COLUMN `delivery_type` VARCHAR(16) NOT NULL DEFAULT 'unclassified' COMMENT 'unclassified/k8s_yaml/helm/host' AFTER `status`,
  ADD COLUMN `delivery_status` VARCHAR(24) NOT NULL DEFAULT 'pending_confirmation' COMMENT 'pending_confirmation/active' AFTER `delivery_type`;

CREATE INDEX `idx_project_delivery_type` ON `project` (`delivery_type`, `delivery_status`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('007_cicd_project_delivery.sql', '');
