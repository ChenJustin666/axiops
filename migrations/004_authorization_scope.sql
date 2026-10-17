-- Fine-grained role bindings. Existing user_role and project_member records remain valid.
CREATE TABLE IF NOT EXISTS `schema_migration` (
    `version` VARCHAR(128) NOT NULL,
    `checksum` CHAR(64) NOT NULL DEFAULT '',
    `applied_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps database migration history';

SET @axiops_has_binding_scope := (
    SELECT COUNT(*) FROM information_schema.columns
    WHERE table_schema = DATABASE() AND table_name = 'role' AND column_name = 'binding_scope'
);
SET @axiops_sql := IF(
    @axiops_has_binding_scope = 0,
    'ALTER TABLE `role` ADD COLUMN `binding_scope` VARCHAR(16) NOT NULL DEFAULT ''scoped'' COMMENT ''platform/scoped'' AFTER `type`',
    'SELECT 1'
);
PREPARE axiops_stmt FROM @axiops_sql;
EXECUTE axiops_stmt;
DEALLOCATE PREPARE axiops_stmt;

UPDATE `role` SET `binding_scope` = 'platform'
WHERE `code` IN ('super_admin', 'platform_admin', 'operator', 'viewer');

CREATE TABLE IF NOT EXISTS `role_binding` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `subject_type` VARCHAR(16) NOT NULL COMMENT 'user/department',
    `subject_id` BIGINT NOT NULL,
    `role_id` BIGINT NOT NULL,
    `scope_type` VARCHAR(32) NOT NULL COMMENT 'platform/project/service/build/cluster/environment',
    `scope_id` BIGINT NOT NULL DEFAULT 0,
    `environment` VARCHAR(32) NOT NULL DEFAULT '*',
    `status` TINYINT NOT NULL DEFAULT 1,
    `expires_at` TIMESTAMP NULL,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_binding` (`subject_type`, `subject_id`, `role_id`, `scope_type`, `scope_id`, `environment`),
    KEY `idx_role_binding_subject` (`subject_type`, `subject_id`, `status`),
    KEY `idx_role_binding_scope` (`scope_type`, `scope_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Scoped user role bindings';

SET @axiops_has_role_permission_index := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'role_permission' AND index_name = 'uk_role_permission'
);
SET @axiops_sql := IF(
    @axiops_has_role_permission_index = 0,
    'ALTER TABLE `role_permission` ADD UNIQUE KEY `uk_role_permission` (`role_id`, `resource`, `action`, `effect`)',
    'SELECT 1'
);
PREPARE axiops_stmt FROM @axiops_sql;
EXECUTE axiops_stmt;
DEALLOCATE PREPARE axiops_stmt;

-- Built-in role templates. Custom roles can use the same action vocabulary.
INSERT INTO `role` (`name`, `code`, `description`, `type`, `created_by`)
SELECT '测试人员', 'tester', '测试环境的只读、构建和验证权限', 'system', 0
WHERE NOT EXISTS (SELECT 1 FROM `role` WHERE `code` = 'tester');

INSERT IGNORE INTO `role_permission` (`role_id`, `resource`, `action`, `effect`)
SELECT r.id, p.resource, p.action, 'allow'
FROM `role` r
JOIN (
    SELECT 'developer' role_code, 'project' resource, 'view' action UNION ALL
    SELECT 'developer', 'cluster', 'view' UNION ALL
    SELECT 'developer', 'monitoring', 'view' UNION ALL
    SELECT 'developer', 'logging', 'view' UNION ALL
    SELECT 'developer', 'service', 'view' UNION ALL
    SELECT 'developer', 'build', 'view' UNION ALL
    SELECT 'developer', 'build', 'run' UNION ALL
    SELECT 'developer', 'build_log', 'view' UNION ALL
    SELECT 'developer', 'build_config', 'view' UNION ALL
    SELECT 'developer', 'build_template', 'view' UNION ALL
    SELECT 'tester', 'project', 'view' UNION ALL
    SELECT 'tester', 'cluster', 'view' UNION ALL
    SELECT 'tester', 'monitoring', 'view' UNION ALL
    SELECT 'tester', 'logging', 'view' UNION ALL
    SELECT 'tester', 'service', 'view' UNION ALL
    SELECT 'tester', 'build', 'view' UNION ALL
    SELECT 'tester', 'build', 'run' UNION ALL
    SELECT 'tester', 'build_log', 'view' UNION ALL
    SELECT 'tester', 'build_config', 'view' UNION ALL
    SELECT 'tester', 'build_template', 'view' UNION ALL
    SELECT 'project_admin', 'project', 'view' UNION ALL
    SELECT 'project_admin', 'project', 'update' UNION ALL
    SELECT 'project_admin', 'project_member', '*' UNION ALL
    SELECT 'project_admin', 'cluster', 'view' UNION ALL
    SELECT 'project_admin', 'credential', '*' UNION ALL
    SELECT 'project_admin', 'service', '*' UNION ALL
    SELECT 'project_admin', 'build', '*' UNION ALL
    SELECT 'project_admin', 'build_log', 'view' UNION ALL
    SELECT 'project_admin', 'build_config', '*' UNION ALL
    SELECT 'platform_admin', 'user', '*' UNION ALL
    SELECT 'platform_admin', 'role', '*' UNION ALL
    SELECT 'platform_admin', 'role_binding', '*' UNION ALL
    SELECT 'platform_admin', 'menu', '*' UNION ALL
    SELECT 'platform_admin', 'project', '*' UNION ALL
    SELECT 'platform_admin', 'cluster', '*' UNION ALL
    SELECT 'platform_admin', 'credential', '*' UNION ALL
    SELECT 'operator', 'project', '*' UNION ALL
    SELECT 'operator', 'project_member', '*' UNION ALL
    SELECT 'operator', 'cluster', '*' UNION ALL
    SELECT 'operator', 'credential', '*' UNION ALL
    SELECT 'operator', 'service', '*' UNION ALL
    SELECT 'operator', 'build', '*' UNION ALL
    SELECT 'operator', 'build_log', '*' UNION ALL
    SELECT 'operator', 'build_config', '*' UNION ALL
    SELECT 'operator', 'build_template', '*' UNION ALL
    SELECT 'operator', 'release', '*' UNION ALL
    SELECT 'operator', 'monitoring', '*' UNION ALL
    SELECT 'operator', 'logging', '*'
) p ON p.role_code = r.code;

INSERT IGNORE INTO `menu` (`name`, `code`, `parent_id`, `path`, `icon`, `sort_order`, `type`, `visible`) VALUES
('总览', 'overview', 0, '/overview', 'dashboard', 10, 2, 1),
('项目', 'projects', 0, '/projects', 'folder', 20, 2, 1),
('CI/CD', 'cicd', 0, '/cicd', 'git-branch', 30, 2, 1),
('Kubernetes', 'kubernetes', 0, '/kubernetes', 'boxes', 40, 2, 1),
('监控', 'monitoring', 0, '/monitoring', 'chart-no-axes-combined', 50, 2, 1),
('日志', 'logging', 0, '/logging', 'logs', 60, 2, 1),
('资产', 'assets', 0, '/assets', 'server', 70, 2, 1),
('用户管理', 'system.users', 0, '/system/users', 'users', 90, 2, 1),
('角色权限', 'system.roles', 0, '/system/roles', 'shield-check', 91, 2, 1),
('授权管理', 'system.access', 0, '/system/access', 'key-round', 92, 2, 1);

INSERT IGNORE INTO `role_menu` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `role` r
JOIN `menu` m
WHERE r.code = 'platform_admin';

INSERT IGNORE INTO `role_menu` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `role` r
JOIN `menu` m ON m.code IN ('overview', 'projects', 'cicd', 'kubernetes', 'monitoring', 'logging', 'assets')
WHERE r.code IN ('operator', 'project_admin', 'developer', 'tester', 'viewer');

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('004_authorization_scope.sql', '');
