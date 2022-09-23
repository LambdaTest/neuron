DROP TABLE IF EXISTS job_queue;

CREATE TABLE IF NOT EXISTS task_queue (
  id varchar(32) PRIMARY KEY,
  status ENUM ("ready", "processing", "completed") NOT NULL,
  org_id varchar(32) NOT NULL,
  task_id varchar(32) NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_task_queue_status_created_at (status, created_at),
  CONSTRAINT fk_task_queue_organizations_org_id FOREIGN KEY (org_id) REFERENCES organizations (id),
  CONSTRAINT fk_task_queue_task_task_id FOREIGN KEY (task_id) REFERENCES task (id)
);
