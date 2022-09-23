UPDATE test_execution SET status = 'aborted' WHERE status = 'blocklisted';

UPDATE test_suite_execution SET status = 'aborted' WHERE status = 'blocklisted';

ALTER TABLE test_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blacklisted','skipped') NOT NULL,
  RENAME COLUMN blocklist_source TO blacklist_source;

ALTER TABLE test_suite_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blacklisted','skipped') NOT NULL,
  RENAME COLUMN blocklist_source TO blacklist_source;

ALTER TABLE blocklistedtests
    RENAME COLUMN blocklisted TO blacklisted;

RENAME TABLE blocklistedtests TO blacklistedtests;

UPDATE test_execution SET status = 'blacklisted' WHERE status = 'aborted';

UPDATE test_suite_execution SET status = 'blacklisted' WHERE status = 'aborted';
