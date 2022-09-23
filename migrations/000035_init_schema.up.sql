ALTER TABLE post_merge_config
  DROP COLUMN patterns;

ALTER TABLE post_merge_config
  DROP COLUMN commit_cnt_since_last_postmerge;

Create TABLE IF NOT EXISTS commit_cnt_since_last_postmerge (
id varchar(32) NOT NULL,
repo_id VARCHAR(32) NOT NULL,
branch VARCHAR(255) NOT NULL,
commit_cnt int,
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (id),
FOREIGN KEY (repo_id) REFERENCES repositories(id),
UNIQUE INDEX uq_idx_commit_cnt_since_last_postmerge_repo_id_branch (repo_id, branch)
);
