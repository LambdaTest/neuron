CREATE TABLE IF NOT EXISTS test_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES test(id)
);

CREATE TABLE IF NOT EXISTS test_suite_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES test_suite(id)
);

CREATE TABLE IF NOT EXISTS repo_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES repositories(id)
);

ALTER TABLE test_execution ADD COLUMN task_id VARCHAR(32) NOT NULL;
ALTER TABLE test_execution ADD CONSTRAINT fk_test_execution_task_id FOREIGN KEY (task_id) REFERENCES task(id);
ALTER TABLE test ADD CONSTRAINT fk_test_git_commits_debut_commit  FOREIGN KEY (debut_commit) REFERENCES git_commits(commit_id);


CREATE TABLE IF NOT EXISTS author_latest_build(
    id VARCHAR(32) NOT NULL,
    author_name VARCHAR(32) NOT NULL,
    repo_id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(author_name, repo_id),
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (author_name, repo_id) REFERENCES git_commits(author_name, repo_id)
);