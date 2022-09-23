ALTER TABLE organizations
  ADD COLUMN secret_key VARCHAR(32) ,
  ADD COLUMN runner_type ENUM ('cloud-runner', 'self-hosted') NOT NULL,
  ADD CONSTRAINT uq_organizations_secret_key UNIQUE(secret_key);

ALTER  TABLE task ADD COLUMN container_image text;