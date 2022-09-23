package flakyconfigstore

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type flakyConfigStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new FlakyConfigStore.
func New(db core.DB, logger lumber.Logger) core.FlakyConfigStore {
	return &flakyConfigStore{db: db, logger: logger}
}

// Create persists a new flakyConfig in the datastore
func (fcs *flakyConfigStore) Create(ctx context.Context, flakyConfig *core.FlakyConfig) error {
	return fcs.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, flakyConfig); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

// Update updates existing flakyConfig in the datastore
func (fcs *flakyConfigStore) Update(ctx context.Context, flakyConfig *core.FlakyConfig) error {
	return fcs.db.Execute(func(db *sqlx.DB) error {
		result, err := db.NamedExecContext(ctx, updateQuery, flakyConfig)
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

// FindAllFlakyConfig returns all flakyConfig by repoID
func (fcs *flakyConfigStore) FindAllFlakyConfig(ctx context.Context,
	repoID string) (flakyConfigList []*core.FlakyConfig, sqlErr error) {
	flakyConfigList = make([]*core.FlakyConfig, 0)
	sqlErr = fcs.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id": repoID,
		}
		rows, err := db.NamedQueryContext(ctx, findAllFlakyConfigQuery, args)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			flakyConfig := new(core.FlakyConfig)
			if err = rows.StructScan(&flakyConfig); err != nil {
				return errs.SQLError(err)
			}
			flakyConfigList = append(flakyConfigList, flakyConfig)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		return nil
	})

	if sqlErr != nil {
		return nil, sqlErr
	}

	if len(flakyConfigList) == 0 {
		return nil, errs.ErrRowsNotFound
	}
	return flakyConfigList, nil
}

func (fcs *flakyConfigStore) FindIfActiveConfigExists(ctx context.Context,
	repoID, branchName string, configType core.FlakyConfigType) (err error) {
	return fcs.db.Execute(func(db *sqlx.DB) error {
		var exists bool
		rows := db.QueryRowContext(ctx, findIfFlakyConfigExistsQuery, repoID, configType, branchName)
		if err := rows.Scan(&exists); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}

func (fcs *flakyConfigStore) FindActiveFlakyConfig(ctx context.Context,
	repoID, branchName string, configType core.FlakyConfigType) (flakyConfig *core.FlakyConfig, err error) {
	var flakyConfigSpecificBranch, flakyConfigBranchALL *core.FlakyConfig

	dbErr := fcs.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"repo_id":     repoID,
			"branch":      branchName,
			"config_type": configType,
		}
		rows, err := db.NamedQueryContext(ctx, findFlakyConfigQuery, args)
		if err != nil {
			fcs.logger.Errorf("failed to find flakyConfig error:%v", err)
			return errs.SQLError(err)
		}
		defer rows.Close()
		for rows.Next() {
			flakyConfig = new(core.FlakyConfig)
			if err = rows.StructScan(&flakyConfig); err != nil {
				return errs.SQLError(err)
			}

			if flakyConfig.Branch == branchName {
				flakyConfigSpecificBranch = flakyConfig
			}

			if flakyConfig.Branch == "*" {
				flakyConfigBranchALL = flakyConfig
			}
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		return nil
	})

	if dbErr != nil {
		return nil, dbErr
	}
	if flakyConfigBranchALL == nil && flakyConfigSpecificBranch == nil {
		return nil, errs.ErrRowsNotFound
	}
	// we give priority to specific branch if it exists
	if flakyConfigSpecificBranch != nil {
		return flakyConfigSpecificBranch, nil
	}
	return flakyConfigBranchALL, nil
}

const insertQuery = `INSERT
INTO
flaky_config(id,
repo_id,
branch,
is_active,
algo_name,
consecutive_runs,
auto_quarantine,
threshold,
config_type,
created_at,
updated_at)
VALUES (:id,
:repo_id,
:branch,	
:is_active,
:algo_name,
:consecutive_runs,
:auto_quarantine,
:threshold,
:config_type,
:created_at,
:updated_at)
ON
DUPLICATE KEY
UPDATE
is_active = 
VALUES(is_active),
algo_name =
VALUES(algo_name),
consecutive_runs =
VALUES(consecutive_runs),
auto_quarantine =
VALUES(auto_quarantine),
threshold =
VALUES(threshold),
updated_at =
VALUES(updated_at);`

const findIfFlakyConfigExistsQuery = `
SELECT 1 
	FROM 
flaky_config 
WHERE
    is_active=1
	AND repo_id=?
	AND config_type=?
	AND branch=?`

const findFlakyConfigQuery = `
SELECT
    id,
	repo_id,
	branch,
	is_active,
	algo_name,
	auto_quarantine,
	consecutive_runs,
	threshold,
	config_type,	
	created_at,
	updated_at
FROM
	flaky_config 
WHERE
    is_active=1
	AND
	repo_id=:repo_id
	AND 
	config_type = :config_type
	AND
	(branch=:branch
	OR 
	branch="*")`

const findAllFlakyConfigQuery = `
SELECT
    id,
	repo_id,
	branch,
	is_active,
	algo_name,
	consecutive_runs,
	auto_quarantine,
	threshold,
	config_type,
	created_at,
	updated_at
FROM
	flaky_config 
WHERE
	repo_id=:repo_id
	AND
	is_active=1`

const updateQuery = `
UPDATE
    flaky_config
SET
	branch =:branch,
	is_active =:is_active,
	algo_name =:algo_name,
	consecutive_runs = :consecutive_runs,
    threshold =:threshold,
	auto_quarantine= :auto_quarantine,
	updated_at =:updated_at
WHERE
	id =:id`
