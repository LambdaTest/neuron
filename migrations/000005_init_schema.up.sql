DROP TABLE IF EXISTS clone_token;
ALTER TABLE git_users DROP COLUMN token;
ALTER TABLE git_users DROP COLUMN refresh_token;
ALTER TABLE git_users DROP COLUMN token_expiry;
