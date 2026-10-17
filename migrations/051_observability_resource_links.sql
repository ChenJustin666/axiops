-- Explicit links between legacy UUIDs and platform IDs. No implicit ID/name matching.
CREATE TABLE IF NOT EXISTS legacy_observability_link (
  kind VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  external_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  target_id BIGINT NOT NULL,
  status TINYINT NOT NULL DEFAULT 1,
  created_by BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (kind, external_id),
  UNIQUE KEY uk_observability_target (kind, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='verified legacy observability resource links';

INSERT IGNORE INTO schema_migration (version, checksum)
VALUES ('051_observability_resource_links.sql', '');
