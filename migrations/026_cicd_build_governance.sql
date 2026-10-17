-- Dynamic build concurrency policies and query indexes for efficiency metrics.
CREATE TABLE IF NOT EXISTS `build_concurrency_policy` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `scope_type` VARCHAR(16) NOT NULL COMMENT 'platform/project',
  `scope_id` BIGINT NOT NULL DEFAULT 0,
  `max_running` INT NOT NULL,
  `queue_timeout_seconds` INT NOT NULL DEFAULT 7200,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_build_concurrency_scope` (`scope_type`,`scope_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Build concurrency and queue timeout policy';

INSERT INTO `build_concurrency_policy` (`scope_type`,`scope_id`,`max_running`,`queue_timeout_seconds`)
VALUES ('platform',0,50,7200)
ON DUPLICATE KEY UPDATE `scope_type`=VALUES(`scope_type`);

CREATE INDEX `idx_build_run_metrics` ON `build_run` (`project_id`,`service_id`,`created_at`,`status`);

INSERT IGNORE INTO `role_permission` (`role_id`,`resource`,`action`,`effect`)
SELECT r.id,'build_governance',p.action,'allow' FROM `role` r JOIN (
  SELECT 'project_admin' role_code,'view' action UNION ALL
  SELECT 'project_admin','update' UNION ALL
  SELECT 'platform_admin','*' UNION ALL
  SELECT 'operator','*'
) p ON p.role_code=r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('026_cicd_build_governance.sql', '');
