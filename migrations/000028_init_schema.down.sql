ALTER TABLE `build` DROP FOREIGN KEY build_branch_name_repoid_fk;

ALTER TABLE `build` DROP COLUMN `branch_name`;
