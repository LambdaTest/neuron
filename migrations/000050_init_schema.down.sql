ALTER TABLE repositories ADD COLUMN custom BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE task DROP COLUMN remark;
