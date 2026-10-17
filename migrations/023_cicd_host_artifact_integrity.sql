-- Host deployments require an immutable checksum before an Agent executes an artifact.
ALTER TABLE `build_run`
  ADD COLUMN `artifact_sha256` CHAR(64) NOT NULL DEFAULT '' AFTER `artifact_url`;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('023_cicd_host_artifact_integrity.sql', '');
