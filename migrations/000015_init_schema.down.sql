ALTER TABLE credits_usage
  DROP FOREIGN KEY fk_credits_usage_tasks_task_id,
  DROP INDEX uq_credits_usage_task_id,
  DROP COLUMN task_id,
  ADD COLUMN  start_time TIMESTAMP NOT NULL,
  ADD end_time TIMESTAMP NOT NULL,
  ADD COLUMN build_id VARCHAR(32) NOT NULL,
  ADD COLUMN tier ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL;

ALTER TABLE task DROP COLUMN tier;
ALTER TABLE build ADD COLUMN tier ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small';
