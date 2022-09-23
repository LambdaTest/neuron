ALTER TABLE block_tests
    DROP CONSTRAINT block_tests_ibfk_1,
    DROP CONSTRAINT fk_block_tests_repositories_repo_id,
    DROP INDEX uq_idx_block_tests_test_id_repo_id_branch,
    DROP INDEX idx_block_tests_type,
    CHANGE COLUMN type blocklisted tinyint(1) DEFAULT '1',
    MODIFY COLUMN id int AUTO_INCREMENT NOT NULL,
    DROP COLUMN blocked_by,
    DROP COLUMN repo_id,
    DROP COLUMN branch,
    RENAME TO blocklistedtests,
    ADD CONSTRAINT fk_blocklistedtests FOREIGN KEY(test_id) REFERENCES test(id);

ALTER TABLE flaky_test_execution
    MODIFY COLUMN status enum('flaky','nonflaky','aborted','blocklisted','skipped','quarantined') NOT NULL;

ALTER TABLE test_execution
    MODIFY COLUMN status enum('aborted','failed','passed','blocklisted','skipped', 'quarantined') NOT NULL;
