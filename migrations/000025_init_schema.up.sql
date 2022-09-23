
CREATE TABLE IF NOT EXISTS commit_discovery(
    id VARCHAR(32) NOT NULL,
    commit_id VARCHAR(40)  NOT NULL,
    repo_id VARCHAR(32) NOT NULL,
    test_ids JSON,
    suite_ids JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_commit_discovery_commit_id_repo_id UNIQUE(commit_id, repo_id),
    CONSTRAINT fk_commit_discovery_commit_id_repo_id FOREIGN KEY(commit_id,repo_id) REFERENCES git_commits(commit_id, repo_id)
);