-- Persist independent image and host artifact integrity verification results.
ALTER TABLE `build_run`
  ADD COLUMN `image_verification_status` VARCHAR(24) NOT NULL DEFAULT 'unverified' AFTER `artifact_size`,
  ADD COLUMN `artifact_verification_status` VARCHAR(24) NOT NULL DEFAULT 'unverified' AFTER `image_verification_status`,
  ADD COLUMN `output_verification_error` VARCHAR(1024) NOT NULL DEFAULT '' AFTER `artifact_verification_status`,
  ADD COLUMN `outputs_verified_at` DATETIME NULL AFTER `output_verification_error`;

CREATE INDEX `idx_build_run_artifact_verified` ON `build_run` (`artifact_verification_status`,`status`,`id`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('027_cicd_build_output_verification.sql', '');
