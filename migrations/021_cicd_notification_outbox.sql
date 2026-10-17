-- Project-scoped notification subscriptions and durable delivery outbox.
CREATE TABLE IF NOT EXISTS `notification_subscription` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `project_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '0 sends project-wide; otherwise only matching trigger user',
  `name` VARCHAR(64) NOT NULL,
  `channel` VARCHAR(24) NOT NULL COMMENT 'feishu/dingtalk/wecom/email/webhook',
  `credential_id` BIGINT NOT NULL,
  `recipient` VARCHAR(256) NOT NULL DEFAULT '',
  `on_success` TINYINT(1) NOT NULL DEFAULT 1,
  `on_failure` TINYINT(1) NOT NULL DEFAULT 1,
  `enabled` TINYINT(1) NOT NULL DEFAULT 1,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_notification_subscription_project` (`project_id`,`enabled`),
  KEY `idx_notification_subscription_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps CI/CD notification subscriptions';

CREATE TABLE IF NOT EXISTS `notification_outbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `subscription_id` BIGINT NOT NULL,
  `project_id` BIGINT NOT NULL,
  `event_type` VARCHAR(48) NOT NULL,
  `event_id` BIGINT NOT NULL,
  `payload` JSON NOT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/succeeded/dead',
  `attempts` INT NOT NULL DEFAULT 0,
  `next_attempt_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `locked_by` VARCHAR(128) NOT NULL DEFAULT '',
  `locked_at` TIMESTAMP NULL,
  `last_error` VARCHAR(1024) NOT NULL DEFAULT '',
  `delivery_key` VARCHAR(192) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_notification_delivery` (`delivery_key`),
  KEY `idx_notification_dispatch` (`status`,`next_attempt_at`,`id`),
  KEY `idx_notification_event` (`event_type`,`event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Durable CI/CD notification deliveries';

INSERT IGNORE INTO `role_permission` (`role_id`,`resource`,`action`,`effect`)
SELECT r.id,'notification',p.action,'allow' FROM `role` r JOIN (
  SELECT 'developer' role_code,'view' action UNION ALL SELECT 'tester','view' UNION ALL SELECT 'viewer','view' UNION ALL
  SELECT 'project_admin','*' UNION ALL SELECT 'platform_admin','*' UNION ALL SELECT 'operator','*'
) p ON p.role_code=r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('021_cicd_notification_outbox.sql', '');
