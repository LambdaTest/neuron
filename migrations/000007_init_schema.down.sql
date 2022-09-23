ALTER TABLE author_latest_build DROP COLUMN created_at;
ALTER TABLE repo_latest_build DROP COLUMN created_at;
ALTER TABLE test_latest_build DROP COLUMN created_at;
ALTER TABLE test_suite_latest_build DROP COLUMN created_at;
ALTER TABLE repositories DROP COLUMN mask;
ALTER TABLE git_users DROP COLUMN mask;
ALTER TABLE repositories DROP COLUMN custom;