Create TABLE IF NOT EXISTS post_merge_config (
  id varchar(32) NOT NULL,
  repo_id VARCHAR(32) NOT NULL,
  branch VARCHAR(255) NOT NULL,
  is_active Boolean NOT NULL DEFAULT 1,
  patterns JSON NOT NULL,
  strategy_name enum('after_n_commits'),
  threshold varchar(32) NOT NULL,
  commit_cnt_since_last_postmerge INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  FOREIGN KEY (repo_id) REFERENCES repositories(id),
  UNIQUE INDEX idx_postmergeconfig_repo_id_branch (repo_id, branch)
);

ALTER TABLE repositories  MODIFY COLUMN run_on_each_commit BOOLEAN DEFAULT 0;
ALTER TABLE build MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error','skipped');
ALTER TABLE task MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error','skipped');
