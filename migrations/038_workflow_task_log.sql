-- Bounded deployment execution evidence, retained with the workflow task.
ALTER TABLE workflow_task_run ADD COLUMN log_text MEDIUMTEXT NULL;
