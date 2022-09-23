CREATE TABLE IF NOT EXISTS repo_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES repositories(id)
);

CREATE TABLE author_latest_build (
  id varchar(32) NOT NULL,
  author_name varchar(32) NOT NULL,
  repo_id varchar(32) NOT NULL,
  build_id varchar(32) NOT NULL,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE(author_name, repo_id),
  KEY build_id (build_id),
  FOREIGN KEY (build_id) REFERENCES build (id),
  FOREIGN KEY (author_name, repo_id) REFERENCES git_commits (author_name, repo_id)
);
