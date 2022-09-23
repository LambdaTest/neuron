package organizations

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries           = 3
	delay                = 250 * time.Millisecond
	maxJitter            = 100 * time.Millisecond
	errMsg               = "failed to perform organization transaction"
	insertQueryChunkSize = 1000
)

type organizationStore struct {
	db      core.DB
	logger  lumber.Logger
	redisDB core.RedisDB
}

// New returns a new OrganizationStore.
func New(db core.DB,
	redisDB core.RedisDB,
	logger lumber.Logger) core.OrganizationStore {
	return &organizationStore{db: db, redisDB: redisDB, logger: logger}
}

func (o *organizationStore) Create(ctx context.Context, orgs []*core.Organization) error {
	return o.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, orgs); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}
func (o *organizationStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, orgs []*core.Organization) error {
	return utils.Chunk(insertQueryChunkSize, len(orgs), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, orgs[start:end]); err != nil {
			return errs.SQLError(err)
		}
		return nil
	})
}
func (o *organizationStore) FindOrgsByNameTx(
	ctx context.Context,
	tx *sqlx.Tx,
	gitProvider core.SCMDriver,
	orgs []*core.Organization) ([]*core.Organization, error) {
	orgNames := make([]string, 0, len(orgs))
	orgz := make([]*core.Organization, 0, len(orgs))

	// make argument map
	for _, o := range orgs {
		orgNames = append(orgNames, o.Name)
	}

	// make argument map
	arg := map[string]interface{}{
		"name":         orgNames,
		"git_provider": gitProvider,
	}
	query, args, err := sqlx.Named("SELECT id,name,git_provider FROM organizations WHERE name IN (:name) AND git_provider=:git_provider", arg)
	if err != nil {
		o.logger.Errorf("failed to created named query, error %v", err)
		return nil, errs.SQLError(err)
	}

	query, args, err = sqlx.In(query, args...)
	if err != nil {
		o.logger.Errorf("failed to created IN query, error %v", err)
		return nil, errs.SQLError(err)
	}

	query = tx.Rebind(query)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		o.logger.Errorf("error while executing insert query, error %v", err)
		return nil, errs.SQLError(err)
	}
	defer rows.Close()

	for rows.Next() {
		o := new(core.Organization)
		if err = rows.Scan(&o.ID, &o.Name, &o.GitProvider); err != nil {
			return nil, errs.SQLError(err)
		}
		orgz = append(orgz, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orgz, nil
}

func (o *organizationStore) FindUserOrgs(ctx context.Context, userID string) ([]*core.Organization, error) {
	orgs := make([]*core.Organization, 0)

	return orgs, o.db.Execute(func(db *sqlx.DB) error {
		rows, err := db.QueryxContext(ctx, selectUserOrgsQuery, userID)
		if err != nil {
			return errs.SQLError(err)
		}
		defer rows.Close()

		for rows.Next() {
			org := new(core.Organization)
			if err := rows.StructScan(org); err != nil {
				o.logger.Errorf("Error in scanning rows, error: %v", err)
				return errs.SQLError(err)
			}
			orgs = append(orgs, org)
		}
		return nil
	})
}

func (o *organizationStore) FindUserOrgsTx(ctx context.Context, tx *sqlx.Tx, userID string) ([]*core.Organization, error) {
	orgs := make([]*core.Organization, 0)
	rows, err := tx.QueryxContext(ctx, selectUserOrgsQuery, userID)
	if err != nil {
		return nil, errs.SQLError(err)
	}
	defer rows.Close()

	for rows.Next() {
		org := new(core.Organization)
		if err := rows.StructScan(org); err != nil {
			o.logger.Errorf("Error in scanning rows, error: %v", err)
			return nil, errs.SQLError(err)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (o *organizationStore) Find(ctx context.Context, org *core.Organization) (orgID string, err error) {
	err = o.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id FROM organizations where name=? and git_provider=?"
		rows := db.QueryRowContext(ctx, selectQuery, org.Name, org.GitProvider)

		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		return nil
	})
	return
}

func (o *organizationStore) FindOrCreate(ctx context.Context, org *core.Organization) (orgID string, err error) {
	err = o.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		selectQuery := "SELECT id FROM organizations where name=? and git_provider=?"
		rows := tx.QueryRowContext(ctx, selectQuery, org.Name, org.GitProvider)
		err := rows.Scan(&orgID)

		// error is not of type no rows
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// org already exists return orgID
		if orgID != "" {
			return nil
		}

		// insert if not exists
		orgID = org.ID
		insertQuery := "INSERT INTO organizations(id, name, git_provider, avatar) VALUES (:id,:name,:git_provider,:avatar)"

		if _, err = tx.NamedExecContext(ctx, insertQuery, org); err != nil {
			return err
		}
		return nil
	})
	return
}

func (o *organizationStore) FindByID(ctx context.Context, orgID string) (*core.Organization, error) {
	organization := new(core.Organization)
	return organization, o.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id, name, git_provider, secret_key, runner_type FROM organizations WHERE id=?"
		rows := db.QueryRowxContext(ctx, selectQuery, orgID)
		return rows.StructScan(organization)
	})
}

func (o *organizationStore) FindRunnerType(ctx context.Context, orgID string) (core.Runner, error) {
	orgCache, err := o.GetOrgCache(ctx, orgID)
	if err != nil && !errors.Is(err, errs.ErrRedisKeyNotFound) {
		return "", err
	}
	if orgCache != nil && orgCache.RunnerType != "" {
		return core.Runner(orgCache.RunnerType), nil
	}
	org, err := o.FindByID(ctx, orgID)
	if err != nil {
		return "", err
	}
	orgKey := utils.GetOrgHashKey(orgID)
	_, err = o.redisDB.Client().HSet(ctx, orgKey, "runner_type", string(org.RunnerType)).Result()
	if err != nil {
		return "", err
	}
	return org.RunnerType, nil
}

func (o *organizationStore) FindBySecretKey(ctx context.Context, secretKey string) (*core.Organization, error) {
	organization := new(core.Organization)
	return organization, o.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id, name, git_provider, secret_key, runner_type FROM organizations WHERE secret_key=?"
		rows := db.QueryRowxContext(ctx, selectQuery, secretKey)
		return rows.StructScan(organization)
	})
}

func (o *organizationStore) GetOrgCache(ctx context.Context, orgID string) (*core.OrgDetailsCache, error) {
	var org core.OrgDetailsCache
	key := utils.GetOrgHashKey(orgID)
	exists, err := o.redisDB.Client().Exists(ctx, key).Result()
	if exists == 0 || err != nil {
		o.logger.Errorf("error while check if org keys exists orgID %s, error: %v", orgID, err)
		return nil, errs.ErrRedisKeyNotFound
	}
	cmd := o.redisDB.Client().HGetAll(ctx, key)
	if err := cmd.Scan(&org); err != nil {
		return nil, err
	}
	return &org, nil
}

func (o *organizationStore) UpdateKeysInCache(ctx context.Context, orgID string, concurrencyKeys map[string]int64) error {
	key := utils.GetOrgHashKey(orgID)
	exists, existsErr := o.redisDB.Client().Exists(ctx, key).Result()
	if exists == 0 || existsErr != nil {
		o.logger.Errorf("error while checking if org key exists for orgID %s, %v", orgID, existsErr)
		return errs.ErrRedisKeyNotFound
	}
	// pipeline the commands to reduce at least 1 RTT
	pipe := o.redisDB.Client().Pipeline()
	defer func() { _ = pipe.Close() }()

	for k, v := range concurrencyKeys {
		if !verifyConcurrencyKey(k) {
			o.logger.Errorf("invalid concurrency key %s for orgID", k, orgID)
			continue
		}
		// skip if value == 0
		if v == 0 {
			continue
		}

		if v < 0 {
			// if we want to decrement the key then we will check if key value on redis is > 0
			// so to avoid unnecessary decrement
			redisVal, err := o.redisDB.Client().HGet(ctx, key, k).Int()
			if err != nil {
				return err
			}
			if redisVal > 0 {
				pipe.HIncrBy(ctx, key, k, v)
			} else {
				o.logger.Warnf("skipping decrement of key %s for orgID %s, redis value is %d", k, orgID, redisVal)
			}
		} else {
			pipe.HIncrBy(ctx, key, k, v)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		o.logger.Errorf("error while updating keys for orgID %s, %v", orgID, err)
		return err
	}
	return nil
}

func (o *organizationStore) UpdateConfigInfo(ctx context.Context, orgID, secretKey, runnerType string) error {
	return o.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		synapseInfo := map[string]interface{}{
			"id":          orgID,
			"runner_type": runnerType,
			"secret_key":  secretKey,
			"concurrency": 50,
		}
		result, err := tx.NamedExecContext(ctx, updateRunnerInfoQuery, synapseInfo)
		if err != nil {
			return errs.SQLError(err)
		}
		if rowCount, err := result.RowsAffected(); err != nil {
			return errs.SQLError(err)
		} else if rowCount == 0 {
			return errs.ErrNotFound
		} else if runnerType == "self-hosted" {
			_, err := tx.NamedExecContext(ctx, updateConcurrencyQuery, synapseInfo)
			if err != nil {
				return errs.SQLError(err)
			}
		}
		return nil
	})
}

func (o *organizationStore) UpdateSynapseToken(ctx context.Context, orgID, secretKey, token string) error {
	return o.db.Execute(func(db *sqlx.DB) error {
		synapseInfo := map[string]interface{}{
			"id":         orgID,
			"token":      token,
			"secret_key": secretKey,
		}
		result, err := db.NamedExecContext(ctx, updateSynapseKeyQuery, synapseInfo)
		if err != nil {
			return errs.SQLError(err)
		}
		if rowCount, err := result.RowsAffected(); err != nil {
			return errs.SQLError(err)
		} else if rowCount == 0 {
			return errs.ErrNotFound
		}
		return nil
	})
}

func verifyConcurrencyKey(key string) bool {
	switch key {
	case "queued_tasks", "running_tasks":
		return true
	default:
		return false
	}
}

const insertQuery = `INSERT
INTO
organizations(id,
name,
git_provider,
avatar,
created_at,
updated_at)
VALUES (:id,
:name,
:git_provider,
:avatar,
:created_at,
:updated_at) ON
DUPLICATE KEY
UPDATE
updated_at =
VALUES(updated_at)`

const updateRunnerInfoQuery = `
UPDATE
    organizations
SET
	runner_type = :runner_type,
	secret_key = :secret_key
WHERE
	id =:id 
	AND secret_key IS NULL`

const updateSynapseKeyQuery = `
	UPDATE
		organizations
	SET
		secret_key = :secret_key
	WHERE
		id =:id 
		AND secret_key = :token`

const updateConcurrencyQuery = `
UPDATE
	license_info
SET
	concurrency = :concurrency 
WHERE
	org_id = :id `

const selectUserOrgsQuery = `SELECT
	o.id,
	o.name,
	o.avatar,
	o.git_provider,
	o.runner_type,
	COALESCE(o.secret_key, "") secret_key
FROM
	git_users gu
JOIN user_organizations uo ON
	uo.user_id = gu.id
JOIN organizations o ON
	o.id = uo.org_id
WHERE
	gu.id =?`
