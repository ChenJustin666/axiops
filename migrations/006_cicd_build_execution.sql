-- Durable build execution model for the AxiOps CI/CD API.
-- The legacy build_record table is intentionally retained for compatibility.

CREATE TABLE IF NOT EXISTS `build_config` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `service_id` BIGINT NOT NULL,
    `template_type` VARCHAR(32) NOT NULL DEFAULT '',
    `build_image` VARCHAR(256) NOT NULL DEFAULT '',
    `build_command` TEXT,
    `dockerfile_path` VARCHAR(256) NOT NULL DEFAULT 'Dockerfile',
    `build_context` VARCHAR(256) NOT NULL DEFAULT '.',
    `image_repo` VARCHAR(512) NOT NULL DEFAULT '',
    `cache_enabled` TINYINT(1) NOT NULL DEFAULT 1,
    `cache_dir` VARCHAR(256) NOT NULL DEFAULT '',
    `timeout_seconds` INT NOT NULL DEFAULT 1800,
    `branch_rules` JSON NULL,
    `env_vars` JSON NULL COMMENT 'Non-secret variables only; secrets must use credential references',
    `resource_cpu` VARCHAR(32) NOT NULL DEFAULT '2',
    `resource_memory` VARCHAR(32) NOT NULL DEFAULT '4Gi',
    `provider` VARCHAR(32) NOT NULL DEFAULT 'kube_job',
    `jenkins_job_name` VARCHAR(128) NOT NULL DEFAULT '',
    `host_agent_id` BIGINT NOT NULL DEFAULT 0,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_build_config_service` (`service_id`),
    KEY `idx_build_config_provider` (`provider`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Build configuration per service';

CREATE TABLE IF NOT EXISTS `build_number_sequence` (
    `service_id` BIGINT NOT NULL,
    `current_number` INT NOT NULL DEFAULT 0,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`service_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Concurrency-safe build number allocator';

CREATE TABLE IF NOT EXISTS `build_run` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `service_id` BIGINT NOT NULL,
    `project_id` BIGINT NOT NULL,
    `pipeline_run_id` BIGINT NOT NULL DEFAULT 0,
    `build_number` INT NOT NULL,
    `config_snapshot` JSON NULL,
    `trigger_type` VARCHAR(16) NOT NULL DEFAULT 'manual',
    `trigger_user_id` BIGINT NOT NULL DEFAULT 0,
    `status` VARCHAR(16) NOT NULL DEFAULT 'queued' COMMENT 'queued/dispatching/running/cancelling/success/failed/timeout/cancelled',
    `provider` VARCHAR(32) NOT NULL,
    `git_branch` VARCHAR(128) NOT NULL DEFAULT '',
    `git_commit` VARCHAR(64) NOT NULL DEFAULT '',
    `git_commit_short` VARCHAR(16) NOT NULL DEFAULT '',
    `git_author` VARCHAR(128) NOT NULL DEFAULT '',
    `git_author_email` VARCHAR(256) NOT NULL DEFAULT '',
    `git_message` TEXT,
    `git_changed_files` JSON NULL,
    `git_compare_url` VARCHAR(512) NOT NULL DEFAULT '',
    `image_url` VARCHAR(512) NOT NULL DEFAULT '',
    `image_digest` VARCHAR(128) NOT NULL DEFAULT '',
    `image_size` BIGINT NOT NULL DEFAULT 0,
    `artifact_url` VARCHAR(512) NOT NULL DEFAULT '',
    `artifact_size` BIGINT NOT NULL DEFAULT 0,
    `job_name` VARCHAR(128) NOT NULL DEFAULT '',
    `pod_name` VARCHAR(128) NOT NULL DEFAULT '',
    `node_name` VARCHAR(128) NOT NULL DEFAULT '',
    `stages` JSON NULL,
    `queued_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `started_at` TIMESTAMP NULL,
    `finished_at` TIMESTAMP NULL,
    `duration_seconds` INT NOT NULL DEFAULT 0,
    `error_message` TEXT,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_build_run_number` (`service_id`, `build_number`),
    KEY `idx_build_run_project_created` (`project_id`, `created_at`),
    KEY `idx_build_run_service_status` (`service_id`, `status`),
    KEY `idx_build_run_status_queued` (`status`, `queued_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AxiOps build execution record';

CREATE TABLE IF NOT EXISTS `build_command` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `build_run_id` BIGINT NOT NULL,
    `command_type` VARCHAR(16) NOT NULL COMMENT 'submit/cancel',
    `provider` VARCHAR(32) NOT NULL,
    `payload` JSON NOT NULL,
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/succeeded/failed/dead',
    `attempts` INT NOT NULL DEFAULT 0,
    `next_attempt_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `locked_by` VARCHAR(128) NOT NULL DEFAULT '',
    `locked_at` TIMESTAMP NULL,
    `last_error` TEXT,
    `idempotency_key` VARCHAR(128) NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_build_command_idempotency` (`idempotency_key`),
    KEY `idx_build_command_dispatch` (`status`, `next_attempt_at`, `id`),
    KEY `idx_build_command_run` (`build_run_id`, `command_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Durable commands consumed by the CI/CD worker';

CREATE TABLE IF NOT EXISTS `build_log` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `build_run_id` BIGINT NOT NULL,
    `stage` VARCHAR(32) NOT NULL DEFAULT '',
    `log_url` VARCHAR(512) NOT NULL DEFAULT '',
    `log_text` MEDIUMTEXT,
    `line_count` INT NOT NULL DEFAULT 0,
    `started_at` TIMESTAMP NULL,
    `finished_at` TIMESTAMP NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_build_log_run` (`build_run_id`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Build log metadata and bounded inline content';

CREATE TABLE IF NOT EXISTS `build_template` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL,
    `type` VARCHAR(32) NOT NULL,
    `description` VARCHAR(512) NOT NULL DEFAULT '',
    `build_image` VARCHAR(256) NOT NULL DEFAULT '',
    `build_command` TEXT,
    `cache_dir` VARCHAR(256) NOT NULL DEFAULT '',
    `detect_files` JSON NULL,
    `is_builtin` TINYINT(1) NOT NULL DEFAULT 0,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_build_template_type_builtin` (`type`, `is_builtin`),
    KEY `idx_build_template_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Reusable build templates';

INSERT INTO `build_template`
    (`name`, `type`, `description`, `build_image`, `build_command`, `cache_dir`, `detect_files`, `is_builtin`)
VALUES
    ('Go', 'go', 'Built-in Go template', 'golang:1.25-alpine', 'go test ./... && go build ./...', '/go/pkg/mod', JSON_ARRAY('go.mod'), 1),
    ('Maven', 'maven', 'Built-in Maven template', 'maven:3.9-eclipse-temurin-17', 'mvn clean package', '/root/.m2', JSON_ARRAY('pom.xml'), 1),
    ('Node.js', 'node', 'Built-in Node.js template', 'node:22-alpine', 'npm ci && npm test && npm run build', '/root/.npm', JSON_ARRAY('package.json'), 1),
    ('Docker', 'docker', 'Built-in Dockerfile template', 'gcr.io/kaniko-project/executor:v1.23.2', '', '/cache', JSON_ARRAY('Dockerfile'), 1)
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `description` = VALUES(`description`),
    `build_image` = VALUES(`build_image`),
    `build_command` = VALUES(`build_command`),
    `cache_dir` = VALUES(`cache_dir`),
    `detect_files` = VALUES(`detect_files`),
    `is_builtin` = VALUES(`is_builtin`);

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`)
VALUES ('006_cicd_build_execution.sql', '');
