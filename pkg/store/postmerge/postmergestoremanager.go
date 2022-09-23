package postmerge

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
)

type postMergeStoreManager struct {
	db                               core.DB
	postMergeConfigStore             core.PostMergeConfigStore
	commitCntSinceLastPostMergeStore core.CommitCntSinceLastPostMergeStore
	logger                           lumber.Logger
}

// NewPostMergeStoreManager returns a new NewPostMergeStoreManager.
func NewPostMergeStoreManager(db core.DB,
	postMergeConfigStore core.PostMergeConfigStore,
	commitCntSinceLastPostMergeStore core.CommitCntSinceLastPostMergeStore,
	logger lumber.Logger) core.PostMergeStoreManager {
	return &postMergeStoreManager{db: db,
		postMergeConfigStore:             postMergeConfigStore,
		commitCntSinceLastPostMergeStore: commitCntSinceLastPostMergeStore,
		logger:                           logger}
}

func (manager *postMergeStoreManager) FindAndUpdateInTx(ctx context.Context, tx *sqlx.Tx, repoID, branch string,
	totalCommits int) (postMergeConfig *core.PostMergeConfig, thresholdMet bool, err error) {
	thresholdMet = false
	now := time.Now()
	postMergeConfig, err = manager.postMergeConfigStore.FindActivePostMergeConfigInTx(ctx, tx, repoID, branch)
	if err != nil {
		manager.logger.Errorf("failed in finding postMerge config for repoID: %s branch: %s error: %v", repoID, branch, err)
		return nil, false, err
	}
	var commitCntSinceLastPostMerge *core.CommitCntSinceLastPostMerge
	commitCntSinceLastPostMerge, err = manager.commitCntSinceLastPostMergeStore.FindInTx(ctx, tx, repoID, branch)
	if commitCntSinceLastPostMerge != nil {
		commitCntSinceLastPostMerge.CommitCnt += totalCommits
	}
	if err != nil {
		if !errors.Is(err, errs.ErrRowsNotFound) {
			manager.logger.Errorf("failed to find commit count for repoID: %s branch: %s error: %v", repoID, branch, err)
			return nil, false, err
		}
		commitCntSinceLastPostMerge = &core.CommitCntSinceLastPostMerge{
			ID:        utils.GenerateUUID(),
			RepoID:    postMergeConfig.RepoID,
			Branch:    branch,
			CommitCnt: totalCommits,
			CreatedAt: now,
			UpdatedAt: now}
	}
	thresholdMet, err = manager.hasThresholdMet(postMergeConfig, commitCntSinceLastPostMerge.CommitCnt)
	if err != nil {
		manager.logger.Errorf("failed to check postMerge strategy :%v", err)
		return nil, false, err
	}
	// resetting commit count to zero
	if thresholdMet {
		commitCntSinceLastPostMerge.CommitCnt = 0
	}
	err = manager.commitCntSinceLastPostMergeStore.CreateInTx(ctx, tx, commitCntSinceLastPostMerge)
	if err != nil {
		manager.logger.Errorf("failed to update commit count for repoID: %s branch: %s error: %v", repoID, branch, err)
		return nil, false, err
	}
	return postMergeConfig, thresholdMet, nil
}

func (manager *postMergeStoreManager) hasThresholdMet(postMergeConfig *core.PostMergeConfig,
	commitCnt int) (bool, error) {
	switch postMergeConfig.StrategyName {
	case core.AfterNCommitStrategy:
		threshold, err := strconv.ParseInt(postMergeConfig.Threshold, constants.Base10, constants.BitSize32)
		if err != nil {
			return false, err
		}
		manager.logger.Debugf("commit strategy threshold: %d commitCountSinceLastPostMerge:%d", threshold, commitCnt)
		return commitCnt >= int(threshold), nil
	default:
		return false, errs.ErrUnSupportedPostMergeStrategy
	}
}
