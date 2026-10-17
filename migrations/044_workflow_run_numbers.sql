-- A visible run number belongs to a workflow Job; the primary key stays global.
ALTER TABLE workflow ADD COLUMN run_sequence BIGINT NOT NULL DEFAULT 0;
ALTER TABLE workflow_run ADD COLUMN run_number BIGINT NULL;
UPDATE workflow_run r JOIN (
  SELECT current_run.id, COUNT(previous_run.id) AS local_number
  FROM workflow_run current_run JOIN workflow_run previous_run
    ON previous_run.workflow_id=current_run.workflow_id AND previous_run.id<=current_run.id
  GROUP BY current_run.id
) numbered ON numbered.id=r.id SET r.run_number=numbered.local_number;
UPDATE workflow w LEFT JOIN (
  SELECT workflow_id,MAX(run_number) AS last_number FROM workflow_run GROUP BY workflow_id
) numbers ON numbers.workflow_id=w.id SET w.run_sequence=COALESCE(numbers.last_number,0);
ALTER TABLE workflow_run ADD UNIQUE KEY uk_workflow_run_number (workflow_id,run_number);
