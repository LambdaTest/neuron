ALTER TABLE test_execution MODIFY COLUMN status enum('completed','aborted','failed','passed','blocklisted','skipped');
ALTER TABLE test_suite_execution MODIFY COLUMN status enum('completed','aborted','failed','passed','blocklisted','skipped');
ALTER TABLE `build`
    DROP COLUMN `remark`,
    DROP COLUMN `tier`;

ALTER TABLE `task`
    ADD COLUMN `tier` enum('xsmall', 'small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small',
    DROP COLUMN base_commit_id,
    DROP COLUMN test_locators,
    ADD COLUMN `payload_link` varchar(512) NOT NULL,
    ADD COLUMN `remark` varchar(1024) DEFAULT NULL,
    DROP INDEX idx_task_type,
    DROP COLUMN type;

CREATE TABLE IF NOT EXISTS test_dag (
  id VARCHAR(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  dag JSON,
  commit_id VARCHAR(40) NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE(test_id),
  FOREIGN KEY (test_id) REFERENCES test(id),
  FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id)
);
