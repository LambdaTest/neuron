ALTER TABLE test_suite
  MODIFY COLUMN debut_commit VARCHAR(40) NOT NULL,
  CHANGE COLUMN repo repo_id VARCHAR(32),
  RENAME COLUMN created TO created_at,
  DROP COLUMN author,
  ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;


ALTER TABLE test
  CHANGE COLUMN repo repo_id VARCHAR(32),
  RENAME COLUMN created TO created_at,
  DROP COLUMN author,
  DROP COLUMN status,
  ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE test_status MODIFY COLUMN status enum('stable','unstable') NOT NULL DEFAULT 'stable';

ALTER TABLE test_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blacklisted','skipped') NOT NULL,
  RENAME COLUMN created TO created_at,
  RENAME COLUMN updated TO updated_at,
  ADD INDEX idx_test_execution_test_id_created_at (test_id, created_at);

ALTER TABLE organizations
  MODIFY COLUMN git_provider enum('github','github_self_hosted','gitlab','gitlab_self_hosted','bitbucket','bitbucket_self_hosted') NOT NULL,
  DROP COLUMN meta,
  ADD COLUMN avatar VARCHAR(1024) NOT NULL;

ALTER TABLE repositories
  MODIFY COLUMN name VARCHAR(512) NOT NULL,
  MODIFY COLUMN tas_file_name VARCHAR(255) NOT NULL DEFAULT '.tas.yml',
  ADD COLUMN admin_id VARCHAR(32) NOT NULL,
  ADD COLUMN git_http_url TEXT,
  ADD COLUMN git_ssh_url TEXT,
  ADD INDEX idx_repositories_active (active),
  ADD CONSTRAINT fk_repositories_git_users_admin_id FOREIGN KEY (admin_id) REFERENCES git_users(id);


Create TABLE IF NOT EXISTS user_organizations (
   id VARCHAR(32) NOT NULL,
   user_id VARCHAR(32) NOT NULL,
   org_id VARCHAR(32) NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (id),
   UNIQUE  uq_user_organizations_user_id_org_id (user_id, org_id),
   CONSTRAINT fk_user_organizations_organizations_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

ALTER TABLE build MODIFY COLUMN status enum('completed','aborted','failed','passed','initiating','running') NOT NULL;

ALTER TABLE test_dag
  RENAME COLUMN created TO created_at,
  RENAME COLUMN updated TO updated_at;


ALTER TABLE git_users
  ADD COLUMN git_provider enum('github','github_self_hosted','gitlab','gitlab_self_hosted','bitbucket','bitbucket_self_hosted') NOT NULL,
  ADD COLUMN user_org_id VARCHAR(32) NOT NULL,
  MODIFY COLUMN username VARCHAR(512) NOT NULL,
  MODIFY COLUMN email VARCHAR(512),
  ADD COLUMN avatar VARCHAR(1024) NOT NULL,
  ADD COLUMN token VARCHAR(255) NOT NULL,
  ADD COLUMN refresh_token VARCHAR(255),
  ADD COLUMN token_expiry TIMESTAMP,
  DROP INDEX uq_git_username,
  ADD CONSTRAINT uq_git_users_username_git_provider UNIQUE(username, git_provider),
  ADD CONSTRAINT UNIQUE(user_org_id),
  ADD CONSTRAINT fk_git_users_user_organizations FOREIGN KEY (user_org_id) REFERENCES user_organizations(user_id),
  DROP FOREIGN KEY fk_git_users_organizations_org_id,
  DROP COLUMN org_id;

ALTER TABLE git_commits
  ADD COLUMN id VARCHAR(32) NOT NULL,
  DROP PRIMARY KEY,
  ADD PRIMARY KEY(id),
  MODIFY COLUMN parent_commit_id VARCHAR(40),
  ADD COLUMN message TEXT,
  ADD COLUMN link TEXT NOT NULL,
  RENAME COLUMN author TO author_name,
  ADD COLUMN author_email VARCHAR(512) NOT NULL,
  ADD COLUMN authored_date TIMESTAMP NOT NULL,
  ADD COLUMN committer_name VARCHAR(512),
  ADD COLUMN committer_email VARCHAR(512),
  DROP INDEX idx_git_commits_commit_time,
  CHANGE COLUMN commit_time committed_date TIMESTAMP,
  ADD CONSTRAINT uq_git_commits_commit_id_repo_id UNIQUE(commit_id, repo_id),
  ADD INDEX idx_git_commits_author_name_repo_id (author_name, repo_id),
  DROP INDEX parent_commit_id;

ALTER TABLE repositories RENAME COLUMN secret TO webhook_secret;

ALTER TABLE task 
  MODIFY COLUMN status enum('initiating','running','failed','aborted','passed','completed') NOT NULL,
  ADD COLUMN repo_id VARCHAR(32) NOT NULL,
  ADD CONSTRAINT fk_task_repositories_repo_id  FOREIGN KEY (repo_id) REFERENCES repositories(id);

ALTER TABLE test_suite ADD CONSTRAINT fk_test_suite_repositories_repo_id FOREIGN KEY (repo_id) REFERENCES repositories(id);
ALTER TABLE test ADD CONSTRAINT fk_test_repositories_repo_id FOREIGN KEY (repo_id) REFERENCES repositories(id);
ALTER TABLE test_execution ADD COLUMN  build_id VARCHAR(32) NOT NULL;
ALTER TABLE test_execution ADD CONSTRAINT fk_test_execution_build_build_id FOREIGN KEY (build_id) REFERENCES build(id);
CREATE INDEX idx_test_execution_build_id_created_at ON test_execution(build_id, created_at);
