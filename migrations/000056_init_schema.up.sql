ALTER TABLE event_queue MODIFY COLUMN status enum('ready','processing','completed','failed', 'aborted') NOT NULL;
