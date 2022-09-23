ALTER TABLE flaky_config 
    DROP INDEX uq_idx_flaky_config_repo_id_branch,
    ADD COLUMN config_type enum('postmerge', 'premerge') NOT NULL DEFAULT 'postmerge',
    ADD UNIQUE INDEX uq_idx_flaky_config_repo_id_branch_config_type (repo_id, branch, config_type),
    ADD COLUMN auto_quarantine BOOLEAN NOT NULL DEFAULT 0;
    
ALTER TABLE task MODIFY COLUMN type enum('discover','execute', 'flaky') NOT NULL;
