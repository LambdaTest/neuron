package userinfo

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type userInfoStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new UserInfoStore
func New(db core.DB, logger lumber.Logger) core.UserInfoStore {
	return &userInfoStore{db: db, logger: logger}
}

func (ui *userInfoStore) Create(ctx context.Context, userInfo *core.UserInfo) error {
	return ui.db.Execute(func(db *sqlx.DB) error {
		query := insertUserQuery
		if _, err := db.NamedExecContext(ctx, query, userInfo); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (ui *userInfoStore) Find(ctx context.Context, userID, orgID string) (*core.UserInfo, error) {
	user := new(core.UserInfo)
	return user, ui.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, findInfoQuery, userID, orgID)
		if err := rows.StructScan(user); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (ui *userInfoStore) UpdateActiveUser(ctx context.Context, userInfo *core.UserInfo) error {
	return ui.db.Execute(func(db *sqlx.DB) error {
		query := updateActiveUserInfo
		if _, err := db.NamedExecContext(ctx, query, userInfo); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

const insertUserQuery = ` INSERT
INTO
user_info(id,
user_id,
user_description,
experience,
team_size,
org_id,
created_at)
VALUES (:id,
:user_id,
:user_description,
:experience,
:team_size,
:org_id,
:created_at) ON
DUPLICATE KEY
UPDATE
updated_at = NOW(),
user_description =
VALUES(user_description),
experience =
VALUES(experience),
team_size =
VALUES(team_size) `

const findInfoQuery = `
SELECT
	ui.user_id,
	COALESCE(ui.user_description, "") user_description,
	ui.experience,
	ui.team_size,
	ui.org_id,
	ui.is_active
FROM
	user_info ui
WHERE
	ui.user_id = ?
	AND ui.org_id = ?`

const updateActiveUserInfo = `INSERT
INTO
	user_info(id,
	user_id,
	org_id,
	created_at,
	is_active)
VALUES (:id,
	:user_id,
	:org_id,
	:created_at,
	:is_active) ON
DUPLICATE KEY
	UPDATE
	updated_at = NOW(),
	is_active =
	VALUES(is_active) `
