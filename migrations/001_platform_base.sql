-- AxiOps 平台底座数据库初始化
-- Migration: 001_platform_base.sql

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- ----------------------------
-- 用户表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `username` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '用户名',
    `password` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '密码（bcrypt）',
    `nickname` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '昵称',
    `email` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '邮箱',
    `phone` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '手机号',
    `avatar` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像 URL',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=启用 0=禁用',
    `department_id` BIGINT NOT NULL DEFAULT 0 COMMENT '部门 ID',
    `last_login_at` TIMESTAMP NULL COMMENT '最后登录时间',
    `last_login_ip` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '最后登录 IP',
    `created_by` BIGINT NOT NULL DEFAULT 0 COMMENT '创建人',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL COMMENT '软删除',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_username` (`username`),
    KEY `idx_user_email` (`email`),
    KEY `idx_user_department` (`department_id`),
    KEY `idx_user_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- ----------------------------
-- 部门表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `department` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '部门名称',
    `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT '上级部门 ID',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序',
    `leader_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '负责人用户 ID',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_department_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门表';

-- ----------------------------
-- 角色表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `role` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色名',
    `code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色编码（唯一标识）',
    `description` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '描述',
    `type` VARCHAR(16) NOT NULL DEFAULT 'custom' COMMENT '类型：system/custom',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';

-- ----------------------------
-- 权限策略表（角色拥有的具体权限）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `role_permission` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `role_id` BIGINT NOT NULL COMMENT '角色 ID',
    `resource` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源：project/cluster/build/release/monitor 等',
    `action` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '动作：read/write/delete/admin',
    `effect` VARCHAR(8) NOT NULL DEFAULT 'allow' COMMENT '效果：allow/deny',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rp_role` (`role_id`),
    KEY `idx_rp_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色权限策略表';

-- ----------------------------
-- 菜单表（前端菜单可见性控制）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `menu` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '菜单名',
    `code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '菜单编码',
    `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT '父菜单',
    `path` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '前端路由',
    `icon` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '图标',
    `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序',
    `type` TINYINT NOT NULL DEFAULT 1 COMMENT '类型：1=目录 2=菜单 3=按钮',
    `visible` TINYINT NOT NULL DEFAULT 1 COMMENT '是否可见',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_menu_code` (`code`),
    KEY `idx_menu_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='菜单表';

-- ----------------------------
-- 角色菜单关联表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `role_menu` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `role_id` BIGINT NOT NULL,
    `menu_id` BIGINT NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_menu` (`role_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色菜单关联表';

-- ----------------------------
-- 项目表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `project` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '项目名',
    `code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '项目编码',
    `description` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '描述',
    `status` VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态：active/archived',
    `owner_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '项目负责人',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_project_code` (`code`),
    KEY `idx_project_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';

-- ----------------------------
-- 项目成员表（用户在项目中的角色）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `project_member` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `project_id` BIGINT NOT NULL,
    `user_id` BIGINT NOT NULL,
    `role_id` BIGINT NOT NULL COMMENT '用户在该项目中的角色',
    `joined_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_pm_project_user` (`project_id`, `user_id`),
    KEY `idx_pm_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目成员表';

-- ----------------------------
-- 全局用户角色（平台级别角色绑定）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user_role` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT NOT NULL,
    `role_id` BIGINT NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ur_user_role` (`user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色绑定表';

-- ----------------------------
-- 审计日志表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `audit_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT NOT NULL DEFAULT 0,
    `username` VARCHAR(64) NOT NULL DEFAULT '',
    `action` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'create/update/delete/login/deploy/approve',
    `resource_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'user/project/build/release 等',
    `resource_id` BIGINT NOT NULL DEFAULT 0,
    `resource_name` VARCHAR(128) NOT NULL DEFAULT '',
    `project_id` BIGINT NOT NULL DEFAULT 0,
    `request_method` VARCHAR(8) NOT NULL DEFAULT '',
    `request_path` VARCHAR(256) NOT NULL DEFAULT '',
    `request_body` JSON COMMENT '脱敏后的请求体',
    `response_code` INT NOT NULL DEFAULT 0,
    `client_ip` VARCHAR(45) NOT NULL DEFAULT '',
    `user_agent` VARCHAR(256) NOT NULL DEFAULT '',
    `duration_ms` INT NOT NULL DEFAULT 0 COMMENT '请求耗时毫秒',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_audit_user` (`user_id`, `created_at`),
    KEY `idx_audit_resource` (`resource_type`, `resource_id`),
    KEY `idx_audit_project` (`project_id`, `created_at`),
    KEY `idx_audit_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志表';

-- ----------------------------
-- 初始数据
-- ----------------------------

-- 超级管理员角色
INSERT INTO `role` (`id`, `name`, `code`, `description`, `type`, `created_by`) VALUES
(1, '超级管理员', 'super_admin', '拥有所有权限', 'system', 0),
(2, '平台管理员', 'platform_admin', '平台级别管理权限', 'system', 0),
(3, '项目管理员', 'project_admin', '项目级别管理权限', 'system', 0),
(4, '开发者', 'developer', '开发人员，可构建和发布到非生产环境', 'system', 0),
(5, '运维', 'operator', '运维人员，可管理集群和发布到生产环境', 'system', 0),
(6, '只读', 'viewer', '只读权限', 'system', 0);

-- 超级管理员权限（全部允许）
INSERT INTO `role_permission` (`role_id`, `resource`, `action`, `effect`) VALUES
(1, '*', '*', 'allow');

-- 只读角色权限
INSERT INTO `role_permission` (`role_id`, `resource`, `action`, `effect`) VALUES
(6, '*', 'read', 'allow');

-- 默认管理员用户（密码：REPLACE_ME，bcrypt 加密）
INSERT INTO `user` (`id`, `username`, `password`, `nickname`, `email`, `status`, `created_by`) VALUES
(1, 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', '管理员', 'admin@axiops.io', 1, 0);

-- 绑定管理员角色
INSERT INTO `user_role` (`user_id`, `role_id`, `created_by`) VALUES
(1, 1, 0);
