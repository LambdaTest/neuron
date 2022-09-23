package userdemo

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type userDemoStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new UserDemoStore
func New(db core.DB, logger lumber.Logger) core.UserDemoStore {
	return &userDemoStore{db: db, logger: logger}
}

func (ud *userDemoStore) Create(ctx context.Context, info *core.UserInfoDemoDetails) error {
	return ud.db.Execute(func(db *sqlx.DB) error {
		query := insertUserDataQuery
		if _, err := db.NamedExecContext(ctx, query, info); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

const insertUserDataQuery = `
INSERT
	INTO
	users_demo_info(id,
	name,
	email_id,
	company_name)
VALUES (:id,
		:name,
		:email_id,
		:company_name) ON
DUPLICATE KEY
UPDATE
	updated_at = NOW()
`
