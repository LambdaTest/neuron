package testsplitter

import (
	"container/heap"
	"context"
	"strings"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/taskheap"
	"github.com/LambdaTest/neuron/pkg/utils"
)

type testSplitter struct {
	logger             lumber.Logger
	taskStore          core.TaskStore
	taskQueueManager   core.TaskQueueManager
	buildStore         core.BuildStore
	testexecutionStore core.TestExecutionStore
	azureClient        core.AzureBlob
	taskBuilder        core.TaskBuilder
}

// NewTestSplitter new returns a new TestSplitter
func NewTestSplitter(taskStore core.TaskStore,
	buildStore core.BuildStore,
	testexecutionStore core.TestExecutionStore,
	taskQueueManager core.TaskQueueManager,
	azureClient core.AzureBlob,
	taskBuilder core.TaskBuilder,
	logger lumber.Logger) core.TestSplitter {
	return &testSplitter{
		logger:             logger,
		taskStore:          taskStore,
		buildStore:         buildStore,
		testexecutionStore: testexecutionStore,
		azureClient:        azureClient,
		taskQueueManager:   taskQueueManager,
		taskBuilder:        taskBuilder,
	}
}

func (t *testSplitter) Split(ctx context.Context,
	orgID,
	buildID,
	discoveryTaskID string,
	testIDs []string,
	executeAll bool,
	parallelCount int,
	branch string,
	splitMode core.SplitMode, subModule string) error {
	discTask, err := t.taskStore.Find(ctx, discoveryTaskID)
	if err != nil {
		t.logger.Errorf("error finding discovery taskID %s, orgID %s, buildID %s, error: %v",
			discoveryTaskID, buildID, orgID, err)
		return err
	}
	// min 1 execution task will be created
	if parallelCount < 1 {
		parallelCount = 1
	}
	executedTests, err := t.testexecutionStore.FindLatestExecution(ctx, discTask.RepoID, testIDs)
	if err != nil {
		t.logger.Errorf("error finding latest execution for impacted tests orgID %s, buildID %s, error: %v",
			orgID, buildID, err)
		return err
	}
	tasks, jobs, err := t.getTasksAndJobs(ctx, executedTests, parallelCount, orgID, subModule, discTask, splitMode)
	if err != nil {
		return err
	}
	if err := t.taskBuilder.EnqueueTaskAndJob(ctx, buildID, orgID, tasks, jobs, core.ExecutionTask); err != nil {
		return err
	}

	return nil
}

func (t *testSplitter) getTasksAndJobs(ctx context.Context,
	executedTests []*core.Test,
	parallelCount int,
	orgID, subModule string,
	discTask *core.Task,
	splitMode core.SplitMode) ([]*core.Task, []*core.Job, error) {
	var err error
	var taskHeap taskheap.Heap
	if splitMode == core.FileSplit && parallelCount > 1 {
		taskHeap = t.doFileBasedSplitting(parallelCount, executedTests)
	} else {
		taskHeap = t.doTestBasedSplitting(parallelCount, executedTests)
	}
	heapLen := taskHeap.Len()
	tasks := make([]*core.Task, heapLen)
	jobs := make([]*core.Job, heapLen)
	j := 0
	for taskHeap.Len() > 0 {
		item := heap.Pop(&taskHeap).(*core.TaskHeapItem)
		config := &core.InputLocatorConfig{
			Locators: make([]core.LocatorConfig, 0),
		}
		for _, locator := range strings.Split(item.TestsAllocated, constants.TestLocatorsDelimiter) {
			if locator != "" {
				locatorconfig := new(core.LocatorConfig)
				locatorconfig.Locator = locator
				config.Locators = append(config.Locators, *locatorconfig)
			}
		}
		tasks[j], err = t.taskBuilder.CreateTask(ctx, item.TaskID, discTask.BuildID, orgID, discTask.RepoID, subModule, core.ExecutionTask,
			config, discTask.Tier)
		if err != nil {
			return nil, nil, err
		}
		jobs[j] = t.taskBuilder.CreateJob(item.TaskID, orgID, core.Ready)
		j++
	}
	return tasks, jobs, err
}

func (t *testSplitter) doFileBasedSplitting(parallelCount int, executedTests []*core.Test) taskheap.Heap {
	fileMap := make(map[string]*core.TaskHeapItem)
	for _, test := range executedTests {
		// min duration we will keep is 1ms
		if test.Execution.Duration == 0 {
			test.Execution.Duration += 1
		}
		file := utils.GetFileNameFromTestLocator(test.TestLocator)
		// maintain a map of files and we will add the the duration and locator of each test in it
		if val, exists := fileMap[file]; !exists {
			fileMap[file] = &core.TaskHeapItem{TotalTestsDuration: test.Execution.Duration * test.Execution.Status.Weight(),
				TestsAllocated: test.TestLocator + constants.TestLocatorsDelimiter}
		} else {
			val.TotalTestsDuration += test.Execution.Duration
			val.TestsAllocated += test.TestLocator + constants.TestLocatorsDelimiter
		}
	}
	heapLen := utils.Min(parallelCount, len(fileMap))
	taskHeap := make(taskheap.Heap, heapLen)
	for i := 0; i < heapLen; i++ {
		taskHeap[i] = &core.TaskHeapItem{TaskID: utils.GenerateUUID()}
	}
	// initialize heap
	heap.Init(&taskHeap)
	for _, file := range fileMap {
		taskHeap.UpdateHead(file.TotalTestsDuration, file.TestsAllocated)
	}
	return taskHeap
}

func (t *testSplitter) doTestBasedSplitting(parallelCount int, executedTests []*core.Test) taskheap.Heap {
	locatorExists := make(map[string]struct{})
	heapLen := utils.Min(parallelCount, len(executedTests))
	taskHeap := make(taskheap.Heap, heapLen)
	for i := 0; i < heapLen; i++ {
		taskHeap[i] = &core.TaskHeapItem{TaskID: utils.GenerateUUID()}
	}
	// update heap with latest test executions
	for _, test := range executedTests {
		// some tests will have common locators, so we assign only one locator to each container
		// so as to avoid duplicate test execution
		if _, exists := locatorExists[test.TestLocator]; exists {
			continue
		}
		// min duration we will keep is 1ms
		if test.Execution.Duration == 0 {
			test.Execution.Duration += 1
		}
		locatorExists[test.TestLocator] = struct{}{}
		// multiply test duration with its status weight
		testDuration := test.Execution.Duration * test.Execution.Status.Weight()
		testLocator := test.TestLocator + constants.TestLocatorsDelimiter
		taskHeap.UpdateHead(testDuration, testLocator)
	}
	return taskHeap
}
