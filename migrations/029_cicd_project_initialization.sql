-- Persist the project onboarding state used by the four-step CI/CD wizard.
ALTER TABLE `project`
  ADD COLUMN `initialization_status` VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'pending, in_progress or completed',
  ADD COLUMN `initialization_step` INT NOT NULL DEFAULT 0 COMMENT 'wizard step from 0 to 4',
  ADD COLUMN `initialized_at` DATETIME NULL COMMENT 'time the project completed onboarding';

-- Existing projects already have production data and must keep opening in the workspace.
UPDATE `project`
SET `initialization_status` = 'completed',
    `initialization_step` = 4,
    `initialized_at` = COALESCE(`initialized_at`, `updated_at`)
WHERE `initialization_status` = 'pending';
