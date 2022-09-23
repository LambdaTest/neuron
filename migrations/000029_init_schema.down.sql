ALTER TABLE test ADD COLUMN latest_build_id VARCHAR(32),
  ADD CONSTRAINT fk_test_build_latest_build_id FOREIGN KEY (latest_build_id) REFERENCES build(id);


ALTER TABLE test_suite 
  ADD COLUMN latest_build_id VARCHAR(32),
  ADD CONSTRAINT fk_test_suite_build_latest_build_id FOREIGN KEY(latest_build_id) REFERENCES build(id);