ALTER TABLE post_merge_config
  ADD COLUMN patterns JSON NOT NULL;

ALTER TABLE post_merge_config
  ADD COLUMN commit_cnt_since_last_postmerge int;

DROP TABLE IF EXISTS commit_cnt_since_last_postmerge;
