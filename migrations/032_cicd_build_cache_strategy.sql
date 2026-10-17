-- Build cache policy. PVC claims are pre-created by platform operators; the
-- builder never creates or expands claims implicitly.
ALTER TABLE `build_config`
  ADD COLUMN `cache_strategy` VARCHAR(16) NOT NULL DEFAULT 'emptydir' AFTER `cache_dir`,
  ADD COLUMN `cache_pvc_name` VARCHAR(253) NOT NULL DEFAULT '' AFTER `cache_strategy`;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('032_cicd_build_cache_strategy.sql', '');
