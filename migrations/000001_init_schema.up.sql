CREATE TABLE IF NOT EXISTS test_suite (
  id VARCHAR(32) NOT NULL,
  name TEXT NOT NULL,
  file_path VARCHAR(255),
  debut_commit VARCHAR(40),
  repo VARCHAR(255) NOT NULL,
  author VARCHAR(255) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS test (
  id VARCHAR(32) NOT NULL,
  name TEXT NOT NULL,
  test_suite_id VARCHAR(32) DEFAULT NULL,
  debut_commit VARCHAR(40) NOT NULL,
  repo VARCHAR(255) NOT NULL,
  author VARCHAR(255) NOT NULL,
  status  enum('stable','unstable'),
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (test_suite_id) REFERENCES test_suite(id)
);

Create TABLE IF NOT EXISTS test_status (
  id VARCHAR(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  status  enum('stable','unstable'),
  PRIMARY KEY (id),
  FOREIGN KEY (test_id) REFERENCES test(id)
);


Create TABLE IF NOT EXISTS test_execution (
  id VARCHAR(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  commit_id VARCHAR(40) NOT NULL,
  status  enum('completed','aborted','failed','passed'),
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  start_time TIMESTAMP,
  end_time TIMESTAMP,
  duration BIGINT NOT NULL DEFAULT 0,
  stdout TEXT,
  stderr TEXT,
  PRIMARY KEY (id),
  FOREIGN KEY (test_id) REFERENCES test(id)
);

Create TABLE IF NOT EXISTS organizations (
  id VARCHAR(32) NOT NULL,
  name VARCHAR(255) NOT NULL,
  git_provider enum('github_cloud','github_self_hosted','gitlab_cloud','gitlab_self_hosted','bitbucket_cloud','bitbucket_self_hosted') NOT NULL,
  meta VARCHAR(255),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT unique_org UNIQUE(name, git_provider)
);

Create TABLE IF NOT EXISTS repositories (
  id VARCHAR(32) NOT NULL,
  org_id VARCHAR(32) NOT NULL,
  name VARCHAR(255) NOT NULL,
  link TEXT NOT NULL,
  active BOOLEAN DEFAULT 1,
  run_on_each_commit BOOLEAN DEFAULT 1,
  tas_file_name VARCHAR(255) DEFAULT '.tas.yml',
  post_merge_strategy enum ('1', ' 2'),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (org_id) REFERENCES organizations(id),
  CONSTRAINT unique_repo UNIQUE(name,org_id)
);


Create TABLE IF NOT EXISTS git_users (
   id VARCHAR(32) NOT NULL,
   org_id VARCHAR(32) NOT NULL,
   username VARCHAR(255) NOT NULL,
   email VARCHAR(255) NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (id),
   CONSTRAINT uq_git_username UNIQUE(username,org_id),
   CONSTRAINT fk_git_users_organizations_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

Create TABLE IF NOT EXISTS repo_admins (
   repo_id VARCHAR(32) NOT NULL,
   user_id VARCHAR(32) NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   FOREIGN KEY (repo_id) REFERENCES repositories(id),
   FOREIGN KEY (user_id) REFERENCES git_users(id),
   PRIMARY KEY (repo_id, user_id)
);

Create TABLE IF NOT EXISTS git_commits (
   commit_id VARCHAR(40) NOT NULL,
   parent_commit_id VARCHAR(40) NOT NULL,
   repo_id VARCHAR(32) NOT NULL,
   author VARCHAR(512) NOT NULL,
   commit_time TIMESTAMP NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (commit_id),
   FOREIGN KEY (repo_id) REFERENCES repositories(id),
   UNIQUE (parent_commit_id)
);

Create TABLE IF NOT EXISTS git_events (
   id VARCHAR(32) NOT NULL,
   repo_id VARCHAR(32) NOT NULL,
   event_name VARCHAR(255) NOT NULL,
   change_list TEXT,
   commit_id VARCHAR(40) NOT NULL,
   git_provider_handle VARCHAR(255) NOT NULL,
   event_payload JSON,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   response TEXT,
   status_code SMALLINT,
   PRIMARY KEY (id),
   FOREIGN KEY (repo_id) REFERENCES repositories(id),
   FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id)
);

Create TABLE IF NOT EXISTS build (
  id VARCHAR(32) NOT NULL,
  commit_id VARCHAR(40) NOT NULL,
  event_id VARCHAR(32) NOT NULL,
  actor VARCHAR(255),
  repo_id VARCHAR(32) NOT NULL,
  start_time TIMESTAMP DEFAULT NULL,
	end_time TIMESTAMP DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status  enum('completed','aborted','failed','passed','initiating','running'),
  PRIMARY KEY (id),
  FOREIGN KEY (repo_id) REFERENCES repositories(id),
  FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id),
  FOREIGN KEY (event_id) REFERENCES git_events(id)
);

Create TABLE IF NOT EXISTS test_transition (
   id VARCHAR(32) NOT NULL,
   test_id  VARCHAR(32) NOT NULL,
   status enum('stable','unstable') NOT NULL,
   build_id VARCHAR(32) NOT NULL,
   commit_id VARCHAR(40) NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
   PRIMARY KEY (id),
   FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id),
   FOREIGN KEY (build_id) REFERENCES build(id),
   FOREIGN KEY (test_id) REFERENCES test(id)
);


Create TABLE IF NOT EXISTS test_dag (
  id VARCHAR(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  dag JSON,
  commit_id VARCHAR(40) NOT NULL,
  updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE(test_id),
  FOREIGN KEY (test_id) REFERENCES test(id),
  FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id)
);

ALTER TABLE test_execution ADD FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id);

