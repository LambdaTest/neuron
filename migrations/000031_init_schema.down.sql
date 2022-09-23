DROP TABLE IF EXISTS post_merge_config;
ALTER TABLE repositories  MODIFY COLUMN run_on_each_commit BOOLEAN DEFAULT 1;
UPDATE  task SET status='passed' WHERE status='skipped';
UPDATE  build SET status='passed' WHERE status='skipped';
ALTER TABLE build MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error');
ALTER TABLE build MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error');
