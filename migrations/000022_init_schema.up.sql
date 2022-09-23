UPDATE task
    SET status = 'passed' WHERE status = 'completed';

ALTER table task
    MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error') NOT NULL;

UPDATE build
    SET status = 'passed' WHERE status = 'completed';

ALTER table build
    MODIFY COLUMN status enum('running','failed','passed','initiating','aborted','error') NOT NULL;
