ALTER TABLE task
    MODIFY COLUMN tier enum('xsmall','small','medium','large','xlarge') NOT NULL DEFAULT 'small' NOT NULL;
