ALTER TABLE branch_commit ADD COLUMN branch_id VARCHAR(32) NULL;

UPDATE branch_commit bc JOIN branch b ON b.repo_id=bc.repo_id AND b.name=bc.branch_name SET bc.branch_id=b.id;


ALTER TABLE branch_commit
  DROP FOREIGN KEY fk_branch_commit_git_commits_commit_id_repo_id,
  DROP FOREIGN KEY fk_branch_commit_branch_branch_name_repo_id,
  DROP COLUMN branch_name,
  DROP COLUMN repo_id,
  DROP INDEX uq_branch_commit_branch_name_commit_id_repo_id,
  DROP INDEX uq_branch_commit_branch_name_repo_id_branch_name;


ALTER TABLE branch_commit
  ADD CONSTRAINT branch_commit_branch_id_commit_id_uindex UNIQUE (branch_id, commit_id),
  ADD CONSTRAINT branch_commit_branch_id_fk FOREIGN KEY (branch_id) REFERENCES branch (id),
  ADD CONSTRAINT branch_commit_git_commits_id_fk FOREIGN KEY (commit_id) REFERENCES git_commits (id);