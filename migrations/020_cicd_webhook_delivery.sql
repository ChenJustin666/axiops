-- Authenticated Git webhook credentials and idempotent delivery records.
ALTER TABLE `code_source`
  ADD COLUMN `webhook_credential_id` BIGINT NULL AFTER `credential_id`;

ALTER TABLE `build_run`
  ADD COLUMN `trigger_key` VARCHAR(256) NULL AFTER `trigger_user_id`,
  ADD UNIQUE KEY `uk_build_run_trigger_key` (`trigger_key`);

CREATE TABLE IF NOT EXISTS `webhook_delivery` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `source_id` BIGINT NOT NULL,
  `delivery_id` VARCHAR(128) NOT NULL,
  `provider` VARCHAR(16) NOT NULL DEFAULT '',
  `event_type` VARCHAR(32) NOT NULL DEFAULT '',
  `branch` VARCHAR(256) NOT NULL DEFAULT '',
  `commit_id` VARCHAR(128) NOT NULL DEFAULT '',
  `status` VARCHAR(16) NOT NULL DEFAULT 'received',
  `build_id` BIGINT NOT NULL DEFAULT 0,
  `error_message` VARCHAR(512) NOT NULL DEFAULT '',
  `payload_hash` CHAR(64) NOT NULL DEFAULT '',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_webhook_delivery` (`source_id`,`delivery_id`),
  KEY `idx_webhook_delivery_created` (`created_at`), KEY `idx_webhook_delivery_build` (`build_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps authenticated Git webhook deliveries';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('020_cicd_webhook_delivery.sql', '');
