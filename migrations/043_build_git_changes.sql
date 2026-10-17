-- Persist checkout evidence independently of the short-lived build Job/Pod.
ALTER TABLE build_run
  MODIFY COLUMN git_compare_url VARCHAR(2048) NOT NULL DEFAULT '',
  ADD COLUMN git_commit_url VARCHAR(2048) NOT NULL DEFAULT '' AFTER git_compare_url,
  ADD COLUMN git_changes JSON NULL AFTER git_commit_url;
