START TRANSACTION;

ALTER TABLE test ADD COLUMN  submodule varchar(512) NOT NULL DEFAULT "";

ALTER TABLE test_suite  ADD COLUMN  submodule varchar(512) NOT NULL DEFAULT "";

ALTER TABLE commit_discovery  ADD COLUMN  submodule varchar(512) NOT NULL DEFAULT "";

ALTER TABLE task  ADD COLUMN  submodule  varchar(512) NOT NULL  DEFAULT  "";

ALTER TABLE  commit_discovery DROP FOREIGN KEY fk_commit_discovery_commit_id_repo_id;

ALTER TABLE  commit_discovery DROP INDEX uq_commit_discovery_commit_id_repo_id;



ALTER TABLE  commit_discovery ADD CONSTRAINT uq_commit_discovery_commit_id_repo_id_submodule UNIQUE(commit_id, repo_id, submodule);


ALTER TABLE  commit_discovery ADD CONSTRAINT fk_commit_discovery_commit_id_repo_id FOREIGN KEY(commit_id,repo_id) REFERENCES git_commits(commit_id, repo_id);
COMMIT;