CREATE TABLE IF NOT EXISTS test_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES test(id)
);

CREATE TABLE IF NOT EXISTS test_suite_latest_build(
    id VARCHAR(32) NOT NULL,
    build_id VARCHAR(32) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (build_id) REFERENCES build(id),
    FOREIGN KEY (id) REFERENCES test_suite(id)
);

ALTER TABLE test
  DROP FOREIGN KEY fk_test_build_latest_build_id,
  DROP COLUMN latest_build_id;

ALTER TABLE test_suite
  DROP FOREIGN KEY fk_test_suite_build_latest_build_id, 
  DROP COLUMN latest_build_id;

DROP TABLE IF EXISTS test_branch;
DROP TABLE IF EXISTS test_suite_branch;