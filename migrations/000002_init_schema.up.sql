CREATE TABLE IF NOT EXISTS  blacklistedtests(
    id INT AUTO_INCREMENT,
    test_id VARCHAR(32) NOT NULL,
    blacklisted BOOLEAN DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (test_id) REFERENCES test(id)
);


ALTER TABLE test ADD COLUMN  test_locator TEXT NOT NULL;
ALTER TABLE test_execution MODIFY COLUMN status enum('completed','aborted','failed','passed','blacklisted','skipped');
ALTER TABLE test_execution ADD COLUMN blacklist_source VARCHAR(255);
ALTER TABLE repositories ADD COLUMN  secret VARCHAR(32) NOT NULL;

CREATE TABLE IF NOT EXISTS clone_token(
    id VARCHAR(32) NOT NULL,
    repo_id VARCHAR(32) NOT NULL,
    token VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS test_coverage(
    id VARCHAR(32) NOT NULL,
    commit_id VARCHAR(40)  NOT NULL,
    repo_id VARCHAR(32) NOT NULL,
    blob_link TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total_coverage JSON,
    PRIMARY KEY (id),
    UNIQUE(commit_id),
    FOREIGN KEY (repo_id) REFERENCES repositories(id)
);

CREATE TABLE IF NOT EXISTS task(
    id VARCHAR(32) NOT NULL,
    commit_id VARCHAR(40)  NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    payload_link VARCHAR(512) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status enum('initiating','running','failed','aborted','passed','completed'),
    start_time TIMESTAMP  DEFAULT NULL,
    end_time TIMESTAMP DEFAULT NULL,
    remark VARCHAR(255),
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id)
);


CREATE TABLE IF NOT EXISTS test_metrics(
    id VARCHAR(32) NOT NULL,
    test_execution_id VARCHAR(32) NOT NULL,
    cpu FLOAT NOT NULL DEFAULT 0.0,
    record_time TIMESTAMP(6) NOT NULL,
    memory BIGINT NOT NULL DEFAULT 0,
    storage BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    FOREIGN KEY (test_execution_id) REFERENCES test_execution(id)
);

CREATE INDEX idx_git_commits_commit_time ON git_commits(commit_time);
ALTER TABLE test_suite ADD COLUMN  parent_suite_id VARCHAR(32) DEFAULT NULL;