CREATE TABLE IF NOT EXISTS users_demo_info (
  id varchar(32) NOT NULL,
  name varchar(32) NOT NULL,
  email_id varchar(32) NOT NULL,
  company_name varchar(32),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT pk_users_demo_info_id PRIMARY KEY (id),
  CONSTRAINT unique_users_demo_info_email_id UNIQUE(email_id)
);
