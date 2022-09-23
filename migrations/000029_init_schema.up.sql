ALTER TABLE test
  DROP FOREIGN KEY fk_test_build_latest_build_id,
  DROP COLUMN latest_build_id;

ALTER TABLE test_suite
  DROP FOREIGN KEY fk_test_suite_build_latest_build_id, 
  DROP COLUMN latest_build_id;