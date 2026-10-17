-- Built-in governance permissions for workflow approval and cancellation.
INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, p.resource, p.action, 'allow'
FROM `role` r
JOIN (
  SELECT 'project_admin' role_code, 'workflow' resource, 'approve' action UNION ALL
  SELECT 'project_admin', 'workflow', 'cancel' UNION ALL
  SELECT 'platform_admin', 'workflow', 'approve' UNION ALL
  SELECT 'platform_admin', 'workflow', 'cancel' UNION ALL
  SELECT 'operator', 'workflow', 'approve' UNION ALL
  SELECT 'operator', 'workflow', 'cancel'
) p ON p.role_code = r.code;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('014_cicd_workflow_governance.sql', '');
