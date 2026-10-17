-- Transactional rolling batches for pull-based host delivery.
CREATE TABLE IF NOT EXISTS `host_deployment_rollout` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `workflow_run_id` BIGINT NOT NULL,
  `task_key` VARCHAR(64) NOT NULL,
  `project_id` BIGINT NOT NULL,
  `service_id` BIGINT NOT NULL,
  `service_revision_id` BIGINT NOT NULL,
  `batch_size` INT NOT NULL DEFAULT 1,
  `interval_seconds` INT NOT NULL DEFAULT 0,
  `total_targets` INT NOT NULL,
  `total_batches` INT NOT NULL,
  `target_hash` CHAR(64) NOT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'running',
  `current_batch` INT NOT NULL DEFAULT 1,
  `error_message` VARCHAR(1024) NOT NULL DEFAULT '',
  `finished_at` TIMESTAMP NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_host_deployment_rollout` (`workflow_run_id`,`task_key`),
  KEY `idx_host_rollout_project` (`project_id`,`status`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Host deployment rolling batch controller';

ALTER TABLE `host_deployment_command`
  ADD COLUMN `batch_number` INT NOT NULL DEFAULT 1 AFTER `timeout_seconds`,
  ADD COLUMN `available_at` TIMESTAMP NULL AFTER `status`,
  ADD COLUMN `stop_reason` VARCHAR(1024) NOT NULL DEFAULT '' AFTER `error_message`,
  ADD KEY `idx_host_deploy_batch` (`workflow_run_id`,`task_key`,`batch_number`,`status`);

-- Preserve visibility and reporting for commands created before this migration.
INSERT IGNORE INTO `host_deployment_rollout`
  (`workflow_run_id`,`task_key`,`project_id`,`service_id`,`service_revision_id`,`batch_size`,`interval_seconds`,`total_targets`,`total_batches`,`target_hash`,`status`,`current_batch`,`error_message`,`finished_at`)
SELECT c.workflow_run_id,c.task_key,MIN(c.project_id),MIN(c.service_id),MIN(c.service_revision_id),COUNT(*),0,COUNT(*),1,
       SHA2(CONCAT('legacy:',c.workflow_run_id,':',c.task_key),256),
       CASE
         WHEN SUM(c.status IN ('failed','rolled_back'))>0 THEN 'failed'
         WHEN SUM(c.status='success')=COUNT(*) THEN 'success'
         ELSE 'running'
       END,
       1,
       CASE WHEN SUM(c.status IN ('failed','rolled_back'))>0 THEN 'legacy host deployment failed' ELSE '' END,
       CASE WHEN SUM(c.status IN ('failed','rolled_back','success'))=COUNT(*) THEN MAX(c.finished_at) ELSE NULL END
FROM `host_deployment_command` c
GROUP BY c.workflow_run_id,c.task_key;

INSERT IGNORE INTO `schema_migration` (`version`, `checksum`) VALUES ('024_cicd_host_rolling_batch.sql', '');
