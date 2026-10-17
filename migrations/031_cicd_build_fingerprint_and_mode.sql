-- Stable build identity and explicit output mode. This lets environments reuse
-- one immutable artifact without rebuilding the same commit.
ALTER TABLE `build_config`
  ADD COLUMN `build_mode` VARCHAR(24) NOT NULL DEFAULT 'dockerfile' AFTER `template_type`;

ALTER TABLE `build_run`
  ADD COLUMN `input_fingerprint` CHAR(64) NOT NULL DEFAULT '' AFTER `config_snapshot`,
  ADD KEY `idx_build_run_fingerprint` (`service_id`, `input_fingerprint`, `status`);
