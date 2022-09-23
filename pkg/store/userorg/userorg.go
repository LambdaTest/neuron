package userorg

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type userOrgStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new OrganizationStore.
func New(db core.DB, logger lumber.Logger) core.UserOrgStore {
	return &userOrgStore{db, logger}
}

func (uo *userOrgStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, userOrg []*core.UserOrg) error {
	return utils.Chunk(insertQueryChunkSize, len(userOrg), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, userOrg[start:end]); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (uo *userOrgStore) FindIfExists(ctx context.Context, userID, orgID string) error {
	return uo.db.Execute(func(db *sqlx.DB) error {
		var exists bool
		row := db.QueryRowxContext(ctx, selectQuery, userID, orgID)
		if err := row.Scan(&exists); err != nil {
			uo.logger.Errorf("error in scanning rows, error: %v", err)
			return errors.SQLError(err)
		}
		return nil
	})
}

const insertQuery = `INSERT
INTO
user_organizations(id,
user_id,
org_id,
created_at,
updated_at)
VALUES (:id,
:user_id,
:org_id,
:created_at,
:updated_at) ON
DUPLICATE KEY
UPDATE
updated_at =
VALUES(updated_at)`

const selectQuery = `SELECT 1 FROM user_organizations WHERE user_id=? AND org_id=?`
