package login

import (
	"context"
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform login transaction"
)

// loginStore executes all the queries during login in a transaction
type loginStore struct {
	db           core.DB
	orgStore     core.OrganizationStore
	userStore    core.GitUserStore
	userOrgStore core.UserOrgStore
	vaultStore   core.Vault
	tokenHandler core.GitTokenHandler
	licenseStore core.LicenseStore
	logger       lumber.Logger
}

// New returns a new LoginStore
func New(db core.DB,
	orgStore core.OrganizationStore,
	userStore core.GitUserStore,
	userOrgStore core.UserOrgStore,
	vaultStore core.Vault,
	tokenHandler core.GitTokenHandler,
	licenseStore core.LicenseStore,
	logger lumber.Logger,
) core.LoginStore {
	return &loginStore{
		db:           db,
		orgStore:     orgStore,
		userStore:    userStore,
		userOrgStore: userOrgStore,
		vaultStore:   vaultStore,
		tokenHandler: tokenHandler,
		licenseStore: licenseStore,
		logger:       logger,
	}
}

func (l *loginStore) Create(ctx context.Context, user *core.GitUser, userExists bool, orgs ...*core.Organization) error {
	return l.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		now := time.Now()
		if err := l.orgStore.CreateInTx(ctx, tx, orgs); err != nil {
			l.logger.Errorf("failed to create organizations in database, error: %v", err)
			return err
		}

		orgz, err := l.orgStore.FindOrgsByNameTx(ctx, tx, user.GitProvider, orgs)
		if err != nil {
			l.logger.Errorf("failed to find organizations in database, error: %v", err)
			return err
		}

		if !userExists {
			if err := l.userStore.CreateInTx(ctx, tx, user); err != nil {
				l.logger.Errorf("failed to create git user in database: %v", err)
				return err
			}
		}

		//TODO: handle case where user is no longer member of an organization
		userOrgs := make([]*core.UserOrg, 0, len(orgz))
		for _, o := range orgz {
			userOrgs = append(userOrgs, &core.UserOrg{ID: utils.GenerateUUID(), UserID: user.ID, OrgID: o.ID, Updated: now, Created: now})
		}

		if err := l.userOrgStore.CreateInTx(ctx, tx, userOrgs); err != nil {
			l.logger.Errorf("failed to create user_organization in database, error:  %v", err)
			return ctx.Err()
		}

		if err := l.licenseStore.CreateInTx(ctx, tx, orgz); err != nil {
			l.logger.Errorf("failed to create default license in database, error: %v", err)
			return err
		}

		path := l.vaultStore.GetTokenPath(user.GitProvider, user.Mask, user.ID)
		if err := l.tokenHandler.UpdateTokenInVault(path, &user.Oauth); err != nil {
			l.logger.Errorf("failed to create create secret in vault store: %v", err)
			return err
		}

		return nil
	})
}

func (l *loginStore) UpdateOrgList(ctx context.Context,
	user *core.GitUser,
	orgs []*core.Organization) (orgz []*core.Organization, err error) {
	err = l.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		now := time.Now()
		if err = l.orgStore.CreateInTx(ctx, tx, orgs); err != nil {
			l.logger.Errorf("failed to create organizations in database, error: %v", err)
			return err
		}

		orgz, err = l.orgStore.FindOrgsByNameTx(ctx, tx, user.GitProvider, orgs)
		if err != nil {
			l.logger.Errorf("failed to find organizations in database, error: %v", err)
			return err
		}

		//TODO: handle case where user is no longer member of an organization
		userOrgs := make([]*core.UserOrg, 0, len(orgz))
		for _, o := range orgz {
			userOrgs = append(userOrgs, &core.UserOrg{ID: utils.GenerateUUID(), UserID: user.ID, OrgID: o.ID, Updated: now, Created: now})
		}

		if err = l.userOrgStore.CreateInTx(ctx, tx, userOrgs); err != nil {
			l.logger.Errorf("failed to create user_organization in database, error:  %v", err)
			return ctx.Err()
		}

		if err = l.licenseStore.CreateInTx(ctx, tx, orgz); err != nil {
			l.logger.Errorf("failed to create default license in database, error: %v", err)
			return err
		}

		path := l.vaultStore.GetTokenPath(user.GitProvider, user.Mask, user.ID)
		if err = l.tokenHandler.UpdateTokenInVault(path, &user.Oauth); err != nil {
			l.logger.Errorf("failed to create create secret in vault store: %v", err)
			return err
		}

		orgz, err = l.orgStore.FindUserOrgsTx(ctx, tx, user.ID)
		if err != nil {
			l.logger.Errorf("failed to find organizations in database for user %s, error: %v", user.ID, err)
			return err
		}
		return nil
	})
	return orgz, err
}
