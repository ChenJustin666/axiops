-- Permission-center completion: department administration and workflow-scoped bindings.
INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT `id`, 'department', '*', 'allow' FROM `role` WHERE `code` = 'platform_admin';

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('028_access_department_workflow_scope.sql', '');
