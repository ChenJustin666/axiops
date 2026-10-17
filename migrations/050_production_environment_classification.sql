-- Standard production codes cannot retain the historical default false flag.
-- Deliberately do not infer classification from display names or namespaces.
UPDATE project_environment
SET is_production = 1, deploy_approval = 1
WHERE code IN ('prod', 'production') AND is_production = 0;
