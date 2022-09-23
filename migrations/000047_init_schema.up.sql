CREATE TABLE IF NOT EXISTS flaky_test_execution (
  id varchar(32) NOT NULL,
  test_id VARCHAR(32) NOT NULL,
  algo_name enum('running_x_times') NOT NULL,
  exec_info JSON NOT NULL,
  status enum('flaky', 'nonflaky') NOT NULL,
  build_id VARCHAR(32) NOT NULL,
  task_id VARCHAR(32) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_flaky_test_execution_id PRIMARY KEY (id),
  CONSTRAINT fk_flaky_test_execution_test_test_id FOREIGN KEY (test_id) REFERENCES test(id),
  CONSTRAINT fk_flaky_test_execution_build_build_id FOREIGN KEY (build_id) REFERENCES build(id),
  CONSTRAINT fk_flaky_test_execution_task_task_id FOREIGN KEY (task_id) REFERENCES task(id),
  INDEX flaky_test_execution_status_idx (status)
);

CREATE TABLE IF NOT EXISTS flaky_config (
  id varchar(32) NOT NULL,
  repo_id VARCHAR(40) NOT NULL,
  branch VARCHAR(255) NOT NULL,
  is_active BOOLEAN DEFAULT '1',
  algo_name enum('running_x_times') NOT NULL,
  consecutive_runs int DEFAULT '1',
  threshold int DEFAULT '1',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_flaky_config_id PRIMARY KEY (id),
  UNIQUE INDEX uq_idx_flaky_config_repo_id_branch (repo_id, branch),
  CONSTRAINT fk_flaky_config_repositories_repo_id FOREIGN KEY (repo_id) REFERENCES repositories(id)
);
