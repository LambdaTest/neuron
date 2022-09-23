ALTER TABLE test_suite_execution
    MODIFY COLUMN status enum('aborted','failed','passed','blocklisted','skipped', 'quarantined') NOT NULL;
