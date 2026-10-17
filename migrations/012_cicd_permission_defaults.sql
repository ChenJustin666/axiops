-- Align built-in developer permissions with the read-only CI/CD policy.
-- Custom roles and explicit scoped bindings are not changed.
DELETE FROM `role_permission`
WHERE `role_id` = (SELECT `id` FROM `role` WHERE `code` = 'developer' LIMIT 1)
  AND `resource` = 'build' AND `action` = 'run' AND `effect` = 'allow';

DELETE FROM `role_permission`
WHERE `role_id` = (SELECT `id` FROM `role` WHERE `code` = 'developer' LIMIT 1)
  AND `resource` = 'workflow' AND `action` = 'run' AND `effect` = 'allow';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('012_cicd_permission_defaults.sql', '');
