-- Identity bridge for the controlled transition from legacy autoops accounts.
-- This migration creates mappings only. Account data is imported by the
-- dedicated migration job after an inventory and approval step.
CREATE TABLE IF NOT EXISTS `schema_migration` (
    `version` VARCHAR(128) NOT NULL,
    `checksum` CHAR(64) NOT NULL DEFAULT '',
    `applied_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps database migration history';

CREATE TABLE IF NOT EXISTS `identity_link` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `provider` VARCHAR(32) NOT NULL COMMENT 'legacy_autoops or another trusted identity source',
    `external_user_id` BIGINT NOT NULL,
    `user_id` BIGINT NOT NULL,
    `external_username` VARCHAR(64) NOT NULL DEFAULT '',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=active, 0=disabled',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_identity_link_external` (`provider`, `external_user_id`),
    UNIQUE KEY `uk_identity_link_target` (`provider`, `user_id`),
    KEY `idx_identity_link_user` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='External identity to AxiOps user mapping';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('005_identity_link.sql', '');
