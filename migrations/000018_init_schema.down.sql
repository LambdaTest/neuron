ALTER TABLE branch_commit
  DROP INDEX uq_branch_commit_commit_id_repo_id_branch_name,
  ADD CONSTRAINT uq_branch_commit_branch_name_commit_id_repo_id UNIQUE(commit_id, repo_id),
  ADD CONSTRAINT uq_branch_commit_branch_name_repo_id_branch_name UNIQUE(repo_id, branch_name);

ALTER TABLE test_suite
	MODIFY total_tests INT;