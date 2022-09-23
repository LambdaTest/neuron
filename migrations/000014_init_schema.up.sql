ALTER TABLE test ADD COLUMN latest_build_id VARCHAR(32),
  ADD CONSTRAINT fk_test_build_latest_build_id FOREIGN KEY (latest_build_id) REFERENCES build(id);

UPDATE test t JOIN test_latest_build tl ON t.id=tl.id SET t.latest_build_id=tl.build_id;

ALTER TABLE test_suite 
  ADD COLUMN latest_build_id VARCHAR(32),
  ADD CONSTRAINT fk_test_suite_build_latest_build_id FOREIGN KEY(latest_build_id) REFERENCES build(id);

UPDATE test_suite ts JOIN test_suite_latest_build tl ON ts.id=tl.id SET ts.latest_build_id=tl.build_id;

DROP TABLE IF EXISTS test_latest_build;
DROP TABLE IF EXISTS test_suite_latest_build;

CREATE TABLE IF NOT EXISTS test_branch(
  id VARCHAR(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  repo_id VARCHAR(32) NOT NULL,
  branch_name VARCHAR(255) NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uq_test_id_repo_id_branch_name UNIQUE(test_id, repo_id, branch_name),
  CONSTRAINT fk_test_branch_branch_test_id FOREIGN KEY (test_id) REFERENCES test(id),
  CONSTRAINT fk_test_branch_branch_repo_id_branch_name FOREIGN KEY (repo_id,branch_name) REFERENCES branch(repo_id,name)
);

CREATE TABLE IF NOT EXISTS test_suite_branch(
  id VARCHAR(32) NOT NULL,
  test_suite_id VARCHAR(32) NOT NULL,
  repo_id VARCHAR(32) NOT NULL,
  branch_name VARCHAR(255) NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT uq_test_suite_id_repo_id_branch_name UNIQUE(test_suite_id, repo_id, branch_name),
  CONSTRAINT fk_test_suite_branch_branch_test_suite_id FOREIGN KEY (test_suite_id) REFERENCES test_suite(id),
  CONSTRAINT fk_test_suite_branch_branch_repo_id_branch_name FOREIGN KEY (repo_id,branch_name) REFERENCES branch(repo_id,name)
);