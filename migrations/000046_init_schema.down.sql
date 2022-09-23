ALTER TABLE task 
    ADD COLUMN commit_id VARCHAR(40) NOT NULL,
    ADD COLUMN base_commit_id VARCHAR(40),
    ADD FOREIGN KEY (commit_id) REFERENCES git_commits(commit_id);
