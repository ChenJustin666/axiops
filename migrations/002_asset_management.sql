-- AxiOps Phase 2: 资产管理表
-- Migration: 002_asset_management.sql

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

-- ----------------------------
-- 集群表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `cluster` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '集群名称',
    `code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '集群编码',
    `description` VARCHAR(512) NOT NULL DEFAULT '',
    `provider` VARCHAR(32) NOT NULL DEFAULT 'self' COMMENT '供应商：self/aliyun/aws/tencent',
    `region` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '区域',
    `version` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'K8s 版本',
    `api_server` VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'API Server 地址',
    `kubeconfig` TEXT COMMENT 'kubeconfig 内容（加密存储）',
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/connected/disconnected/error',
    `node_count` INT NOT NULL DEFAULT 0 COMMENT '节点数',
    `cpu_total` VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'CPU 总量',
    `memory_total` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '内存总量',
    `last_heartbeat` TIMESTAMP NULL COMMENT '最后心跳',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_cluster_code` (`code`),
    KEY `idx_cluster_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='集群表';

-- ----------------------------
-- 环境表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `environment` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '环境名称',
    `code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '环境编码：dev/staging/prod',
    `cluster_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联集群',
    `namespace` VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'K8s namespace',
    `description` VARCHAR(256) NOT NULL DEFAULT '',
    `is_production` TINYINT NOT NULL DEFAULT 0 COMMENT '是否生产环境',
    `deploy_approval` TINYINT NOT NULL DEFAULT 0 COMMENT '是否需要审批',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联项目（0=全局）',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_env_cluster` (`cluster_id`),
    KEY `idx_env_project` (`project_id`),
    KEY `idx_env_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='环境表';

-- ----------------------------
-- 凭据表（统一管理各类密钥）
-- ----------------------------
CREATE TABLE IF NOT EXISTS `credential` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '凭据名称',
    `type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '类型：git_token/harbor_auth/kubeconfig/ssh_key/secret',
    `description` VARCHAR(256) NOT NULL DEFAULT '',
    `data` TEXT NOT NULL COMMENT '加密后的凭据数据（JSON）',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '所属项目（0=全局）',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_cred_type` (`type`),
    KEY `idx_cred_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='凭据表';

-- ----------------------------
-- 代码源表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `code_source` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '名称',
    `type` VARCHAR(16) NOT NULL DEFAULT 'gitlab' COMMENT '类型：gitlab/github/gitee',
    `url` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '服务地址',
    `credential_id` BIGINT NOT NULL DEFAULT 0 COMMENT '认证凭据 ID',
    `status` VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '所属项目（0=全局）',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_cs_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='代码源表';

-- ----------------------------
-- 镜像仓库表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `image_registry` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '名称',
    `type` VARCHAR(16) NOT NULL DEFAULT 'harbor' COMMENT '类型：harbor/dockerhub/acr/ecr',
    `url` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '仓库地址',
    `credential_id` BIGINT NOT NULL DEFAULT 0 COMMENT '认证凭据 ID',
    `is_default` TINYINT NOT NULL DEFAULT 0 COMMENT '是否默认仓库',
    `status` VARCHAR(16) NOT NULL DEFAULT 'active',
    `project_id` BIGINT NOT NULL DEFAULT 0 COMMENT '所属项目（0=全局）',
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_ir_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='镜像仓库表';
