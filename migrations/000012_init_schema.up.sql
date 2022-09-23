CREATE TABLE IF NOT EXISTS branch
(
    id         VARCHAR(32) NOT NULL PRIMARY KEY,
    repo_id    VARCHAR(32) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT branch_repo_id_name_uindex UNIQUE (repo_id, name),
    CONSTRAINT branch_repositories_id_fk FOREIGN KEY (repo_id) REFERENCES
        repositories (id)
);

CREATE TABLE IF NOT EXISTS branch_commit
(
    id         VARCHAR(32) NOT NULL PRIMARY KEY,
    commit_id  VARCHAR(32) NULL,
    branch_id  VARCHAR(32) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT branch_commit_branch_id_commit_id_uindex UNIQUE (branch_id,
                                                                commit_id),
    CONSTRAINT branch_commit_branch_id_fk FOREIGN KEY (branch_id) REFERENCES
        branch (id),
    CONSTRAINT branch_commit_git_commits_id_fk FOREIGN KEY (commit_id)
        REFERENCES git_commits (id)
);