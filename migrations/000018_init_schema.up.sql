ALTER TABLE branch_commit
  DROP FOREIGN KEY fk_branch_commit_git_commits_commit_id_repo_id,
  DROP FOREIGN KEY fk_branch_commit_branch_branch_name_repo_id,
  DROP INDEX uq_branch_commit_branch_name_commit_id_repo_id,
  DROP INDEX uq_branch_commit_branch_name_repo_id_branch_name,
  ADD CONSTRAINT uq_branch_commit_commit_id_repo_id_branch_name UNIQUE(commit_id, repo_id, branch_name);

ALTER TABLE branch_commit
  ADD CONSTRAINT fk_branch_commit_git_commits_commit_id_repo_id FOREIGN KEY(commit_id, repo_id) REFERENCES git_commits(commit_id, repo_id),
  ADD CONSTRAINT fk_branch_commit_branch_branch_name_repo_id FOREIGN KEY(repo_id, branch_name) REFERENCES branch (repo_id, name);

UPDATE test_suite 
    SET total_tests = 0 WHERE total_tests IS NULL;

ALTER TABLE test_suite
	MODIFY total_tests INT NOT NULL DEFAULT 0;
