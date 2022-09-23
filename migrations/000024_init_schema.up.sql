CREATE TABLE `build_coverage_status` (
  `build_id` varchar(32) NOT NULL,
  `base_commit_id` varchar(40) NOT NULL,
  `coverage_available` BOOLEAN NOT NULL DEFAULT 0,
  `coverage_recoverable` BOOLEAN NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`build_id`),
  CONSTRAINT `fk_build_coverage_status_build_id` FOREIGN KEY (`build_id`) REFERENCES `build` (`id`)
);
