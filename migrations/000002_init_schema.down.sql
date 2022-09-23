DROP TABLE IF EXISTS blacklistedtests;
ALTER TABLE test DROP COLUMN test_locator;
ALTER TABLE test_execution DROP COLUMN status;
ALTER TABLE test_execution DROP COLUMN blacklist_source;
DROP TABLE IF EXISTS clone_token;
DROP TABLE IF EXISTS test_coverage;
DROP TABLE IF EXISTS task;
DROP TABLE IF EXISTS test_metrics;
DROP INDEX idx_git_commits_commit_time ON git_commits;
ALTER TABLE test_suite DROP COLUMN  parent_suite_id;
