WITH suite_total_test_mapping AS (
	SELECT
		cd.id,
		tt.test_suite_id
	FROM
		commit_discovery cd
	JOIN JSON_TABLE(
			cd.suite_ids,
			'$[*]' COLUMNS(
				test_suite_id varchar(32) PATH '$.suite_id' ERROR ON
				ERROR
			)
		) tt
),
updated_commit_discovery AS (
	SELECT
		sttm.id,
		JSON_ARRAYAGG(sttm.test_suite_id) suite_ids
	FROM
		suite_total_test_mapping sttm
	GROUP BY sttm.id
)
UPDATE commit_discovery cd
JOIN updated_commit_discovery ucd ON
	cd.id = ucd.id
SET cd.suite_ids = ucd.suite_ids;
