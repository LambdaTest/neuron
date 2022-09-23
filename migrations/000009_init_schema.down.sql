ALTER TABLE test_coverage
  DROP FOREIGN KEY fk_test_coverage_commit_id_repo_id,
  DROP INDEX uq_test_coverage_commit_id_repo_id,
  ADD CONSTRAINT UNIQUE(commit_id),
  ADD CONSTRAINT FOREIGN KEY(repo_id) REFERENCES repositories(id);