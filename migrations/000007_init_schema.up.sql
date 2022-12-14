ALTER TABLE author_latest_build ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE repo_latest_build ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE test_latest_build ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE test_suite_latest_build ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE repositories ADD COLUMN mask VARCHAR(16) NOT NULL DEFAULT 'mask';
ALTER TABLE git_users ADD COLUMN mask VARCHAR(16) NOT NULL DEFAULT 'mask';
ALTER TABLE repositories ADD COLUMN custom BOOLEAN NOT NULL DEFAULT 0;