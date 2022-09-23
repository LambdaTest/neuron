ALTER TABLE user_organizations DROP FOREIGN KEY fk_user_organizations_git_users_user_id;

ALTER TABLE git_users
  ADD COLUMN user_org_id VARCHAR(32) NOT NULL,
  ADD CONSTRAINT UNIQUE(user_org_id);
  

UPDATE git_users SET user_org_id=REPLACE(UUID(),'-','');
UPDATE user_organizations uo JOIN git_users gu ON gu.id=uo.user_id SET uo.user_id=gu.user_org_id;

ALTER TABLE git_users ADD CONSTRAINT fk_git_users_user_organizations FOREIGN KEY (user_org_id) REFERENCES user_organizations(user_id);
