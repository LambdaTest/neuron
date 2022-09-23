package flakytaskbuilder

import (
	"context"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
)

type flakyTask struct {
	logger             lumber.Logger
	testexecutionStore core.TestExecutionStore
	taskBuilder        core.TaskBuilder
	buildStore         core.BuildStore
	flakyConfigStore   core.FlakyConfigStore
}

func New(testexecutionStore core.TestExecutionStore,
	taskBuilder core.TaskBuilder,
	buildStore core.BuildStore,
	flakyConfigStore core.FlakyConfigStore,
	logger lumber.Logger) core.FlakyTaskBuilder {
	return &flakyTask{logger: logger,
		testexecutionStore: testexecutionStore,
		taskBuilder:        taskBuilder,
		buildStore:         buildStore,
		flakyConfigStore:   flakyConfigStore}
}

// TODO: this will be modify after flaky has parallelism support
func (f *flakyTask) CreateFlakyTask(ctx context.Context, build *core.BuildCache,
	lastExecTask *core.Task, flakyConfig *core.FlakyConfig, orgID string) error {
	tasks := make([]*core.Task, 0)
	jobs := make([]*core.Job, 0)
	executedTestMap, err := f.testexecutionStore.FindImpactedTestsByBuild(ctx, lastExecTask.BuildID)
	if err != nil {
		f.logger.Errorf("error while reading impacted test for buildID %s orgID %s, error %v", lastExecTask.BuildID, orgID, err)
		return err
	}
	if err = f.buildStore.UpdateFlakyExecutionCount(ctx, lastExecTask.BuildID, flakyConfig.ConsecutiveRuns); err != nil {
		f.logger.Errorf("error while updating flaky execution count buildID %s orgID %s, error %v", lastExecTask.BuildID, orgID, err)
		return err
	}

	for subModule, executedTests := range executedTestMap {
		config := &core.InputLocatorConfig{
			Locators: make([]core.LocatorConfig, 0),
		}

		taskID := utils.GenerateUUID()
		for _, test := range executedTests {
			locatorconfig := new(core.LocatorConfig)
			locatorconfig.Locator = test.TestLocator
			config.Locators = append(config.Locators, *locatorconfig)
		}
		f.logger.Debugf("%d impacted tests found for orgID %s buildID %s, submodule %s",
			len(executedTests), orgID, lastExecTask.BuildID, subModule)
		taskObj, err := f.taskBuilder.CreateTask(ctx, taskID, lastExecTask.BuildID, orgID, build.RepoID, subModule,
			core.FlakyTask, config, lastExecTask.Tier)
		if err != nil {
			return err
		}
		jobObj := f.taskBuilder.CreateJob(taskID, orgID, core.Ready)
		if taskObj != nil && jobObj != nil {
			f.logger.Debugf("taskID %s, jobID %s will be enqueued for buildID %s orgID %s, by flaky builder",
				taskObj.ID, jobObj.ID, lastExecTask.BuildID, orgID)
			tasks = append(tasks, taskObj)
			jobs = append(jobs, jobObj)
		}
	}
	totalTask := len(tasks)
	if err := f.buildStore.IncrementCacheCount(ctx, lastExecTask.BuildID, core.TotalFlakyTasks, totalTask); err != nil {
		f.logger.Errorf("error occurred while updating %s to %d, for buildID %s, orgID %s, err %v",
			core.TotalFlakyTasks, totalTask, lastExecTask.BuildID, orgID)
		return err
	}
	return f.taskBuilder.EnqueueTaskAndJob(ctx, lastExecTask.BuildID, orgID, tasks, jobs, core.FlakyTask)
}
