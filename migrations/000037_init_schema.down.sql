ALTER TABLE build 
    DROP CONSTRAINT uq_build_repo_id_build_num;

ALTER TABLE build
    DROP COLUMN build_num;
