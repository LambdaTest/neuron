UPDATE  task SET tier='small' WHERE tier='xsmall';

ALTER TABLE task
    MODIFY COLUMN tier enum('small','medium','large','xlarge') NOT NULL DEFAULT 'small' NOT NULL;
