package branch

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/errors"

	"github.com/LambdaTest/neuron/pkg/core"

	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type branchStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new BranchStore
func New(db core.DB, logger lumber.Logger) core.BranchStore {
	return &branchStore{db: db, logger: logger}
}

// CreateInTx new branch in datastore
func (g *branchStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, branch *core.Branch) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, branch); err != nil {
		return err
	}
	return nil
}

func (g *branchStore) FindByRepo(ctx context.Context, repoID string) (branchList []string, err error) {
	return branchList, g.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoID}
		query := listBranchQuery

		rows, err := db.QueryxContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var branch string

			if err = rows.Scan(
				&branch); err != nil {
				return errors.SQLError(err)
			}
			branchList = append(branchList, branch)
		}
		return nil
	})
}

const insertQuery = `INSERT INTO branch(id, repo_id, name) 
VALUES (:id,:repo_id,:name) 
ON DUPLICATE KEY UPDATE updated_at=VALUES(updated_at)`

const listBranchQuery = `
SELECT
	b.name
FROM
	branch b
WHERE
    b.repo_id = ? 
`
