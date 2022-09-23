ALTER TABLE build 
ADD COLUMN `time_all_tests_ms` BIGINT NOT NULL DEFAULT '0' AFTER `status`,
ADD COLUMN `time_impacted_tests_ms` BIGINT NOT NULL DEFAULT '0' AFTER `time_all_tests_ms`;
