CREATE TABLE environment_service_yaml_revision (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  environment_id BIGINT NOT NULL,
  service_id BIGINT NOT NULL,
  base_revision_id BIGINT NOT NULL,
  definition MEDIUMTEXT NOT NULL,
  created_by BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_environment_service_yaml (environment_id, service_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE environment_service_override
  ADD COLUMN environment_yaml_revision_id BIGINT NOT NULL DEFAULT 0;
