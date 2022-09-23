package postmerge

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type postMergeConfigStore struct {
	db     core.DB
	logger lumber.Logger
}

// NewPostMergeConfigStore returns a new postMergeConfigStore.
func NewPostMergeConfigStore(db core.DB, logger lumber.Logger) core.PostMergeConfigStore {
	return &postMergeConfigStore{db: db, logger: logger}
}

// Create persists a new postmergeconfig in the datastore within specified transaction
func (pmcs *postMergeConfigStore) Create(ctx context.Context, postMergeConfig *core.PostMergeConfig) error {
	return pmcs.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, postMergeConfig); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

// Update updates existing postmergeconfig in the datastore
func (pmcs *postMergeConfigStore) Update(ctx context.Context, postMergeConfig *core.PostMergeConfig) error {
	return pmcs.db.Execute(func(db *sqlx.DB) error {
		result, err := db.NamedExecContext(ctx, updateQuery, postMergeConfig)
		if err != nil {
			return errs.SQLError(err)
		}
		if count, err := result.RowsAffected(); err != nil {
			return errs.SQLError(err)
		} else if count == 0 {
			return errs.ErrNotFound
		}
		return nil
	})
}

// FindAllPostMergeConfig returns all postmergeconfig by repoID
func (pmcs *postMergeConfigStore) FindAllPostMergeConfig(ctx context.Context,
	repoID string) (postMergeConfigList []*core.PostMergeConfig, sqlErr error) {
	postMergeConfigList = make([]*core.PostMergeConfig, 0)
	sqlErr = pmcs.db.Execute(func(db *sqlx.DB) error {
		args := []interface{}{repoID}
		rows, err := db.QueryxContext(ctx, findAllPostMergeConfigQuery, args...)
		if rows.Err() != nil {
			return rows.Err()
		}
		if err != nil {
			pmcs.logger.Errorf("failed to find postmergeconfig error:%v", err)
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			postMergeConfig := new(core.PostMergeConfig)
			if err = rows.StructScan(&postMergeConfig); err != nil {
				return errs.SQLError(err)
			}
			postMergeConfigList = append(postMergeConfigList, postMergeConfig)
		}
		return nil
	})

	if sqlErr != nil {
		return nil, sqlErr
	}

	if postMergeConfigList == nil {
		return nil, errs.ErrNotFound
	}
	return postMergeConfigList, nil
}

// FindActivePostMergeConfigInTx returns postmergeconfig by repoid and branch name within specified transaction
func (pmcs *postMergeConfigStore) FindActivePostMergeConfigInTx(ctx context.Context, tx *sqlx.Tx,
	repoID, branchName string) (postMergeConfig *core.PostMergeConfig, err error) {
	var postMergeConfigSpecificBranch, postMergeConfigBranchALL *core.PostMergeConfig
	args := []interface{}{repoID, branchName}
	rows, err := tx.QueryxContext(ctx, findPostMergeConfigQuery, args...)
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if err != nil {
		pmcs.logger.Errorf("failed to find postmergeconfig error:%v", err)
		return nil, errs.SQLError(err)
	}
	defer rows.Close()
	for rows.Next() {
		postMergeConfig = new(core.PostMergeConfig)
		if err = rows.StructScan(&postMergeConfig); err != nil {
			return nil, errs.SQLError(err)
		}

		if postMergeConfig.Branch == branchName {
			postMergeConfigSpecificBranch = postMergeConfig
		}

		if postMergeConfig.Branch == "*" {
			postMergeConfigBranchALL = postMergeConfig
		}
	}

	if postMergeConfigBranchALL == nil && postMergeConfigSpecificBranch == nil {
		return nil, errs.ErrRowsNotFound
	}
	if postMergeConfigSpecificBranch != nil {
		return postMergeConfigSpecificBranch, nil
	}
	return postMergeConfigBranchALL, nil
}

const insertQuery = `INSERT
INTO
post_merge_config(id,
repo_id,
branch,
is_active,
strategy_name,
threshold,
created_at,
updated_at)
VALUES (:id,
:repo_id,
:branch,	
:is_active,
:strategy_name,
:threshold,
:created_at,
:updated_at)`

const findPostMergeConfigQuery = `
SELECT
    id,
	repo_id,
	branch,
	is_active,
	strategy_name,
	threshold,
	created_at,
	updated_at
FROM
	post_merge_config 
WHERE
    is_active=1
	AND
	repo_id=?
	AND
	(branch=?
	OR 
	branch="*")`

const findAllPostMergeConfigQuery = `
SELECT
    id,
	repo_id,
	branch,
	is_active,
	strategy_name,
	threshold,
	created_at,
	updated_at
FROM
	post_merge_config 
WHERE
	repo_id=?`

const updateQuery = `
UPDATE
    post_merge_config
SET
	branch =:branch,
	is_active =:is_active,
	strategy_name =:strategy_name,
	threshold =:threshold,
	updated_at =:updated_at
WHERE
	id =:id`
