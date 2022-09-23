ALTER TABLE test_coverage 
  DROP INDEX commit_id,
  DROP FOREIGN KEY test_coverage_ibfk_1,
  DROP INDEX repo_id,
  ADD CONSTRAINT uq_test_coverage_commit_id_repo_id UNIQUE(commit_id, repo_id),
  ADD CONSTRAINT fk_test_coverage_commit_id_repo_id FOREIGN KEY(commit_id,repo_id) REFERENCES git_commits(commit_id,repo_id);