CREATE TABLE IF NOT EXISTS test_suite_execution
(
    id               VARCHAR(32) NOT NULL,
    suite_id         VARCHAR(32) NOT NULL,
    commit_id        VARCHAR(40) NOT NULL,
    status           enum ('completed','aborted','failed','passed','blacklisted','skipped'),
    created_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    start_time       TIMESTAMP,
    end_time         TIMESTAMP,
    duration         BIGINT      NOT NULL DEFAULT 0,
    stdout           TEXT,
    stderr           TEXT,
    blacklist_source VARCHAR(255),
    build_id         VARCHAR(32) NOT NULL,
    task_id          VARCHAR(32) NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (suite_id) REFERENCES test_suite (id),
    FOREIGN KEY (commit_id) REFERENCES git_commits (commit_id),
    FOREIGN KEY (build_id) REFERENCES build (id),
    FOREIGN KEY (task_id) REFERENCES task (id)
);

CREATE TABLE IF NOT EXISTS test_suite_metrics
(
    id                      VARCHAR(32)  NOT NULL,
    test_suite_execution_id VARCHAR(32)  NOT NULL,
    cpu                     FLOAT        NOT NULL DEFAULT 0.0,
    memory                  BIGINT       NOT NULL DEFAULT 0,
    storage                 BIGINT       NOT NULL DEFAULT 0,
    record_time             TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (test_suite_execution_id) REFERENCES test_suite_execution (id)
);
