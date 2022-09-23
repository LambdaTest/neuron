UPDATE test_execution SET status = 'aborted' WHERE status = 'blacklisted';

UPDATE test_suite_execution SET status = 'aborted' WHERE status = 'blacklisted';

ALTER TABLE test_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blocklisted','skipped') NOT NULL,
  RENAME COLUMN blacklist_source TO blocklist_source;

ALTER TABLE test_suite_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blocklisted','skipped') NOT NULL,
  RENAME COLUMN blacklist_source TO blocklist_source;

ALTER TABLE blacklistedtests
    RENAME COLUMN blacklisted TO blocklisted;

RENAME TABLE blacklistedtests TO blocklistedtests;

UPDATE test_execution SET status = 'blocklisted' WHERE status = 'aborted';

UPDATE test_suite_execution SET status = 'blocklisted' WHERE status = 'aborted';
