-- Project-scoped code source permissions. Developers and testers remain read-only.
INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, 'code_source', p.action, 'allow'
FROM `role` r
JOIN (
  SELECT 'developer' role_code, 'view' action UNION ALL
  SELECT 'tester', 'view' UNION ALL
  SELECT 'viewer', 'view' UNION ALL
  SELECT 'project_admin', '*' UNION ALL
  SELECT 'platform_admin', '*' UNION ALL
  SELECT 'operator', '*'
) p ON p.role_code=r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('019_cicd_code_source_permissions.sql', '');
