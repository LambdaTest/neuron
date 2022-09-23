CREATE TABLE `event_queue` (
  `id` varchar(32) NOT NULL,
  `status` enum('ready','processing','completed','failed') NOT NULL,
  `org_id` varchar(32) NOT NULL,
  `event_type` enum('parsing', 'task'),
  `ref_entity_id` varchar(32) NOT NULL,
  `ref_entity_type` enum('build', 'task'),
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_event_queue_ref_entity_id_ref_entity_type_event_type` (`ref_entity_id`, `ref_entity_type`, `event_type`),
  CONSTRAINT `fk_event_queue_organizations_org_id` FOREIGN KEY (`org_id`) REFERENCES `organizations` (`id`)
);

INSERT
	INTO
	event_queue (
		id,
		status,
		org_id,
		event_type,
		ref_entity_id,
		ref_entity_type,
		created_at,
		updated_at
	)
SELECT
	id,
	status,
	org_id,
	'parsing',
	build_id,
	'build',
	created_at,
	updated_at
FROM
	parsing_queue;

INSERT
	INTO
	event_queue (
		id,
		status,
		org_id,
		event_type,
		ref_entity_id,
		ref_entity_type,
		created_at,
		updated_at
	)
SELECT
	id,
	status,
	org_id,
	'task',
	task_id,
	'task',
	created_at,
	updated_at
FROM
	task_queue;

DROP TABLE `parsing_queue`;
DROP TABLE `task_queue`;
