ALTER TABLE blocklistedtests
    RENAME TO block_tests,
    ADD COLUMN repo_id varchar(32) NOT NULL,
    ADD COLUMN blocked_by varchar(255) NOT NULL,
    ADD COLUMN branch varchar(255) NOT NULL,
    CHANGE COLUMN blocklisted status enum('blocklisted', 'quarantined', 'nonflaky'),
    MODIFY COLUMN id varchar(32) NOT NULL,
    ADD CONSTRAINT fk_block_tests_repositories_repo_id FOREIGN KEY(repo_id) REFERENCES repositories(id),
    ADD UNIQUE INDEX uq_idx_block_tests_test_id_repo_id_branch (test_id, repo_id, branch),
    ADD INDEX idx_block_tests_status (status);

ALTER TABLE flaky_test_execution
    MODIFY COLUMN status enum('flaky','nonflaky','aborted','blocklisted','skipped','quarantined')  NOT NULL;

ALTER TABLE test_execution
    MODIFY COLUMN status enum('aborted','failed','passed','blocklisted','skipped', 'quarantined') NOT NULL;
