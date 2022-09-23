SET FOREIGN_KEY_CHECKS = 0;

ALTER TABLE `branch`
    MODIFY `name` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

ALTER TABLE `branch_commit`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

ALTER TABLE `build`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET latin1 NOT NULL,
    ADD COLUMN `actor` VARCHAR(255);

ALTER TABLE `commit_cnt_since_last_postmerge`
    MODIFY `branch` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

ALTER TABLE `credits_usage`
    MODIFY `user` VARCHAR(32) CHARACTER SET latin1 DEFAULT NULL;

ALTER TABLE `git_commits`
    MODIFY `author_name` VARCHAR(512) CHARACTER SET latin1 NOT NULL,
    MODIFY `committer_name` VARCHAR(512) CHARACTER SET latin1 DEFAULT NULL,
    MODIFY `message` TEXT CHARACTER SET latin1;

ALTER TABLE `git_events`
    ADD COLUMN `response` TEXT,
    ADD COLUMN `status_code` SMALLINT;

ALTER TABLE `post_merge_config`
    MODIFY `branch` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

ALTER TABLE `test_branch`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

ALTER TABLE `test_suite_branch`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET latin1 NOT NULL;

SET FOREIGN_KEY_CHECKS = 1;
