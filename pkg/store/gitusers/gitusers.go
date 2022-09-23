package gitusers

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type gitUserStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new GitUserStore
func New(db core.DB, logger lumber.Logger) core.GitUserStore {
	return &gitUserStore{db: db, logger: logger}
}

func (g *gitUserStore) FindByID(ctx context.Context, userID string) (*core.GitUser, error) {
	user := new(core.GitUser)
	return user, g.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectUserByIDQuery, userID)
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.GitProvider, &user.Mask); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (g *gitUserStore) Find(ctx context.Context, username string, gitProvider core.SCMDriver) (*core.GitUser, error) {
	user := new(core.GitUser)
	return user, g.db.Execute(func(db *sqlx.DB) error {
		selectQuery := "SELECT id, username, avatar, git_provider, mask FROM git_users where git_provider=? and username=?"
		rows := db.QueryRowxContext(ctx, selectQuery, gitProvider, username)

		if err := rows.StructScan(user); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (g *gitUserStore) Create(ctx context.Context, user *core.GitUser) error {
	return g.db.Execute(func(db *sqlx.DB) error {
		if _, err := db.NamedExecContext(ctx, insertQuery, user); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (g *gitUserStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, user *core.GitUser) error {
	if _, err := tx.NamedExecContext(ctx, insertQuery, user); err != nil {
		return errors.SQLError(err)
	}
	return nil
}
func (g *gitUserStore) FindByOrg(ctx context.Context, userID, namespace string) (*core.GitUser, string, error) {
	user := new(core.GitUser)
	var orgID string
	return user, orgID, g.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowContext(ctx, selectByOrgQuery, userID, namespace)
		if err := rows.Scan(&user.ID, &user.Username, &user.GitProvider, &user.Mask, &orgID); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (g *gitUserStore) FindUserTasUsageInfo(ctx context.Context, orgID, userID string) ([]*core.AuthorMeta, error) {
	authors := make([]*core.AuthorMeta, 0)
	return authors, g.db.Execute(func(db *sqlx.DB) error {
		args := map[string]interface{}{
			"org_id": orgID,
		}
		rows, err := db.NamedQueryContext(ctx, findUsageQuery, args)
		if err != nil {
			return errors.SQLError(err)
		}
		defer rows.Close()

		for rows.Next() {
			u := new(core.AuthorMeta)
			if err := rows.StructScan(u); err != nil {
				return errors.SQLError(err)
			}
			authors = append(authors, u)
		}
		return nil
	})
}

func (g *gitUserStore) FindAdminByOrgNameAndGitProvider(ctx context.Context, orgName, gitProvider string) (*core.GitUser, error) {
	user := new(core.GitUser)
	return user, g.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectAdminUserByOrgNameAndGitProvider, orgName, gitProvider)
		if err := rows.StructScan(user); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
}

func (g *gitUserStore) FindAdminByBuildID(ctx context.Context, buildID string) (*core.GitUser, error) {
	admin := new(core.GitUser)
	err := g.db.Execute(func(db *sqlx.DB) error {
		rows := db.QueryRowxContext(ctx, selectAdminUsingBuildID, buildID)
		if err := rows.StructScan(admin); err != nil {
			return errors.SQLError(err)
		}
		return nil
	})
	return admin, err
}

const selectAdminUsingBuildID = `
SELECT 
    gu.username,
    gu.email
FROM 
    git_users gu 
JOIN
    repositories r ON 
    r.admin_id = gu.id 
JOIN 
    build b ON 
    b.repo_id = r.id 
WHERE b.id = ?
`

const selectAdminUserByOrgNameAndGitProvider = `
SELECT
    gu.id,
    gu.username,
    gu.email 
FROM
    git_users gu
JOIN 
    repositories r ON 
    r.admin_id = gu.id
JOIN organizations o ON 
    o.id = r.org_id 
WHERE 
    o.name = ?
    AND o.git_provider = ?
ORDER BY gu.created_at
`

const selectByOrgQuery = `SELECT
gu.id,
gu.username,
gu.git_provider,
gu.mask,
o.id
FROM
git_users gu
JOIN user_organizations uo ON
uo.user_id = gu.id
JOIN organizations o ON
o.id = uo.org_id
WHERE
gu.id =?
AND o.name =?`

const selectUserByIDQuery = `SELECT
u.id,
u.username,
u.email,
u.git_provider,
u.mask
FROM
git_users u
WHERE
u.id =?`

const insertQuery = `INSERT
INTO
git_users(id,
git_provider,
avatar,
username,
email,
mask)
VALUES (:id,
:git_provider,
:avatar,
:username,
:email,
:mask)
`

const findUsageQuery = `
WITH build_meta AS (
	SELECT
		COUNT(DISTINCT b.id) total_builds,
		o.name org_name
	FROM
		build b
	JOIN repositories r ON
		b.repo_id = r.id
	JOIN organizations o ON
		o.id = r.org_id 
	WHERE
		r.org_id = :org_id
	GROUP BY
		r.org_id 
	),
	repos_meta AS (
	SELECT
		o.name org_name,
		COUNT(DISTINCT r.id) total_repos
	FROM 
		repositories r
	JOIN organizations o ON
		o.id = r.org_id 
	WHERE
		r.org_id = :org_id
		AND r.active = 1
	GROUP BY
		r.org_id
	),
	test_meta AS (
	SELECT
		o.name org_name,
		COUNT(DISTINCT t.id) total_tests
	FROM
		test t
	JOIN repositories r ON
		t.repo_id = r.id
	JOIN organizations o ON
		o.id = r.org_id 
	WHERE
		r.org_id = :org_id
	GROUP BY
		r.org_id 
	)
	SELECT
		COALESCE(ANY_VALUE(b.total_builds), 0) total_builds,
		ANY_VALUE(o.name) author,
		COALESCE(ANY_VALUE(r.total_repos), 0) total_repos,
		COALESCE(ANY_VALUE(t.total_tests), 0) total_tests,
		COUNT(DISTINCT gc.commit_id) total_commits
	FROM
		git_commits gc
	JOIN repositories r ON
		r.id = gc.repo_id
	JOIN organizations o ON
		o.id = :org_id
	LEFT JOIN repos_meta r ON
		r.org_name = o.name
	LEFT JOIN test_meta t ON
		t.org_name = o.name
	LEFT JOIN build_meta b ON
		b.org_name = o.name
	WHERE
		r.org_id=:org_id
	GROUP BY
		r.org_id 
`
