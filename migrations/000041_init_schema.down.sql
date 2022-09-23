ALTER TABLE repositories RENAME COLUMN strict TO run_on_each_commit;
ALTER TABLE repositories MODIFY COLUMN run_on_each_commit BOOLEAN DEFAULT 1;
