
ALTER TABLE task 
  MODIFY COLUMN status enum('initiating','running','failed','aborted','passed','completed'),
  DROP COLUMN repo_id,
  DROP FOREIGN KEY fk_task_repositories_repo_id ;

ALTER TABLE repositories RENAME COLUMN  webhook_secret TO secret;


ALTER TABLE git_commits
  DROP COLUMN id,
  DROP PRIMARY KEY,
  ADD PRIMARY KEY(commit_id),
  MODIFY COLUMN parent_commit_id VARCHAR(40) NOT NULL,
  DROP COLUMN message,
  DROP COLUMN link ,
  DROP INDEX idx_git_commits_author_name_repo_id,
  RENAME COLUMN author_name TO author,
  DROP COLUMN author_email ,
  DROP COLUMN authored_date,
  DROP COLUMN committer_name,
  DROP COLUMN committer_email ,
  CHANGE COLUMN committed_date commit_time TIMESTAMP NOT NULL,
  ADD INDEX idx_git_commits_commit_time (commit_time),
  DROP INDEX uq_git_commits_commit_id_repo_id,
  ADD CONSTRAINT UNIQUE(parent_commit_id);

ALTER TABLE git_users
  DROP COLUMN git_provider,
  DROP COLUMN user_org_id,
  MODIFY COLUMN username VARCHAR(255) NOT NULL,
  MODIFY COLUMN email VARCHAR(255),
  DROP COLUMN avatar,
  DROP COLUMN token,
  DROP COLUMN refresh_token,
  DROP COLUMN token_expiry,
  ADD COLUMN org_id VARCHAR(32) NOT NULL,
  DROP INDEX uq_git_users_username_git_provider,
  DROP INDEX user_org_id,
  DROP FOREIGN KEY fk_git_users_user_organizations,
  ADD CONSTRAINT uq_git_username UNIQUE(username, org_id),
  ADD CONSTRAINT fk_git_users_organizations_org_id FOREIGN KEY (org_id) REFERENCES organizations(id);

ALTER TABLE test_dag
  RENAME COLUMN created_at TO created,
  RENAME COLUMN updated_at TO updated;

ALTER TABLE build MODIFY COLUMN status enum('completed','aborted','failed','passed','initiating','running');

DROP TABLE IF EXISTS user_organizations;

ALTER TABLE repositories
  MODIFY COLUMN name VARCHAR(255) NOT NULL,
  MODIFY COLUMN tas_file_name VARCHAR(255) DEFAULT '.tas.yml',
  DROP COLUMN admin_id,
  DROP COLUMN git_http_url,
  DROP COLUMN git_ssh_url,
  DROP INDEX idx_repositories_active,
  DROP FOREIGN KEY fk_repositories_git_users_admin_id;

ALTER TABLE organizations
  MODIFY COLUMN  git_provider enum('github_cloud','github_self_hosted','gitlab_cloud','gitlab_self_hosted','bitbucket_cloud','bitbucket_self_hosted') NOT NULL,
  ADD COLUMN meta VARCHAR(255),
  DROP COLUMN avatar;

ALTER TABLE test_suite DROP FOREIGN KEY fk_test_suite_repositories_repo_id;
ALTER TABLE test DROP FOREIGN KEY fk_test_repositories_repo_id;

ALTER TABLE test_suite
  MODIFY COLUMN debut_commit VARCHAR(40),
  CHANGE COLUMN repo_id repo VARCHAR(255),
  RENAME COLUMN created_at TO created,
  ADD COLUMN author VARCHAR(255),
  DROP COLUMN updated_at;


ALTER TABLE test
  CHANGE COLUMN repo_id repo VARCHAR(255),
  RENAME COLUMN created_at TO created,
  ADD COLUMN author VARCHAR(255),
  DROP COLUMN updated_at;

ALTER TABLE test_status MODIFY COLUMN status enum('stable','unstable');


ALTER TABLE test_execution DROP FOREIGN KEY fk_test_execution_build_build_id;
DROP INDEX idx_test_execution_build_id_created_at ON test_execution;
ALTER TABLE test_execution DROP COLUMN build_id;


ALTER TABLE test_execution
  MODIFY COLUMN status enum('completed','aborted','failed','passed','blacklisted','skipped'),
  RENAME COLUMN created_at TO created,
  RENAME COLUMN updated_at TO updated,
  ADD INDEX (test_id),
  DROP INDEX idx_test_execution_test_id_created_at;
