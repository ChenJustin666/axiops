-- Portable namespace choices; these do not create namespaces in the registry.
ALTER TABLE `image_registry`
  ADD COLUMN `namespaces` JSON NULL COMMENT 'Configured image namespace choices' AFTER `url`;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('035_cicd_registry_namespaces.sql', '');
