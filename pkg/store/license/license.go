package license

import (
	"context"
	"database/sql"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type licenseStore struct {
	db     core.DB
	logger lumber.Logger
}

const insertQueryChunkSize = 1000

// New returns a new licenseStore
func New(db core.DB, logger lumber.Logger) core.LicenseStore {
	return &licenseStore{db: db, logger: logger}
}

func (ls *licenseStore) Create(ctx context.Context, license *core.License) error {
	return ls.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, license); err != nil {
			return err
		}
		return nil
	})
}

func (ls *licenseStore) Find(ctx context.Context, orgID string) (license *core.License, err error) {
	license = new(core.License)
	return license, ls.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectQuery, orgID)
		if err := rows.StructScan(license); err != nil {
			return err
		}
		return nil
	})
}

func (ls *licenseStore) FindInTx(ctx context.Context, tx *sqlx.Tx, orgID string) (license *core.License, err error) {
	license = new(core.License)
	rows := tx.QueryRowxContext(ctx, selectQuery, orgID)
	if err := rows.StructScan(license); err != nil {
		return nil, err
	}
	return license, nil
}

func (ls *licenseStore) FindOrCreateInTx(ctx context.Context, tx *sqlx.Tx, orgID string) (*core.License, error) {
	license := &core.License{
		ID:             utils.GenerateUUID(),
		Type:           core.Basic,
		OrgID:          orgID,
		CreditsBought:  constants.DefaultCredits,
		CreditsBalance: constants.DefaultCredits,
		CreditsExpiry:  time.Now().AddDate(0, 0, constants.Base10),
		Concurrency:    2,
		Tier:           core.Small,
	}
	err := tx.QueryRowxContext(ctx, selectQuery, orgID).StructScan(license)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	}
	return license, nil
}

func (ls *licenseStore) Update(ctx context.Context, license *core.License) error {
	return ls.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, updateQuery, license); err != nil {
			return err
		}
		return nil
	})
}

func (ls *licenseStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, orgz []*core.Organization) error {
	licenses := make([]*core.License, 0)
	for i := 0; i < len(orgz); i++ {
		license := &core.License{
			ID:             utils.GenerateUUID(),
			Type:           core.Basic,
			OrgID:          orgz[i].ID,
			CreditsBought:  constants.DefaultCredits,
			CreditsBalance: constants.DefaultCredits,
			CreditsExpiry:  time.Now().AddDate(0, 0, constants.Base10),
			Concurrency:    2,
			Tier:           core.Small,
		}
		licenses = append(licenses, license)
	}
	return utils.Chunk(insertQueryChunkSize, len(licenses), func(start int, end int) error {
		if _, err := tx.NamedExecContext(ctx, insertQuery, licenses[start:end]); err != nil {
			return err
		}
		return nil
	})
}

const selectQuery = `
SELECT
	id,
	type,
	org_id,
	credits_bought,
	credits_balance,
	credits_expiry,
	concurrency,
	tier
FROM
	license_info
WHERE
	org_id = ?
`

const insertQuery = `
INSERT INTO
	license_info(
		id,
		type,
		org_id,
		credits_bought,
		credits_balance,
		credits_expiry,
		concurrency,
		tier
	)
VALUES (
	:id,
	:type,
	:org_id,
	:credits_bought,
	:credits_balance,
	:credits_expiry,
	:concurrency,
	:tier
)
ON DUPLICATE KEY UPDATE
	updated_at = updated_at
`

const updateQuery = `
UPDATE
	license_info
SET
	type = :type,
	org_id = :org_id,
	credits_bought = :credits_bought,
	credits_balance = :credits_balance,
	credits_expiry = :credits_expiry,
	concurrency = :concurrency,
	tier = :tier
WHERE
	id = :id
`
