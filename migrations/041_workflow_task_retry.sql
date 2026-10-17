-- A retry is a new execution, distinct from reclaiming a crashed worker lease.
ALTER TABLE workflow_task_run
  ADD COLUMN retry_count INT NOT NULL DEFAULT 0,
  ADD COLUMN next_retry_at DATETIME NULL,
  ADD COLUMN retry_history JSON NULL;
