ALTER table parsing_queue
    MODIFY COLUMN status enum('ready','processing','completed') NOT NULL,
    DROP INDEX idx_parsing_queue_status_created_at;

ALTER table task_queue
    MODIFY COLUMN status enum('ready','processing','completed') NOT NULL,
    ADD INDEX  idx_task_queue_task_id (task_id),
    DROP INDEX uq_task_queue_task_id;
