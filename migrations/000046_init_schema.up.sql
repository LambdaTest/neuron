ALTER TABLE task 
    DROP FOREIGN KEY  task_ibfk_2,
    DROP COLUMN commit_id,
    DROP COLUMN base_commit_id;
