ALTER TABLE build_config
  ADD COLUMN dockerfile_source VARCHAR(20) NOT NULL DEFAULT 'repository',
  ADD COLUMN dockerfile_content MEDIUMTEXT NULL;
