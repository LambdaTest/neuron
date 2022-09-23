ALTER TABLE repositories RENAME COLUMN run_on_each_commit TO strict;
ALTER TABLE repositories  MODIFY COLUMN strict BOOLEAN DEFAULT 0;
UPDATE repositories SET strict=0;
