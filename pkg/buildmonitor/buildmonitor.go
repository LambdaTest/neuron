// Package buildmonitor keeps track of all tasks within a build
package buildmonitor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LambdaTest/neuron/config"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/drone/go-scm/scm"
	"gopkg.in/guregu/null.v4/zero"
)

// buildMonitor tracks progress of each task in a build
type buildMonitor struct {
	logger                   lumber.Logger
	buildStore               core.BuildStore
	bcStatusStore            core.BuildCoverageStatusStore
	ppQueueProducer          core.QueueProducer
	gitStatusService         core.GitStatusService
	flakyConfigStore         core.FlakyConfigStore
	flakyTaskBuilder         core.FlakyTaskBuilder
	repoStore                core.RepoStore
	emailNotificationmanager core.EmailNotificationManager
	testExecutionStore       core.TestExecutionStore
}

// New returns new build monitoring service
func New(cfg *config.Config,
	buildStore core.BuildStore,
	bcStatusStore core.BuildCoverageStatusStore,
	ppQueueProducer core.QueueProducer,
	gitStatusService core.GitStatusService,
	flakyConfigStore core.FlakyConfigStore,
	flakyTaskBuilder core.FlakyTaskBuilder,
	repoStore core.RepoStore,
	emailNotificationManager core.EmailNotificationManager,
	testExecutionStore core.TestExecutionStore,
	logger lumber.Logger) core.BuildMonitor {
	return &buildMonitor{
		buildStore:               buildStore,
		bcStatusStore:            bcStatusStore,
		repoStore:                repoStore,
		emailNotificationmanager: emailNotificationManager,
		ppQueueProducer:          ppQueueProducer,
		gitStatusService:         gitStatusService,
		logger:                   logger,
		flakyConfigStore:         flakyConfigStore,
		flakyTaskBuilder:         flakyTaskBuilder,
		testExecutionStore:       testExecutionStore,
	}
}

func (m *buildMonitor) FindAndUpdate(ctx context.Context, orgID string, task *core.Task) error {
	buildCache, err := m.buildStore.GetBuildCache(ctx, task.BuildID)
	if err != nil {
		m.logger.Errorf("failed to find build details in redis for taskID %s, buildID %s, orgID %s, error: %v",
			task.ID, task.BuildID, orgID, err)
		return err
	}
	// when discovery starts the build only then build is marked as running
	if task.Status == core.TaskRunning && task.Type == core.DiscoveryTask {
		build := &core.Build{
			ID:        task.BuildID,
			StartTime: task.StartTime,
			Updated:   time.Now(),
			Status:    core.BuildRunning}

		if err := m.buildStore.MarkStarted(ctx, build); err != nil {
			m.logger.Errorf("failed to mark buildID %s as started, orgID %s, error %v", task.BuildID, orgID, err)
			return err
		}

		// update label as running
		go m.UpdateGitStatus(buildCache, task, orgID, build.Status)

		return nil
	}

	// if task isn't completed, return. else, mark build as completed
	if !utils.TaskFinished(task.Status) {
		return nil
	}
	return m.stopBuild(ctx, orgID, task, buildCache)
}
func (m *buildMonitor) stopBuild(ctx context.Context, orgID string, task *core.Task, buildCache *core.BuildCache) error {
	buildStatus, remark := m.getBuildStatusAndRemark(task, buildCache)
	if buildStatus == "" {
		return nil
	}
	now := time.Now()
	build := &core.Build{
		ID:      task.BuildID,
		Status:  buildStatus,
		EndTime: task.EndTime,
		Remark:  zero.StringFrom(remark),
		Updated: now,
	}
	// check and create flaky only after execution tasks are completed and all are passed
	if task.Type == core.ExecutionTask && build.Status == core.BuildPassed {
		flakyTaskRun, err := m.createFlakyJob(ctx, buildCache, task, orgID)
		// don't update build status if flaky task successfully created
		if err == nil && flakyTaskRun {
			return nil
		}
		if err != nil {
			m.logger.Errorf("failed to create flaky task for buildID %s, orgID %s, error %v", build.ID, orgID, err)
			build.Status = core.BuildError
			build.Remark = zero.StringFrom(errs.GenericErrorMessage.Error())
		}
	}
	m.logger.Debugf("marking buildID %s as stopped with status %s, orgID %s", build.ID, build.Status, orgID)
	go m.UpdateGitStatus(buildCache, task, orgID, build.Status)
	// mark build as stopped and delete cache from redis.
	if err := m.buildStore.MarkStopped(ctx, build, orgID); err != nil {
		m.logger.Errorf("failed to mark buildID %s as completed, orgID %s, error %v", build.ID, orgID, err)
		return err
	}

	go m.UpdateTestsRuntimes(context.Background(), build.ID, buildCache.RepoID, buildCache.TargetCommitID)
	go func() {
		err := m.emailNotification(context.Background(), buildCache, task, buildStatus)
		if err != nil {
			m.logger.Errorf("error while trying to send email: %v", err)
		}
	}()

	// if collect coverage enabled, then only start coverage job.
	if buildCache.CollectCoverage {
		bcStatus := &core.BuildCoverageStatus{
			BuildID:             build.ID,
			BaseCommitID:        buildCache.BaseCommitID,
			CoverageRecoverable: buildCache.ExecTasksError == 0 && buildCache.ExecTasksAborted == 0,
			Created:             now,
			Updated:             now,
		}
		if err := m.bcStatusStore.Create(ctx, bcStatus); err != nil {
			// We don't want to fail the build-marking in DB (steps being done after this) due to coverage.
			// Right now, a build is not failed/errored due to issues in coverage.
			m.logger.Errorf("failed to write commit coverage status data: %+v, buildID %s, orgID %s, error %v",
				*bcStatus, build.ID, orgID, err)
		}
		// if no tasks with status error and aborted then start coverage job.
		if bcStatus.CoverageRecoverable {
			go func() {
				payload := &core.PostProcessingQueuePayload{
					OrgID:          orgID,
					BuildID:        build.ID,
					BaseCommitID:   buildCache.BaseCommitID,
					RepoID:         buildCache.RepoID,
					PayloadAddress: buildCache.PayloadAddress,
				}
				m.logger.Debugf("pushing coverage job for buildID %s, orgID %s", payload.BuildID, payload.OrgID)
				// insert post processing queue if build is succussfully completed.
				if err := m.ppQueueProducer.Enqueue(payload); err != nil {
					m.logger.Errorf("failed to enqueue in post processing queue, orgID %s, buildID %s, error: %v",
						payload.OrgID, payload.BuildID, err)
				}
			}()
		}
	}
	return nil
}

func (m *buildMonitor) createFlakyJob(ctx context.Context, buildCache *core.BuildCache, task *core.Task, orgID string) (bool, error) {
	flakyConfigType := core.FlakyConfigPostMerge
	if buildCache.BuildTag == core.PreMergeTag {
		flakyConfigType = core.FlakyConfigPreMerge
	}
	flakyConfig, err := m.flakyConfigStore.FindActiveFlakyConfig(ctx, buildCache.RepoID, buildCache.Branch, flakyConfigType)
	if err != nil {
		if errors.Is(err, errs.ErrRowsNotFound) {
			m.logger.Debugf("no flaky config found for repoID %s, branch %s, type %s. buildID %s orgID %s",
				buildCache.RepoID, buildCache.Branch, flakyConfigType, task.BuildID, orgID)
			return false, nil
		}
		m.logger.Errorf("error while finding flaky config for repoID %s, branch %s, type %s. buildID %s orgID %s, error: %v",
			buildCache.RepoID, buildCache.Branch, flakyConfigType, task.BuildID, orgID, err)
		return false, err
	}
	m.logger.Debugf("flaky config found for repoID %s, buildID %s, orgID %s , config %+v",
		buildCache.RepoID, task.BuildID, orgID, *flakyConfig)
	if err := m.flakyTaskBuilder.CreateFlakyTask(ctx, buildCache, task, flakyConfig, orgID); err != nil {
		return false, err
	}
	return true, nil
}

func (m *buildMonitor) getBuildStatusAndRemark(task *core.Task, buildCache *core.BuildCache) (buildStatus core.BuildStatus, remark string) {
	if task.Type == core.DiscoveryTask {
		// if discovery task is passed and it impacted tests exist then we skip marking build as stopped
		if task.Status == core.TaskPassed && buildCache.ImpactedTestExists {
			return "", ""
		}
		buildStatus = core.BuildStatus(task.Status)
		remark = task.Remark.String
	} else if task.Type == core.FlakyTask {
		// if all flaky task is not completed then return
		if !m.allFlakyTasksCompleted(buildCache) {
			return "", ""
		}

		buildStatus, remark = m.winnerStatusAndRemark(buildCache)
	} else {
		// if all execution tasks aren't completed, return.
		if !m.allExecutionTasksCompleted(buildCache) {
			return "", ""
		}
		// check if  discovery has completed for all submodule
		if !m.allSubmoduleProcessed(buildCache) {
			return "", ""
		}
		// find the winner status
		buildStatus, remark = m.winnerStatusAndRemark(buildCache)
	}
	return buildStatus, remark
}

func (m *buildMonitor) allSubmoduleProcessed(buildCache *core.BuildCache) bool {
	return buildCache.TotalSubModule == buildCache.ProcessedSubModule
}

func (m *buildMonitor) allExecutionTasksCompleted(buildCache *core.BuildCache) bool {
	execCount := buildCache.ExecTasksPassed + buildCache.ExecTasksFailed + buildCache.ExecTasksAborted + buildCache.ExecTasksError
	return buildCache.TotalExecTasks <= execCount
}

func (m *buildMonitor) allFlakyTasksCompleted(buildCache *core.BuildCache) bool {
	flakyCount := buildCache.FlakyTasksPassed + buildCache.FlakyTasksFailed + buildCache.FlakyTasksAborted + buildCache.FlakyTasksError
	return buildCache.TotalFlakyTasks == flakyCount
}

func getRemark(count, totalCount int, status, taskType string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s task %s out of %d", count, taskType, status, totalCount)
	}
	return fmt.Sprintf("%d %s tasks %s out of %d", count, taskType, status, totalCount)
}

func (m *buildMonitor) winnerStatusAndRemark(buildCache *core.BuildCache) (status core.BuildStatus, remark string) {
	if buildCache.FlakyTasksAborted >= 1 {
		return core.BuildAborted, getRemark(buildCache.FlakyTasksAborted, buildCache.TotalFlakyTasks, "aborted", "flaky")
	}
	if buildCache.FlakyTasksError >= 1 {
		errorRemark := getRemark(buildCache.FlakyTasksError, buildCache.TotalFlakyTasks, "errored", "flaky")
		if buildCache.BuildFailedRemarks != "" {
			errorRemark = fmt.Sprintf("%s , reason: %s", errorRemark, buildCache.BuildFailedRemarks)
		}
		return core.BuildError, errorRemark
	}
	if buildCache.FlakyTestCount > 0 {
		// If buildCache.FlakyTestCount > 0 then it is guarantee that atleast one FTM task has failed
		remark := getRemark(buildCache.FlakyTasksFailed, buildCache.TotalFlakyTasks, "failed", "flaky")
		remark = fmt.Sprintf("%s. %d test case(s) were flaky", remark, buildCache.FlakyTestCount)
		return core.BuildFailed, remark
	}

	if buildCache.ExecTasksAborted >= 1 {
		if buildCache.ExecTasksAborted == 1 {
			return core.BuildAborted, fmt.Sprintf("%d execution task has aborted out of %d", buildCache.ExecTasksAborted, buildCache.TotalExecTasks)
		}
		return core.BuildAborted, fmt.Sprintf("%d execution tasks have aborted out of %d", buildCache.ExecTasksAborted, buildCache.TotalExecTasks)
	}

	if buildCache.ExecTasksError >= 1 {
		errorRemark := fmt.Sprintf("%d execution tasks have errored out of %d", buildCache.ExecTasksError, buildCache.TotalExecTasks)
		if buildCache.ExecTasksError == 1 {
			errorRemark = fmt.Sprintf("%d execution task has errored out of %d", buildCache.ExecTasksError, buildCache.TotalExecTasks)
		}

		if buildCache.BuildFailedRemarks != "" {
			errorRemark = fmt.Sprintf("%s , reason: %s", errorRemark, buildCache.BuildFailedRemarks)
		}
		return core.BuildError, errorRemark
	}

	if buildCache.ExecTasksFailed >= 1 {
		failedRemark := ""
		if buildCache.FailedTestCount > 1 {
			failedRemark += fmt.Sprintf(". Also, %d test cases have failed.", buildCache.FailedTestCount)
		} else if buildCache.FailedTestCount > 0 {
			failedRemark += fmt.Sprintf(". Also, %d test case has failed.", buildCache.FailedTestCount)
		}

		if buildCache.ExecTasksFailed == 1 {
			return core.BuildFailed,
				fmt.Sprintf("%d execution task has failed out of %d%s", buildCache.ExecTasksFailed, buildCache.TotalExecTasks, failedRemark)
		}
		return core.BuildFailed,
			fmt.Sprintf("%d execution tasks have failed out of %d%s", buildCache.ExecTasksFailed, buildCache.TotalExecTasks, failedRemark)
	}
	return core.BuildPassed, ""
}

func (m *buildMonitor) UpdateGitStatus(build *core.BuildCache, task *core.Task, orgID string, buildStatus core.BuildStatus) {
	if err := m.gitStatusService.UpdateGitStatus(context.TODO(),
		build.GitProvider,
		build.RepoSlug,
		task.BuildID,
		build.TargetCommitID,
		build.TokenPath,
		build.InstallationTokenPath,
		buildStatus,
		build.BuildTag); err != nil {
		m.logger.Errorf("error while updating git status, taskID  %s, commitID %s, orgID %s, repoID %s, buildID %s, error %v",
			task.ID, orgID, build.TargetCommitID, build.RepoID, task.BuildID, err)
	}
}

// send build notification if build has passed or failed and it is the first build for the repo
func (m *buildMonitor) emailNotification(ctx context.Context,
	build *core.BuildCache,
	task *core.Task,
	buildStatus core.BuildStatus) error {
	if buildStatus != core.BuildPassed && buildStatus != core.BuildFailed {
		return nil
	}
	count, err := m.buildStore.CountPassedFailedBuildsForRepo(ctx, task.RepoID)
	if err != nil {
		m.logger.Errorf("error while finding count of passed & failed builds: %v, aborting to send build notification email",
			err)
		return err
	}
	// send email iff it is the first build for the repo
	if count != 1 {
		return nil
	}

	repoSlug := build.RepoSlug
	buildID := task.BuildID
	gitProvider := build.GitProvider
	orgName, repoName := scm.Split(repoSlug)
	statusData, err := m.repoStore.FindBadgeData(ctx, repoName, orgName, "", buildID, gitProvider.String())
	if err != nil {
		m.logger.Errorf("failed to find test status data for buildID: %s, error: %v, aborting to send build notification email",
			buildID, err)
		return err
	}

	passedBuild := (buildStatus == core.BuildPassed)
	totalTests := strconv.Itoa(statusData.TotalTests)
	if errS := m.emailNotificationmanager.SendBuildStatus(ctx, orgName, buildID, gitProvider.String(),
		repoName, totalTests, passedBuild); errS != nil {
		m.logger.Errorf("failed to send first successful build notifiaction email for repo: %s, error: %v",
			repoName, errS)
		return errS
	}
	return nil
}

func (m *buildMonitor) UpdateTestsRuntimes(ctx context.Context, buildID, repoID, commitID string) {
	timeSavedData, err := m.FindTimeSavedData(ctx, buildID, commitID, repoID)
	if err != nil {
		m.logger.Errorf("error in finding time saved data for buildID: %s, commitID: %s, repoID: %s, error: %v", buildID, commitID, repoID, err)
		return
	}

	build := &core.Build{
		ID:                buildID,
		TimeAllTests:      timeSavedData.TimeTakenByAllTests,
		TimeImpactedTests: timeSavedData.TimeTakenByImpactedTests,
		Updated:           time.Now(),
	}
	if errU := m.buildStore.UpdateTestsRuntimes(ctx, build); errU != nil {
		m.logger.Errorf("failed to update data in data store for buildID: %s, error: %v", buildID, errU)
		return
	}
}
