ALTER TABLE build
    ADD build_num int NOT NULL;

WITH buildno AS (SELECT id, repo_id, created_at, updated_at, RANK() OVER (PARTITION BY repo_id ORDER BY created_at, updated_at, id) as sno from build)
UPDATE build b JOIN buildno bno ON (bno.id = b.id) set b.build_num=bno.sno;

ALTER TABLE build   
    ADD CONSTRAINT uq_build_repo_id_build_num UNIQUE (repo_id, build_num);
