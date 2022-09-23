START TRANSACTION;

ALTER TABLE test DROP COLUMN  submodule;

ALTER TABLE test_suite  DROP COLUMN  submodule;

DELETE FROM commit_discovery WHERE submodule != '';

ALTER TABLE commit_discovery  DROP COLUMN  submodule;

ALTER TABLE task  DROP  COLUMN  submodule;



ALTER TABLE  commit_discovery DROP FOREIGN KEY  fk_commit_discovery_commit_id_repo_id;

ALTER TABLE  commit_discovery DROP INDEX  uq_commit_discovery_commit_id_repo_id_submodule;


ALTER TABLE  commit_discovery ADD CONSTRAINT uq_commit_discovery_commit_id_repo_id UNIQUE(commit_id, repo_id);

ALTER TABLE  commit_discovery ADD CONSTRAINT fk_commit_discovery_commit_id_repo_id FOREIGN KEY(commit_id,repo_id) REFERENCES git_commits(commit_id, repo_id);

COMMIT;