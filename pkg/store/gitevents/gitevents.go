package gitevents

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

type giteventStore struct {
	db     core.DB
	logger lumber.Logger
}

// New returns a new giteventStore
func New(db core.DB, logger lumber.Logger) core.GitEventStore {
	return &giteventStore{db: db, logger: logger}
}

func (g *giteventStore) CreateInTx(ctx context.Context, tx *sqlx.Tx, event *core.GitEvent) error {
	if _, err := tx.NamedExecContext(ctx, insertEvent, event); err != nil {
		return err
	}
	return nil
}

func (g *giteventStore) FindByBuildID(ctx context.Context, buildID string) (*core.GitEvent, error) {
	gitEvent := &core.GitEvent{}
	return gitEvent, g.db.Execute(func(db *sqlx.DB) error {
		row := db.QueryRowxContext(ctx, findGitEventQuery, buildID)
		if err := row.StructScan(gitEvent); err != nil {
			return err
		}
		return nil
	})
}

const insertEvent = `INSERT
INTO
git_events(id,
repo_id,
event_name,
commit_id,
git_provider_handle,
event_payload)
VALUES (:id,
:repo_id,
:event_name,
:commit_id,
:git_provider_handle,
:event_payload)`

const findGitEventQuery = `
SELECT
	g.id,
	g.repo_id,
	g.event_name,
	g.commit_id,
	g.git_provider_handle,
	g.event_payload
FROM
	git_events g
JOIN build b ON
	g.id = b.event_id
WHERE
	b.id = ?
LIMIT 1`
