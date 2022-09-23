ALTER TABLE repositories DROP COLUMN custom;
ALTER TABLE task ADD COLUMN remark varchar(1024) DEFAULT NULL;
