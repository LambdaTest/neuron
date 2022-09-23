DROP TABLE IF EXISTS task_queue;

CREATE TABLE IF NOT EXISTS job_queue (
  id varchar(32) PRIMARY KEY,
  status ENUM ("ready", "processing") NOT NULL,
  org_id varchar(32) NOT NULL,
  task_id varchar(32) NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX created_at_idx (created_at),
  INDEX status_idx (status),
  FOREIGN KEY (org_id) REFERENCES organizations (id),
  FOREIGN KEY (task_id) REFERENCES task (id)
);
