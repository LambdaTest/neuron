package hook

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/jmoiron/sqlx"
)

const (
	maxRetries         = 3
	delay              = 250 * time.Millisecond
	maxJitter          = 100 * time.Millisecond
	errMsg             = "failed to perform hook transaction"
	pullRequestInitial = "#"
)

// hookStore executes all the queries in a common transaction when git webhook is received
type hookStore struct {
	db                    core.DB
	commitStore           core.GitCommitStore
	eventStore            core.GitEventStore
	buildStore            core.BuildStore
	taskStore             core.TaskStore
	branchStore           core.BranchStore
	branchCommitStore     core.BranchCommitStore
	postMergeStoreManager core.PostMergeStoreManager
	logger                lumber.Logger
}

// New returns a new HookStore
func New(
	db core.DB,
	commitStore core.GitCommitStore,
	eventStore core.GitEventStore,
	taskStore core.TaskStore,
	buildStore core.BuildStore,
	branchStore core.BranchStore,
	branchCommitStore core.BranchCommitStore,
	postMergeStoreManager core.PostMergeStoreManager,
	logger lumber.Logger) core.HookStore {
	return &hookStore{
		db:                    db,
		commitStore:           commitStore,
		eventStore:            eventStore,
		taskStore:             taskStore,
		buildStore:            buildStore,
		branchStore:           branchStore,
		branchCommitStore:     branchCommitStore,
		postMergeStoreManager: postMergeStoreManager,
		logger:                logger,
	}
}

func (h *hookStore) CreateEntities(ctx context.Context, payload *core.ParserResponse) (bool, error) {
	buildID := payload.EventBlob.BuildID
	orgID := payload.Repository.OrgID
	repoID := payload.Repository.ID
	var createBuild bool
	var err error
	return createBuild, h.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			now := time.Now()
			if err = h.commitStore.CreateInTx(ctx, tx, payload.GitCommits); err != nil {
				h.logger.Errorf("error while inserting git commits commits: %#v, orgID %s, buildID %s, repoID %s, error %v,",
					payload.GitCommits, orgID, buildID, repoID, err)
				return err
			}
			gitEvent := &core.GitEvent{ID: payload.EventBlob.EventID,
				RepoID:            repoID,
				EventName:         payload.EventBlob.EventType,
				GitProviderHandle: payload.EventBlob.GitProvider,
				CommitID:          payload.EventBlob.TargetCommit,
				EventPayload:      payload.EventBlob.EventBody}
			if err = h.eventStore.CreateInTx(ctx, tx, gitEvent); err != nil {
				h.logger.Errorf("failed to insert git event, orgID %s, buildID %s, repoID %s, error: %v",
					orgID, buildID, repoID, err)
				return err
			}
			branchName := ""
			if payload.EventBlob.EventType == core.EventPullRequest {
				branchName = pullRequestInitial + strconv.Itoa(payload.EventBlob.PullRequestNumber)
			} else {
				branchName = payload.EventBlob.BranchName
			}
			branch := &core.Branch{ID: utils.GenerateUUID(),
				RepoID: repoID,
				Name:   branchName,
			}
			if err = h.branchStore.CreateInTx(ctx, tx, branch); err != nil {
				h.logger.Errorf("failed to insert git branch, orgID %s, buildID %s, repoID %s, error %v",
					orgID, buildID, repoID, err)
				return err
			}
			branchCommits := make([]*core.BranchCommit, 0, len(payload.GitCommits))
			for _, c := range payload.GitCommits {
				branchCommits = append(branchCommits, &core.BranchCommit{ID: utils.GenerateUUID(),
					BranchName: branch.Name,
					RepoID:     branch.RepoID,
					CommitID:   c.CommitID,
					Updated:    now,
					Created:    now})
			}
			if err = h.branchCommitStore.CreateInTx(ctx, tx, branchCommits); err != nil {
				h.logger.Errorf("failed to insert branch commit relationships, orgID %s, buildID %s, repoID %s, error %v",
					orgID, buildID, repoID, err)
				return err
			}
			createBuild, err = h.checkBuildCreation(ctx, tx, payload)
			return err
		})
}

// checkBuildCreation checks if build can be created
func (h *hookStore) checkBuildCreation(ctx context.Context,
	tx *sqlx.Tx,
	payload *core.ParserResponse) (bool, error) {
	// if build is rebuild it is allowed to create new build
	if payload.Rebuild {
		return true, nil
	}
	repoID := payload.Repository.ID
	orgID := payload.Repository.OrgID
	if payload.BuildTag == core.PreMergeTag {
		return strings.EqualFold(payload.EventBlob.ForkSlug, payload.EventBlob.RepoSlug), nil
	}
	passedBuildExists, err := h.buildStore.CheckIfPassedBuildExistsInTx(ctx, tx, repoID)
	if err != nil {
		h.logger.Errorf("failed to find passed build counts for repoID %s, OrgID %s, error %v",
			repoID, orgID, err)
		return false, err
	}
	// if no passed build exists, then we skip checking post merge config
	//  and create a new build
	if !passedBuildExists {
		h.logger.Debugf("no passed build exists for repoID %s, OrgID %s", repoID, orgID)
		return true, nil
	}
	_, thresholdMet, err := h.postMergeStoreManager.FindAndUpdateInTx(ctx, tx, repoID,
		payload.EventBlob.BranchName, len(payload.EventBlob.Commits))
	if err != nil {
		if !errors.Is(err, errs.ErrRowsNotFound) {
			h.logger.Errorf("failed in find and update post merge config repoID %s, orgID %s, error: %v",
				repoID, orgID, err)
			return false, err
		}
	}
	// skip build creation when postmerge threshold is not met
	if !thresholdMet {
		return false, nil
	}

	return true, nil
}

func (h *hookStore) CreateBuild(ctx context.Context, payloadAddress string, payload *core.ParserResponse) (*core.Job, error) {
	buildID := payload.EventBlob.BuildID
	orgID := payload.Repository.OrgID
	repoID := payload.Repository.ID
	var discoveryJob *core.Job
	dbErr := h.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg,
		func(tx *sqlx.Tx) error {
			discoveryTier := h.getDiscoveryTier(ctx, tx, repoID, payload.License)
			now := time.Now()
			build := &core.Build{ID: buildID,
				CommitID:       payload.EventBlob.TargetCommit,
				RepoID:         repoID,
				EventID:        payload.EventBlob.EventID,
				Status:         core.BuildInitiating,
				PayloadAddress: payloadAddress,
				BaseCommit:     payload.EventBlob.BaseCommit,
				Tag:            payload.BuildTag,
				Branch:         payload.EventBlob.BranchName}
			if err := h.buildStore.CreateInTx(ctx, tx, build); err != nil {
				h.logger.Errorf("failed to insert build, orgID %s, buildID %s, repoID %s, error: %v",
					orgID, buildID, repoID, err)
				return err
			}
			buildCache := &core.BuildCache{TokenPath: payload.TokenPath,
				InstallationTokenPath: payload.InstallationTokenPath,
				BuildTag:              payload.BuildTag,
				SecretPath:            payload.SecretPath,
				GitProvider:           payload.EventBlob.GitProvider,
				RepoSlug:              payload.EventBlob.RepoSlug,
				PullRequestNumber:     payload.EventBlob.PullRequestNumber,
				PayloadAddress:        payloadAddress,
				BaseCommitID:          payload.EventBlob.BaseCommit,
				TargetCommitID:        payload.EventBlob.TargetCommit,
				RepoID:                repoID,
				CollectCoverage:       payload.Repository.CollectCoverage,
				Tier:                  discoveryTier,
				Branch:                payload.EventBlob.BranchName,
			}
			// store build cache details in redis
			if err := h.buildStore.StoreBuildCache(ctx, build.ID, buildCache); err != nil {
				h.logger.Errorf("failed to store build details in cache orgID %s, buildID %s, repoID %s, error: %v",
					orgID, buildID, repoID, err)
				return err
			}
			task := &core.Task{
				ID:      utils.GenerateUUID(),
				RepoID:  repoID,
				BuildID: buildID,
				Status:  core.TaskInitiating,
				Created: now,
				Updated: now,
				Type:    core.DiscoveryTask,
			}
			if err := h.taskStore.CreateInTx(ctx, tx, task); err != nil {
				h.logger.Errorf("error creating task for buildID %s, orgID %s, err %v", buildID, orgID, err)
				return err
			}
			discoveryJob = &core.Job{
				ID:      utils.GenerateUUID(),
				Status:  core.Ready,
				OrgID:   orgID,
				TaskID:  task.ID,
				Created: now,
				Updated: now,
			}
			return nil
		})

	return discoveryJob, dbErr
}

func (h *hookStore) getDiscoveryTier(ctx context.Context, tx *sqlx.Tx, repoID string, license *core.License) core.Tier {
	lastBuildTier, err := h.buildStore.FindLastBuildTierInTx(ctx, tx, repoID)
	if err != nil {
		h.logger.Errorf("failed to find tier used in last build, repoID %s, error: %v", repoID, err)
		lastBuildTier = core.Small
	}
	if license.Tier.IsSmallerThan(lastBuildTier) {
		return license.Tier
	}

	return lastBuildTier
}
