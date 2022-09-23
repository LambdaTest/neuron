-- Added this to add total-tests counts relative to each test-suites in commit discovery table to get branch dependdency
WITH suite_total_test_mapping AS (
	SELECT
		cd.id,
		tt.test_suite_id,
		ts.total_tests
	FROM
		commit_discovery cd
	JOIN JSON_TABLE(
			cd.suite_ids,
			'$[*]' COLUMNS(
				test_suite_id varchar(32) PATH '$' ERROR ON
				ERROR
			)
		) tt
	JOIN test_suite ts ON
		ts.id = tt.test_suite_id COLLATE utf8mb4_0900_ai_ci
),
updated_commit_discovery AS (
	SELECT
		sttm.id,
		JSON_ARRAYAGG(JSON_OBJECT('suite_id', sttm.test_suite_id, 'total_tests', sttm.total_tests)) suite_ids
	FROM
		suite_total_test_mapping sttm
	GROUP BY sttm.id
)
UPDATE commit_discovery cd
JOIN updated_commit_discovery ucd ON
	cd.id = ucd.id
SET cd.suite_ids = ucd.suite_ids;
