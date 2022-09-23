DROP INDEX uq_organizations_secret_key ON organizations;

ALTER TABLE organizations
  DROP COLUMN secret_key,
  DROP COLUMN runner_type;

ALTER  TABLE task DROP COLUMN container_image;