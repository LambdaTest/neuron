SET FOREIGN_KEY_CHECKS = 0;

ALTER TABLE `branch`
    MODIFY `name` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

ALTER TABLE `branch_commit`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

ALTER TABLE `build`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL,
    DROP COLUMN `actor`;

ALTER TABLE `commit_cnt_since_last_postmerge`
    MODIFY `branch` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

ALTER TABLE `credits_usage`
    MODIFY `user` VARCHAR(32) CHARACTER SET utf8mb4 DEFAULT NULL;

ALTER TABLE `git_commits`
    MODIFY `author_name` VARCHAR(512) CHARACTER SET utf8mb4 NOT NULL,
    MODIFY `committer_name` VARCHAR(512) CHARACTER SET utf8mb4 DEFAULT NULL,
    MODIFY `message` TEXT CHARACTER SET utf8mb4;

-- `event_payload` is of `json` type which stores in utf8mb4 https://dev.mysql.com/doc/refman/5.7/en/json.html --
ALTER TABLE `git_events`
    DROP COLUMN `response`,
    DROP COLUMN `status_code`;

ALTER TABLE `post_merge_config`
    MODIFY `branch` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

ALTER TABLE `test_branch`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

ALTER TABLE `test_suite_branch`
    MODIFY `branch_name` VARCHAR(255) CHARACTER SET utf8mb4 NOT NULL;

SET FOREIGN_KEY_CHECKS = 1;
