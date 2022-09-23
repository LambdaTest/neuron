package coverage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/synapse"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

const (
	maxRetries = 3
	delay      = 250 * time.Millisecond
	maxJitter  = 100 * time.Millisecond
	errMsg     = "failed to perform coverage transaction"
)

type manager struct {
	db                core.DB
	internalJWT       core.Session
	runner            core.K8sRunner
	bcStatusStore     core.BuildCoverageStatusStore
	testCoverageStore core.TestCoverageStore
	taskStore         core.TaskStore
	buildStore        core.BuildStore
	orgStore          core.OrganizationStore
	repoStore         core.RepoStore
	taskRunner        core.TaskRunner
	logger            lumber.Logger
}

// New returns a new CoverageService.
func New(
	cfg *config.Config,
	internalJWT core.Session,
	db core.DB,
	runner core.K8sRunner,
	bcStatusStore core.BuildCoverageStatusStore,
	testCoverageStore core.TestCoverageStore,
	taskStore core.TaskStore,
	buildStore core.BuildStore,
	orgStore core.OrganizationStore,
	repoStore core.RepoStore,
	taskRunner core.TaskRunner,
	logger lumber.Logger) core.CoverageManager {
	return &manager{
		db:                db,
		internalJWT:       internalJWT,
		runner:            runner,
		bcStatusStore:     bcStatusStore,
		testCoverageStore: testCoverageStore,
		taskStore:         taskStore,
		buildStore:        buildStore,
		orgStore:          orgStore,
		repoStore:         repoStore,
		logger:            logger,
		taskRunner:        taskRunner,
	}
}

func (m *manager) InsertCoverageData(ctx context.Context, data []*core.TestCoverage) error {
	repoID := data[0].RepoID
	commitIDs := make([]string, 0, len(data))
	for _, item := range data {
		commitIDs = append(commitIDs, item.CommitID)
	}
	if err := m.db.ExecuteTransactionWithRetry(ctx, maxRetries, delay, maxJitter, errMsg, func(tx *sqlx.Tx) error {
		if err := m.testCoverageStore.CreateInTx(ctx, tx, data); err != nil {
			m.logger.Errorf("error while creating test coverage data %v", err)
			return err
		}
		if err := m.bcStatusStore.UpdateBulkCoverageAvailableTrueInTx(ctx, tx, repoID, commitIDs); err != nil {
			m.logger.Errorf("error while setting coverageAvailable in bulk %v", err)
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	go m.runDependentCoverageJobs(repoID, commitIDs)
	return nil
}

func (m *manager) IsBaseCommitCoverageAvailable(ctx context.Context, baseCommit string, repo *core.Repository) (bool, error) {
	if !repo.CollectCoverage {
		// if coverage is not to be collected for a repo, set parentCommitCoverageExists if baseCommit is defined
		return baseCommit != "", nil
	}
	coverageAvailable, coverageRecoverable, taskCount, err := m.bcStatusStore.GetCoverageStatusCounts(ctx, baseCommit, repo.ID)
	if err != nil {
		m.logger.Errorf("Error in getting coverage status counts for base-commit %v, repoID %v", baseCommit, repo.ID)
		return false, err
	}

	if coverageAvailable == nil {
		// no rows present; either first commit, or base commit is still building
		if taskCount > 0 {
			// base commit is still building, therefore soft guarantee that coverage will eventually be available
			return true, nil
		}
		// else first commit, so base-commit's coverage is unavailable
		return false, nil
	}
	if *coverageAvailable > 0 {
		return true, nil
	}
	if *coverageRecoverable > 0 {
		// base commit coverage is not available yet, it is recoverable
		return true, nil
	}
	return false, nil
}

func (m *manager) CanRunCoverageImmediately(ctx context.Context, baseCommit, repoID string) (bool, error) {
	_, err := m.testCoverageStore.FindCommitCoverage(ctx, repoID, baseCommit)
	if err != nil {
		if errors.Is(err, errs.ErrRowsNotFound) {
			m.logger.Debugf("coverage not found for base commitID: %v, repoID: %v.", baseCommit, repoID)
			// Checking if baseCommit was ever built on TAS. If not, current commit would have produced complete coverage
			hasTasks, errC := m.buildStore.CheckIfCommitIDBuilt(ctx, baseCommit, repoID)
			if errC != nil {
				return false, errC
			}
			if !hasTasks {
				// baseCommit wasn't ever built on TAS, so current commit will have complete coverage info
				return true, nil
			}
			return false, nil
		}
		m.logger.Errorf("error checking if base-commit's %s, repo %s coverage is recorded or not. error: %v", baseCommit, repoID, err)
		return false, err
	}
	return true, nil
}

func (m *manager) RunCoverageJob(ctx context.Context, input *core.CoverageInput) error {
	m.logger.Infof("Got coverage payload Address: %s for buildID %s", input.PayloadAddress, input.BuildID)
	cmdFlags := []string{"--payloadAddress", input.PayloadAddress, "--coverage"}

	if viper.GetString("env") == constants.Dev {
		cmdFlags = append(cmdFlags, "--env", constants.Dev)
	}

	repo, err := m.repoStore.FindByBuildID(ctx, input.BuildID)
	if err != nil {
		m.logger.Errorf("Unable to find repo for  build id %s", input.BuildID)
		return err
	}
	labels := utils.CreateRunnerLabels(input.BuildID, input.BuildID, input.BuildID, synapse.CoverageMode, repo.Name)

	token, err := m.internalJWT.CreateTokenInternal(&core.BuildData{
		BuildID: input.BuildID,
		OrgID:   input.OrgID,
		RepoID:  repo.ID,
	})
	if err != nil {
		m.logger.Errorf("error generating internal JWT token: %v", err)
		return errs.ErrInvalidJWTToken
	}

	envList := []string{
		"BUILD_ID=" + input.BuildID,
		"REPO_ID=" + repo.ID,
		"ORG_ID=" + input.OrgID,
		"TOKEN=" + token,
	}

	runnerOptions := &core.RunnerOptions{
		Label:                     labels,
		NameSpace:                 utils.GetRunnerNamespaceFromOrgID(input.OrgID),
		PodName:                   fmt.Sprintf("nucleus-coverage-%s", input.BuildID),
		ContainerPort:             constants.DefaultRunnerPort,
		ContainerName:             input.BuildID,
		ContainerArgs:             cmdFlags,
		PersistentVolumeClaimName: utils.GetBuildHashKey(input.BuildID),
		OrgID:                     input.OrgID,
		PodType:                   core.CoveragePod,
		Tier:                      core.Internal,
		Env:                       envList,
		LogfilePath:               fmt.Sprintf("%s/%s/internal/coverage.log", input.OrgID, input.BuildID),
	}

	runnerErr := <-m.taskRunner.ScheduleTask(ctx, runnerOptions, input.BuildID, "", "", "")
	if runnerErr != nil {
		m.logger.Errorf("error while spawning coverage pod for buildID %s, orgID %s, error: %v",
			input.BuildID, input.OrgID, runnerErr)
	}
	if err := m.runner.Cleanup(context.TODO(), runnerOptions); err != nil {
		m.logger.Errorf("failed to delete pvc in k8s for buildID %s, orgID %s, error %v",
			input.BuildID, input.OrgID, err)
	}
	return runnerErr
}

func (m *manager) RunPendingCoverageJobs(ctx context.Context) error {
	startTime := time.Now()
	// orgID sorted inputs
	inputs, err := m.bcStatusStore.FindPendingCoverageJobsToRun(ctx)
	if err != nil {
		m.logger.Errorf("error while finding pending coverage jobs to run: %v", err)
		return err
	}
	if len(inputs) == 0 {
		return nil
	}
	currOrgID := inputs[0].OrgID
	currOrgIDJobs := make([]*core.CoverageInput, 0)
	wg := &sync.WaitGroup{}
	for _, coverageInput := range inputs {
		if coverageInput.OrgID != currOrgID {
			wg.Add(1)
			go m.runCoverageJobs(ctx, wg, currOrgIDJobs)
			currOrgIDJobs = make([]*core.CoverageInput, 0)
		}
		currOrgIDJobs = append(currOrgIDJobs, coverageInput)
	}
	wg.Add(1)
	go m.runCoverageJobs(ctx, wg, currOrgIDJobs)
	wg.Wait()
	m.logger.Infof("Ran all pending jobs in %v", time.Since(startTime))
	return nil
}

func (m *manager) runCoverageJobs(ctx context.Context, wg *sync.WaitGroup, coverageJobs []*core.CoverageInput) {
	defer wg.Done()
	for _, job := range coverageJobs {
		// not running each item in go-routine because it may clog the resources for org
		if err := m.RunCoverageJob(ctx, job); err != nil {
			m.logger.Errorf("error running coverage job %+v, error: %v", job, err)
		}
	}
}

// Trigger coverage jobs of dependent builds that have already completed their build process before this build.
// This will only trigger coverage jobs for builds that have coverage_available = false and coverage_recoverable = true
func (m *manager) runDependentCoverageJobs(repoID string, commitIDs []string) {
	ctx := context.Background()
	coverageJobs, err := m.bcStatusStore.FindDependentCoverageJobsToRun(ctx, repoID, commitIDs)
	if err != nil {
		m.logger.Errorf("error in finding dependent coverage jobs, error: %v, repoID %s", err, repoID)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	m.runCoverageJobs(ctx, wg, coverageJobs)
	wg.Wait()
	m.logger.Debugf("Ran all dependent coverage jobs for commitIDs: %v, repoID %s", commitIDs, repoID)
}
