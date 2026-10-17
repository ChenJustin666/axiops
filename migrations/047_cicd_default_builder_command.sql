-- The standard builder image expects its entrypoint to be invoked explicitly
-- because the Kubernetes Job runs the command through /bin/sh -c.
UPDATE build_platform_settings
SET build_command = 'exec /usr/local/bin/entrypoint.sh'
WHERE id = 1 AND TRIM(COALESCE(build_command, '')) = '';

UPDATE build_template
SET build_command = 'exec /usr/local/bin/entrypoint.sh'
WHERE type = 'docker' AND is_builtin = 1 AND TRIM(COALESCE(build_command, '')) = '';
