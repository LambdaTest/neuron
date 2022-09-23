CREATE TABLE IF NOT EXISTS test_metrics(
    id VARCHAR(32) NOT NULL,
    test_execution_id VARCHAR(32) NOT NULL,
    cpu FLOAT NOT NULL DEFAULT 0.0,
    record_time TIMESTAMP(6) NOT NULL,
    memory BIGINT NOT NULL DEFAULT 0,
    storage BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    FOREIGN KEY (test_execution_id) REFERENCES test_execution(id)
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
