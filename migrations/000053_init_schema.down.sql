ALTER TABLE flaky_config 
    DROP INDEX uq_idx_flaky_config_repo_id_branch_config_type,
    DROP COLUMN config_type,
    DROP COLUMN auto_quarantine,
    ADD  UNIQUE INDEX uq_idx_flaky_config_repo_id_branch (repo_id, branch);

ALTER TABLE task MODIFY COLUMN type enum('discover','execute') NOT NULL;
