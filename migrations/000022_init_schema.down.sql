ALTER table task
    MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','completed') NOT NULL;

UPDATE task
    SET status = 'completed' WHERE status = 'passed';

ALTER table build
    MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','completed') NOT NULL;

UPDATE build
    SET status = 'completed' WHERE status = 'passed';
