ALTER TABLE git_users
  DROP FOREIGN KEY fk_git_users_user_organizations,
  DROP INDEX user_org_id;

UPDATE user_organizations uo JOIN git_users gu ON gu.user_org_id=uo.user_id SET uo.user_id=gu.id;
ALTER TABLE user_organizations ADD  CONSTRAINT fk_user_organizations_git_users_user_id FOREIGN KEY (user_id) REFERENCES git_users(id);

ALTER TABLE git_users DROP COLUMN user_org_id;
