CREATE TABLE IF NOT EXISTS `license_info` (
  `id` varchar(32) PRIMARY KEY,
  `type` ENUM ('Basic', 'Teams', 'Enterprise') NOT NULL DEFAULT 'Basic',
  `org_id` varchar(32) NOT NULL,
  `credits_bought` bigint NOT NULL DEFAULT 0,
  `credits_balance` bigint NOT NULL DEFAULT 0,
  `credits_expiry` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `concurrency` integer NOT NULL DEFAULT 2,
  `tier` ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`),
  CONSTRAINT unique_org UNIQUE(`org_id`)
);

CREATE TABLE IF NOT EXISTS `credits_usage` (
  `id` varchar(32) PRIMARY KEY,
  `org_id` varchar(32),
  `user` varchar(32),
  `build_id` varchar(32),
  `tier` ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL,
  `start_time` timestamp NOT NULL,
  `end_time` timestamp NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `user_idx` (`user`),
  INDEX `created_at_idx` (`created_at`),
  FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`)
);

CREATE TABLE IF NOT EXISTS `job_queue` (
  `id` varchar(32) PRIMARY KEY,
  `status` ENUM ("ready", "processing") NOT NULL,
  `org_id` varchar(32) NOT NULL,
  `task_id` varchar(32) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `created_at_idx` (`created_at`),
  INDEX `status_idx` (`status`),
  FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`),
  FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
);

ALTER TABLE `build` ADD COLUMN `tier` ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small';
