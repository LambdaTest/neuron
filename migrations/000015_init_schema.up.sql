TRUNCATE TABLE credits_usage;
ALTER TABLE credits_usage
  DROP COLUMN build_id,
  DROP COLUMN start_time,
  DROP COLUMN end_time,
  DROP COLUMN tier,
  ADD COLUMN task_id VARCHAR(32) NOT NULL,
  ADD CONSTRAINT fk_credits_usage_tasks_task_id FOREIGN KEY (task_id) REFERENCES task(id),
  ADD CONSTRAINT uq_credits_usage_task_id UNIQUE(task_id);

ALTER TABLE task ADD COLUMN tier ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small';
ALTER TABLE build DROP COLUMN tier;
