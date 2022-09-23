
ALTER TABLE `build`
    ADD COLUMN `tier` enum('internal', 'xsmall', 'small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small',
    ADD COLUMN `remark` varchar(1000) DEFAULT NULL;

ALTER TABLE `task`
    DROP COLUMN `tier`,
    DROP COLUMN `payload_link`,
    DROP COLUMN `remark`,
    ADD COLUMN base_commit_id VARCHAR(40),
    ADD COLUMN test_locators TEXT,
    ADD COLUMN type enum ('discover', 'execute') NOT NULL,
    ADD INDEX idx_task_type (type);

ALTER TABLE test_execution MODIFY COLUMN status enum('aborted','failed','passed','blocklisted','skipped');
ALTER TABLE test_suite_execution MODIFY COLUMN status enum('aborted','failed','passed','blocklisted','skipped');

DROP TABLE IF EXISTS test_dag;
