ALTER TABLE `build` ADD COLUMN `branch_name` VARCHAR(255) NOT NULL;

-- Best guess to map existing builds to branches based on commits; being done so that fk constraint can be added --
UPDATE
	`build` b
JOIN (
		SELECT
			b.id,
			bc.branch_name
		FROM
			build b
		JOIN task t ON
			t.build_id = b.id
		JOIN git_commits gc ON
			gc.commit_id = t.commit_id
		JOIN branch_commit bc ON
			bc.commit_id = gc.commit_id
	) build_branch_mapping ON
	build_branch_mapping.id = b.id
SET
	b.branch_name = build_branch_mapping.branch_name;

ALTER TABLE `build` ADD CONSTRAINT build_branch_name_repoid_fk FOREIGN KEY (
	`repo_id`,
	`branch_name`
) REFERENCES `branch` (
	`repo_id`,
	`name`
);
