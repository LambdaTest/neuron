package core

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// LicenseType specifies the type of the License
type LicenseType string

// Tier specifies the license type.
type Tier string

// LicenseType values.
const (
	Basic      LicenseType = "Basic"
	Teams      LicenseType = "Teams"
	Enterprise LicenseType = "Enterprise"
)

// LicenseTier values.
const (
	Internal Tier = "internal"
	XSmall   Tier = "xsmall"
	Small    Tier = "small"
	Medium   Tier = "medium"
	Large    Tier = "large"
	XLarge   Tier = "xlarge"
)

var tierEnumMapping = map[Tier]int{
	XSmall: 1,
	Small:  2,
	Medium: 3,
	Large:  4,
	XLarge: 5,
}

// License represents information pertaining to license given to an org
type License struct {
	ID             string      `db:"id"`
	Type           LicenseType `db:"type" json:"license_type"`
	OrgID          string      `db:"org_id" json:"org_id" binding:"required"`
	CreditsBought  int64       `db:"credits_bought" json:"credits_bought"`
	CreditsBalance int64       `db:"credits_balance"`
	CreditsExpiry  time.Time   `db:"credits_expiry" json:"credits_expiry"`
	Concurrency    int64       `db:"concurrency" json:"concurrency"`
	Tier           Tier        `db:"tier" json:"tier"`
	Created        time.Time   `db:"created_at"`
	Updated        time.Time   `db:"updated_at"`
}

// LicenseStore defines datastore operation for working with license
type LicenseStore interface {
	// Create persists a new license in the datastore.
	Create(ctx context.Context, l *License) error
	// Find returns a license from the datastore.
	Find(ctx context.Context, orgID string) (*License, error)
	// FindInTx returns a license from the datastore and executes the statement in the given transaction.
	FindInTx(ctx context.Context, tx *sqlx.Tx, orgID string) (license *License, err error)
	// Update the license info in the datastore.
	Update(ctx context.Context, l *License) error
	// CreateInTx creates the new license in datastore.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, orgz []*Organization) error
}

// TierConfig the license tier config values.
type TierConfig struct {
	CPUs      string
	RAM       string
	Storage   string
	CreditsPM int32
}

// LicenseTierConfigMapping is used for License mapping
var LicenseTierConfigMapping = map[Tier]TierConfig{
	"small":  {CPUs: "2", RAM: "4Gi", Storage: "4Gi", CreditsPM: 5},
	"medium": {CPUs: "4", RAM: "8Gi", Storage: "8Gi", CreditsPM: 10},
	"large":  {CPUs: "8", RAM: "16Gi", Storage: "16Gi", CreditsPM: 30},
	"xlarge": {CPUs: "16", RAM: "32Gi", Storage: "32Gi", CreditsPM: 100},
}

// TaskTierConfigMapping is used for License mapping
var TaskTierConfigMapping = map[Tier]TierConfig{
	"internal": {CPUs: "0.5", RAM: "256Mi", Storage: "256Mi", CreditsPM: 0},
	"xsmall":   {CPUs: "1", RAM: "2Gi", Storage: "2Gi", CreditsPM: 2},
	"small":    {CPUs: "2", RAM: "4Gi", Storage: "4Gi", CreditsPM: 5},
	"medium":   {CPUs: "4", RAM: "8Gi", Storage: "8Gi", CreditsPM: 10},
	"large":    {CPUs: "8", RAM: "16Gi", Storage: "16Gi", CreditsPM: 30},
	"xlarge":   {CPUs: "16", RAM: "32Gi", Storage: "32Gi", CreditsPM: 100},
}

func (tc TierConfig) String() string {
	return fmt.Sprintf("%v vCPUs %v RAM", tc.CPUs, tc.RAM)
}

func (t1 Tier) IsSmallerThan(t2 Tier) bool {
	return tierEnumMapping[t1] < tierEnumMapping[t2]
}
