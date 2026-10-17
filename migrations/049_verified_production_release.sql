CREATE TABLE IF NOT EXISTS production_release_policy (
  environment_id BIGINT NOT NULL PRIMARY KEY,
  project_id BIGINT NOT NULL,
  source_environment_id BIGINT NOT NULL,
  revision BIGINT NOT NULL DEFAULT 1,
  updated_by BIGINT NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_production_policy_project(project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS deployment_verification (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT NOT NULL,
  environment_id BIGINT NOT NULL,
  service_id BIGINT NOT NULL,
  snapshot_id BIGINT NOT NULL,
  build_run_id BIGINT NOT NULL,
  image_ref VARCHAR(1024) NOT NULL,
  manifest_digest CHAR(64) NOT NULL,
  status VARCHAR(24) NOT NULL DEFAULT 'passed',
  method VARCHAR(24) NOT NULL DEFAULT 'manual',
  note VARCHAR(1024) NOT NULL DEFAULT '',
  verified_by BIGINT NOT NULL,
  verified_username VARCHAR(128) NOT NULL DEFAULT '',
  revision BIGINT NOT NULL DEFAULT 1,
  verified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_verification_snapshot(snapshot_id),
  KEY idx_verification_candidates(project_id,environment_id,service_id,status,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS deployment_verification_event (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT NOT NULL,
  verification_id BIGINT NOT NULL,
  action VARCHAR(24) NOT NULL,
  actor_id BIGINT NOT NULL,
  actor_username VARCHAR(128) NOT NULL DEFAULT '',
  note VARCHAR(1024) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_verification_event(project_id,verification_id,id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
