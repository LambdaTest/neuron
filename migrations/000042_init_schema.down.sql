CREATE TABLE `parsing_queue` (
  `id` varchar(32) NOT NULL,
  `status` enum('ready','processing','completed','failed') NOT NULL,
  `org_id` varchar(32) NOT NULL,
  `build_id` varchar(32) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `parsing_queue_ibfk_2` (`build_id`),
  KEY `parsing_queue_ibfk_1` (`org_id`),
  KEY `idx_parsing_queue_status_created_at` (`status`,`created_at`),
  CONSTRAINT `parsing_queue_build_id_fk` FOREIGN KEY (`build_id`) REFERENCES `build` (`id`),
  CONSTRAINT `parsing_queue_ibfk_1` FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`)
);

CREATE TABLE `task_queue` (
  `id` varchar(32) NOT NULL,
  `status` enum('ready','processing','completed','failed') NOT NULL,
  `org_id` varchar(32) NOT NULL,
  `task_id` varchar(32) NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_task_queue_task_id` (`task_id`),
  KEY `idx_task_queue_status_created_at` (`status`,`created_at`),
  KEY `fk_task_queue_organizations_org_id` (`org_id`),
  CONSTRAINT `fk_task_queue_organizations_org_id` FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`),
  CONSTRAINT `fk_task_queue_task_task_id` FOREIGN KEY (`task_id`) REFERENCES `task` (`id`)
);

INSERT
	INTO
	parsing_queue (
		id,
		status,
		org_id,
		build_id,
		created_at,
		updated_at
	)
SELECT
	id,
	status,
	org_id,
	ref_entity_id,
	created_at,
	updated_at
FROM
	event_queue
WHERE
    event_type = 'parsing';

INSERT
	INTO
	task_queue (
		id,
		status,
		org_id,
		task_id,
		created_at,
		updated_at
	)
SELECT
	id,
	status,
	org_id,
	ref_entity_id,
	created_at,
	updated_at
FROM
	event_queue
WHERE
    event_type = 'task';

DROP TABLE `event_queue`;
