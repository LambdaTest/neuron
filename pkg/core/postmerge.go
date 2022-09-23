package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// PostMergeStrategyName is a strategy name
type PostMergeStrategyName string

const (
	// AfterNCommitStrategy is commit strategy name
	AfterNCommitStrategy PostMergeStrategyName = "after_n_commits"
)

// PostMergeConfig defines postmerge config
type PostMergeConfig struct {
	ID           string                `db:"id" json:"id"`
	Repo         string                `json:"repo"`
	Org          string                `json:"org"`
	RepoID       string                `db:"repo_id" json:"-"`
	Branch       string                `db:"branch" json:"branch"`
	IsActive     bool                  `db:"is_active" json:"is_active"`
	StrategyName PostMergeStrategyName `db:"strategy_name" json:"strategy_name"`
	Threshold    string                `db:"threshold" json:"threshold"`
	CreatedAt    time.Time             `db:"created_at" json:"-"`
	UpdatedAt    time.Time             `db:"updated_at" json:"-"`
}

// CommitCntSinceLastPostMerge define commitcnt since last postmerge build
type CommitCntSinceLastPostMerge struct {
	ID        string    `db:"id" json:"id"`
	RepoID    string    `db:"repo_id" json:"repo_id"`
	Branch    string    `db:"branch" json:"branch"`
	CommitCnt int       `db:"commit_cnt" json:"commit_cnt"`
	CreatedAt time.Time `db:"created_at" json:"-"`
	UpdatedAt time.Time `db:"updated_at" json:"-"`
}

// PostMergeConfigStore has datastore operations contracts for postmergeconfig table
type PostMergeConfigStore interface {
	// Create persists a new postmergeconfig in the datastore
	Create(ctx context.Context, postMergeConfig *PostMergeConfig) error
	// Update updates existing postmergeconfig in the datastore
	Update(ctx context.Context, postMergeConfig *PostMergeConfig) error
	// FindActivePostMergeConfigInTx returns active postmergeconfig by repoid and branch name within specified transaction
	FindActivePostMergeConfigInTx(ctx context.Context, tx *sqlx.Tx, repoID string,
		branchName string) (postMergeConfig *PostMergeConfig, err error)
	// FindAllPostMergeConfig returns all postmergeconfig by repoID
	FindAllPostMergeConfig(ctx context.Context, repoID string) (postMergeConfigList []*PostMergeConfig, sqlErr error)
}

// CommitCntSinceLastPostMergeStore defines operations to store commitCnt by repoID and branch
type CommitCntSinceLastPostMergeStore interface {
	// CreateInTx  creates commitCntFromLastPostMerge in the datastore within specified transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, commitCnt *CommitCntSinceLastPostMerge) error
	// FindInTx returns  commitCnt by repoid and branch name within specified transaction
	FindInTx(ctx context.Context, tx *sqlx.Tx, repoID string, branchName string) (*CommitCntSinceLastPostMerge, error)
}

// PostMergeStoreManager defines operations on postmergeconfig and commitCnt
type PostMergeStoreManager interface {
	// FindAndUpdateInTx finds postmerge config in db and  checks strategy threshold
	// and update commit cnt in commitcnt store and runs the queries in same transaction
	FindAndUpdateInTx(ctx context.Context, tx *sqlx.Tx, repoID, branch string,
		commitCnt int) (postMergeConfig *PostMergeConfig, thresholdMet bool, err error)
}
