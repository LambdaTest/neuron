CREATE TABLE IF NOT EXISTS `user_info` (
  `id` varchar(32) NOT NULL PRIMARY KEY,
  `user_id` varchar(32) NOT NULL,
  `user_description` varchar(32),
  `experience` ENUM ('L1', 'L2', 'L3', 'L4') NOT NULL DEFAULT 'L1',
  `team_size` ENUM ('small', 'medium', 'large', 'xlarge') NOT NULL DEFAULT 'small',
  `org_id` varchar(32) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uq_user_info_org_id_user_id UNIQUE(org_id, user_id),
  FOREIGN KEY (`user_id`) REFERENCES `git_users` (`id`),
  FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`)
);