ALTER table parsing_queue
    MODIFY COLUMN status enum('ready','processing','completed', 'failed') NOT NULL,
    ADD INDEX idx_parsing_queue_status_created_at (status, created_at);


ALTER table task_queue
    MODIFY COLUMN status enum('ready','processing','completed', 'failed') NOT NULL,
    ADD CONSTRAINT uq_task_queue_task_id UNIQUE(task_id);
