TRUNCATE TABLE branch_commit;

ALTER TABLE branch_commit
  DROP INDEX branch_commit_branch_id_commit_id_uindex,
  DROP FOREIGN KEY branch_commit_branch_id_fk,
  DROP FOREIGN KEY branch_commit_git_commits_id_fk,
  DROP INDEX branch_commit_git_commits_id_fk,
  DROP COLUMN branch_id,
  MODIFY COLUMN commit_id VARCHAR(40) NOT NULL,
  ADD COLUMN branch_name VARCHAR(255) NOT NULL,
  ADD COLUMN repo_id VARCHAR(32) NOT NULL,
  ADD CONSTRAINT uq_branch_commit_branch_name_commit_id_repo_id UNIQUE(commit_id, repo_id),
  ADD CONSTRAINT uq_branch_commit_branch_name_repo_id_branch_name UNIQUE(repo_id, branch_name),
  ADD CONSTRAINT fk_branch_commit_git_commits_commit_id_repo_id FOREIGN KEY(commit_id, repo_id) REFERENCES git_commits(commit_id, repo_id),
  ADD CONSTRAINT fk_branch_commit_branch_branch_name_repo_id FOREIGN KEY(repo_id,branch_name) REFERENCES branch (repo_id,name);