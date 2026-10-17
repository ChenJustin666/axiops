-- AxiOps Phase 3: CI/CD 核心表
-- Migration: 003_cicd_core.sql

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- ----------------------------
-- 应用/服务表（一个应用对应一个代码仓库+构建+部署配置）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `app_service` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '服务名称',
    `code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '服务编码',
    `description` VARCHAR(512) NOT NULL DEFAULT '',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '所属项目',
    `language` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '语言：go/java/node/python',
    `framework` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '框架：gin/spring/next',
    `repo_url` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '代码仓库地址',
    `repo_branch` VARCHAR(64) NOT NULL DEFAULT 'main' COMMENT '默认分支',
    `code_source_id` BIGINT NOT NULL DEFAULT 0 COMMENT '代码源 ID',
    `registry_id` BIGINT NOT NULL DEFAULT 0 COMMENT '镜像仓库 ID',
    `image_name` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '镜像名称（不含 tag）',
    `dockerfile_path` VARCHAR(256) NOT NULL DEFAULT 'Dockerfile' COMMENT 'Dockerfile 路径',
    `build_context` VARCHAR(256) NOT NULL DEFAULT '.' COMMENT '构建上下文',
    `deploy_type` VARCHAR(16) NOT NULL DEFAULT 'k8s' COMMENT '部署类型：k8s/host/serverless',
    `status` VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/archived',
    `owner_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '负责人',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_svc_project_code` (`project_id`, `code`),
    KEY `idx_svc_project` (`project_id`),
    KEY `idx_svc_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用服务表';

-- ----------------------------
-- 构建记录表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `build_record` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `service_id` BIGINT NOT NULL COMMENT '服务 ID',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '项目 ID',
    `build_number` INT NOT NULL DEFAULT 0 COMMENT '构建编号',
    `branch` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '构建分支',
    `commit_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'Git commit SHA',
    `commit_msg` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Commit 消息',
    `image_tag` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '构建产出镜像 tag',
    `image_full` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '完整镜像地址',
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/building/success/failed/cancelled',
    `trigger_type` VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT '触发方式：manual/webhook/schedule',
    `trigger_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发人',
    `engine` VARCHAR(16) NOT NULL DEFAULT 'docker' COMMENT '构建引擎：docker/jenkins/tekton',
    `engine_build_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '引擎侧构建 ID',
    `log_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '构建日志地址',
    `duration_sec` INT NOT NULL DEFAULT 0 COMMENT '构建耗时秒',
    `started_at` TIMESTAMP NULL,
    `finished_at` TIMESTAMP NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_build_service` (`service_id`, `build_number`),
    KEY `idx_build_project` (`project_id`),
    KEY `idx_build_status` (`status`),
    KEY `idx_build_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建记录表';

-- ----------------------------
-- 流水线模板表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pipeline` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '流水线名称',
    `description` VARCHAR(256) NOT NULL DEFAULT '',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '所属项目',
    `service_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联服务（0=通用模板）',
    `stages` JSON NOT NULL COMMENT '阶段定义 JSON',
    `trigger_rules` JSON COMMENT '触发规则 JSON',
    `status` VARCHAR(16) NOT NULL DEFAULT 'active',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pipeline_project` (`project_id`),
    KEY `idx_pipeline_service` (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线模板表';

-- ----------------------------
-- 流水线执行记录表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pipeline_run` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `pipeline_id` BIGINT NOT NULL COMMENT '流水线 ID',
    `service_id` BIGINT NOT NULL DEFAULT 0,
    `project_id` BIGINT NOT NULL DEFAULT 0,
    `run_number` INT NOT NULL DEFAULT 0 COMMENT '执行编号',
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/running/success/failed/cancelled',
    `stages_status` JSON COMMENT '各阶段状态 JSON',
    `trigger_type` VARCHAR(16) NOT NULL DEFAULT 'manual',
    `trigger_user_id` BIGINT NOT NULL DEFAULT 0,
    `branch` VARCHAR(64) NOT NULL DEFAULT '',
    `commit_id` VARCHAR(64) NOT NULL DEFAULT '',
    `duration_sec` INT NOT NULL DEFAULT 0,
    `started_at` TIMESTAMP NULL,
    `finished_at` TIMESTAMP NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pr_pipeline` (`pipeline_id`, `run_number`),
    KEY `idx_pr_service` (`service_id`),
    KEY `idx_pr_project` (`project_id`),
    KEY `idx_pr_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线执行记录表';

-- ----------------------------
-- 发布/部署记录表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `release_record` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `service_id` BIGINT NOT NULL COMMENT '服务 ID',
    `project_id` BIGINT NOT NULL DEFAULT 0,
    `environment_id` BIGINT NOT NULL COMMENT '目标环境 ID',
    `cluster_id` BIGINT NOT NULL DEFAULT 0 COMMENT '目标集群',
    `namespace` VARCHAR(64) NOT NULL DEFAULT '',
    `release_number` INT NOT NULL DEFAULT 0 COMMENT '发布编号',
    `image_tag` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '发布镜像 tag',
    `image_full` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '完整镜像地址',
    `strategy` VARCHAR(16) NOT NULL DEFAULT 'rolling' COMMENT '发布策略：rolling/bluegreen/canary',
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/deploying/success/failed/rollback',
    `approval_status` VARCHAR(16) NOT NULL DEFAULT 'none' COMMENT 'none/pending/approved/rejected',
    `approval_user_id` BIGINT NOT NULL DEFAULT 0,
    `deploy_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '部署人',
    `rollback_from` BIGINT NOT NULL DEFAULT 0 COMMENT '回滚来源发布 ID',
    `deploy_manifest` JSON COMMENT '部署配置快照',
    `duration_sec` INT NOT NULL DEFAULT 0,
    `started_at` TIMESTAMP NULL,
    `finished_at` TIMESTAMP NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_rel_service` (`service_id`, `release_number`),
    KEY `idx_rel_env` (`environment_id`),
    KEY `idx_rel_project` (`project_id`),
    KEY `idx_rel_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布部署记录表';
