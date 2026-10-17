-- 将平台级代码源、镜像仓库和 CI 构建资源统一放入系统管理。
SET NAMES utf8mb4;

INSERT IGNORE INTO `menu` (`name`, `code`, `parent_id`, `path`, `icon`, `sort_order`, `type`, `visible`)
VALUES ('构建资源', 'system.resources', 0, '/system/resources', 'cpu', 93, 2, 1);

-- 平台管理员和已有系统菜单可见角色都可以进入平台资源配置。
INSERT IGNORE INTO `role_menu` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `role` r
JOIN `menu` m ON m.code = 'system.resources'
WHERE r.code IN ('platform_admin', 'operator', 'project_admin', 'developer', 'tester', 'viewer');

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('034_system_build_resources_menu.sql', '');
