CREATE TABLE delivery_template (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 kind VARCHAR(24) NOT NULL,
 name VARCHAR(128) NOT NULL,
 description VARCHAR(512) NOT NULL DEFAULT '',
 files JSON NOT NULL,
 variables JSON NOT NULL,
 revision INT NOT NULL DEFAULT 1,
 builtin_key VARCHAR(64) NULL,
 status VARCHAR(16) NOT NULL DEFAULT 'active',
 created_by BIGINT NOT NULL,
 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 UNIQUE KEY uk_delivery_template_builtin (builtin_key),
 KEY idx_delivery_template_kind (kind,status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
