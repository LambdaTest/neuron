DROP TABLE IF EXISTS test_latest_build;
DROP TABLE IF EXISTS test_suite_latest_build;
DROP TABLE IF EXISTS repo_latest_build;
ALTER TABLE test_execution DROP FOREIGN KEY fk_test_execution_task_id;
ALTER TABLE test_execution DROP COLUMN  task_id;
ALTER TABLE test DROP FOREIGN KEY fk_test_git_commits_debut_commit;
DROP TABLE IF EXISTS author_latest_build;
