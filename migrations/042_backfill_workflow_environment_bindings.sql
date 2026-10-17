-- Repair workflows created before target_environment_id was populated.
-- Only generated, unambiguous deployment workflows are repaired.
UPDATE workflow w
JOIN project p ON p.id = w.project_id
JOIN project_environment e
  ON e.project_id = w.project_id
 AND e.status = 'active'
 AND w.code = CONCAT(p.code, '-', e.code)
SET w.target_environment_id = e.id,
    w.updated_at = CURRENT_TIMESTAMP
WHERE w.target_environment_id = 0
  AND EXISTS (
    SELECT 1 FROM workflow_node n
    WHERE n.workflow_id = w.id
      AND n.node_type IN ('yaml_deploy', 'helm_deploy', 'host_deploy')
  )
  AND (
    SELECT COUNT(*) FROM project_environment e2
    WHERE e2.project_id = w.project_id
      AND e2.status = 'active'
      AND e2.code = e.code
  ) = 1;
