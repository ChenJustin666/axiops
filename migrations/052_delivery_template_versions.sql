CREATE TABLE delivery_template_version (
 template_id BIGINT NOT NULL,
 revision INT NOT NULL,
 snapshot JSON NOT NULL,
 definition MEDIUMTEXT NOT NULL,
 content_hash CHAR(64) NOT NULL,
 created_by BIGINT NOT NULL,
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY (template_id, revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Immutable published delivery template snapshots';
