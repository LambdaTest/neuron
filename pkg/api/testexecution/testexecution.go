package testexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/guregu/null.v4/zero"
)

// HandleCreate inserts test execution report data in table
func HandleCreate(
	testStore core.TestStore,
	testSuiteStore core.TestSuiteStore,
	testExecutionStore core.TestExecutionStore,
	testSuiteExecutionStore core.TestSuiteExecutionStore,
	testExecutionService core.TestExecutionService,
	executionStore core.ExecutionStore,
	buildStore core.BuildStore,
	taskStore core.TaskStore,
	flakyExecutionStore core.FlakyExecutionStore,
	flakyConfigStore core.FlakyConfigStore,
	commentService core.CommentService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var reqBody = &core.TestReportRequestPayload{}
		if err := c.ShouldBindJSON(reqBody); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, errs.ValidationErr(err))
			return
		}
		orgID := reqBody.OrgID
		repoID := reqBody.RepoID
		buildID := reqBody.BuildID
		taskID := reqBody.TaskID

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		build, err := buildStore.GetBuildCache(ctx, buildID)
		if err != nil {
			logger.Errorf("failed to find build records for buildID %s, orgID %s, err %s",
				buildID, orgID, err)
			if !errors.Is(err, errs.ErrRedisKeyNotFound) {
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
				return
			}
			c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("buildID", "build"))
			return
		}

		if reqBody.TaskType == core.FlakyTask {
			var flakyConfig *core.FlakyConfig
			flakyConfigType := core.FlakyConfigPostMerge
			if build.BuildTag == core.PreMergeTag {
				flakyConfigType = core.FlakyConfigPreMerge
			}
			flakyConfig, err = flakyConfigStore.FindActiveFlakyConfig(ctx, repoID, build.Branch, flakyConfigType)
			if err != nil {
				logger.Errorf("failed to find flaky config for orgID %s, buildID %s, repoID %s, branch %s, error: %v",
					orgID, buildID, repoID, build.Branch, err)
				if !errors.Is(err, errs.ErrRowsNotFound) {
					c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
					return
				}
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("flakyconfig", "repository, branch and build-type"))
				return
			}
			flakyExecutionResults, blockTests, flakyMeta, taskStatus, errE := extractFlakyTestExecutionInfo(reqBody, flakyConfig, build.Branch)
			if errE != nil {
				logger.Errorf("failed to extract ExecutionInfo orgID %s, buildID %s, repoID %s, branch %s, error: %v",
					orgID, buildID, repoID, build.Branch, errE)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
				return
			}

			if err = flakyExecutionStore.Create(ctx, flakyExecutionResults, blockTests, buildID); err != nil {
				logger.Errorf("error while running queries in transaction for"+
					"flaky test executions, orgID %s, buildID %s, taskID %s, repoID %s, error: %v",
					orgID, buildID, taskID, repoID, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
				return
			}

			if build.BuildTag == core.PreMergeTag {
				go commentService.CreateFlakyComment(context.Background(), build, flakyMeta, buildID)
			}
			if err = buildStore.IncrementCacheCount(context.Background(), buildID, "flaky_test_count", flakyMeta.FlakyTests); err != nil {
				logger.Errorf("error while incrementing flaky test count in cache, orgID %s, buildID %s, taskID %s, repoID %s, error: %v",
					orgID, buildID, taskID, repoID, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
				return
			}
			c.JSON(http.StatusOK, &core.TestReportResponsePayload{TaskID: taskID, TaskStatus: taskStatus})
			return
		}

		testExecutions, testSuiteExecutions, taskStatus, err := extractExecutionResults(reqBody, buildStore, testExecutionService, logger)
		if err != nil {
			logger.Errorf("failed to extract execution results for orgID %s, buildID %s, taskID %s, repoID %s, error: %v",
				orgID, buildID, taskID, repoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		if err := executionStore.Create(ctx,
			testExecutions,
			testSuiteExecutions,
			buildID); err != nil {
			logger.Errorf("error while running queries in transaction for test executions, orgID %s, buildID %s, taskID %s, repoID %s, error: %v",
				orgID, buildID, taskID, repoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, &core.TestReportResponsePayload{TaskID: taskID, TaskStatus: taskStatus})
	}
}

// HandleList return tests executed for a repo paginated
func HandleList(
	testExecutionStore core.TestExecutionStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		testID := c.Param("testID")
		branchName := c.Query("branch")
		statusFilter := c.Query("status")
		searchID := c.Query("id")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tests, err := testExecutionStore.FindByTestID(ctx, testID, cd.RepoName, cd.OrgID, branchName, statusFilter, searchID,
			cd.StartDate, cd.EndDate, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Test details", "testID"))
				return
			}
			logger.Errorf("error while finding test execution data for testID %s  orgID %s, repoID %s, %v",
				testID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		// set the element as next_cursor value
		if len(tests) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			// remove last element to return len==limit
			tests = tests[:len(tests)-1]
		}
		c.JSON(http.StatusOK, gin.H{"test_results": tests, "response_metadata": responseMetadata})
	}
}

// HandleExecMetricsBlob returns details of test metrics for a repo
func HandleExecMetricsBlob(
	testExecutionService core.TestExecutionService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}, "task_id": {}, "build_id": {}, "exec_id": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testMetricsPath := fmt.Sprintf("test/%s/%s/%s/%s/%s", cd.OrgID, cd.RepoID, cd.BuildID, cd.TaskID, constants.DefaultMetricsFileName)
		metrics, err := testExecutionService.FetchMetrics(ctx, testMetricsPath, cd.ExecID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Metrics", "testID"))
				return
			}
			logger.Errorf("error while finding metrics for taskID %s in buildID %s, orgID %s, repoID %s, %v",
				cd.BuildID, cd.TaskID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, metrics)
	}
}

// nolint:dupl
// HandleFailureLogs returns details of test failure logs for a given test execution id
func HandleFailureLogs(
	testExecutionService core.TestExecutionService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}, "task_id": {}, "build_id": {}, "exec_id": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		testMetricsPath := fmt.Sprintf("test/%s/%s/%s/%s/%s", cd.OrgID, cd.RepoID, cd.BuildID, cd.TaskID, constants.DefaultFailureDetailsFileName)
		failureLog, err := testExecutionService.FetchTestFailures(ctx, testMetricsPath, cd.ExecID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Failure Logs", "testID"))
				return
			}
			logger.Errorf("error while failure logs for taskID %s in buildID %s, orgID %s, repoID %s, %v",
				cd.BuildID, cd.TaskID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, failureLog)
	}
}

// HandleStatusData returns status details of tests for a repo
func HandleStatusData(
	testExecutionStore core.TestExecutionStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {},
			"start_date": {}, "end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		testID := c.Param("testID")
		branchName := c.Query("branch")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		tests, err := testExecutionStore.FindStatus(ctx, testID, cd.IntervalType, cd.RepoName, cd.OrgID, branchName, cd.StartDate, cd.EndDate)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Statuses", "testID"))
				return
			}
			logger.Errorf("error while finding statuses for test %s orgID %s, repoID %s, %v",
				testID, cd.OrgID, cd.RepoID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, tests)
	}
}

func initTestResults(numberOfExecutions int) []core.TestExecutionStatus {
	results := make([]core.TestExecutionStatus, numberOfExecutions)
	for idx := range results {
		results[idx] = core.TestNotRun
	}
	return results
}

func extractFlakyTestExecutionInfo(reqBody *core.TestReportRequestPayload,
	flakyConfig *core.FlakyConfig,
	branch string) ([]*core.FlakyTestExecution, []*core.BlockTest, *core.FlakyExecutionMetadata, core.Status, error) {
	executionResults := reqBody.Results
	testResultMap := make(map[string]*core.AggregatedFlakyTestExecInfo, len(executionResults))
	for i := range executionResults {
		testPayload := executionResults[i].TestPayload
		for j := range testPayload {
			testResult := testPayload[j]
			aggregatedTestExecInfo := testResultMap[testResult.TestID]
			if aggregatedTestExecInfo == nil {
				aggregatedTestExecInfo = &core.AggregatedFlakyTestExecInfo{
					Threshold:          flakyConfig.Threshold,
					NumberOfExecutions: flakyConfig.ConsecutiveRuns,
					Results:            initTestResults(flakyConfig.ConsecutiveRuns),
					FirstTransition:    -1,
				}
				testResultMap[testResult.TestID] = aggregatedTestExecInfo
			}
			aggregatedTestExecInfo.Results[i] = testResult.Status
			isValidStatus := testResult.Status == core.TestPassed || testResult.Status == core.TestFailed
			if aggregatedTestExecInfo.LastValidStatus != "" && isValidStatus && testResult.Status != aggregatedTestExecInfo.LastValidStatus {
				aggregatedTestExecInfo.TotalTransitions++
				if aggregatedTestExecInfo.FirstTransition == -1 {
					aggregatedTestExecInfo.FirstTransition = i + 1
				}
			}
			if isValidStatus {
				// set new last valid status
				aggregatedTestExecInfo.LastValidStatus = testResult.Status
			}
		}
	}
	return createFlakyTestExecutionResults(testResultMap, flakyConfig,
		reqBody.BuildID, reqBody.TaskID, branch)
}

func createFlakyTestExecutionResults(testResultMap map[string]*core.AggregatedFlakyTestExecInfo,
	flakyConfig *core.FlakyConfig,
	buildID, taskID, branch string) ([]*core.FlakyTestExecution, []*core.BlockTest, *core.FlakyExecutionMetadata, core.Status, error) {
	results := make([]*core.FlakyTestExecution, 0, len(testResultMap))
	blocktests := make([]*core.BlockTest, 0)
	flakyMeta := new(core.FlakyExecutionMetadata)
	flakyMeta.ImpactedTests = len(testResultMap)
	now := time.Now()
	taskStatus := core.TaskPassed
	for testID, execInfo := range testResultMap {
		execInfoRaw, err := json.Marshal(execInfo)
		if err != nil {
			return nil, nil, nil, core.TaskError, err
		}
		result := &core.FlakyTestExecution{
			ID:       utils.GenerateUUID(),
			TestID:   testID,
			BuildID:  buildID,
			TaskID:   taskID,
			AlgoName: flakyConfig.AlgoName,
			ExecInfo: string(execInfoRaw),
			Status:   getFlakyTestStatus(execInfo),
		}
		results = append(results, result)
		if result.Status == core.TestFlaky {
			// if any test is flaky, the task status is failed
			taskStatus = core.TaskFailed
			flakyMeta.FlakyTests++
			if flakyConfig.AutoQuarantine {
				blockTest := &core.BlockTest{
					ID:        utils.GenerateUUID(),
					TestID:    result.TestID,
					Status:    core.TestQuarantined,
					BlockedBy: "FTM",
					Branch:    branch,
					RepoID:    flakyConfig.RepoID,
					Created:   now,
					Updated:   now,
				}
				blocktests = append(blocktests, blockTest)
			}
		}
	}
	return results, blocktests, flakyMeta, taskStatus, nil
}

func extractExecutionResults(reqBody *core.TestReportRequestPayload,
	buildStore core.BuildStore,
	testExecutionService core.TestExecutionService,
	logger lumber.Logger,
) ([]*core.TestExecution, []*core.TestSuiteExecution, core.Status, error) {
	orgID := reqBody.OrgID
	repoID := reqBody.RepoID
	buildID := reqBody.BuildID
	commitID := reqBody.CommitID
	taskID := reqBody.TaskID
	taskStatus := core.TaskPassed
	testExecutions := make([]*core.TestExecution, 0)
	testSuiteExecutions := make([]*core.TestSuiteExecution, 0)
	for ind := range reqBody.Results {
		testExecutionsPerRun, azBlobTestMetrics, failureDetails := processTestPayloads(&reqBody.Results[ind], commitID, taskID, buildID)
		testSuiteExecutionsPerRun, azBlobTestSuiteMetrics := processTestSuitePayloads(&reqBody.Results[ind], commitID, taskID, buildID)

		testSuiteExecutions = append(testSuiteExecutions, testSuiteExecutionsPerRun...)
		testExecutions = append(testExecutions, testExecutionsPerRun...)

		if len(azBlobTestMetrics) > 0 {
			testMetricsPath := fmt.Sprintf("test/%s/%s/%s/%s/%s", orgID, repoID, buildID, taskID, constants.DefaultMetricsFileName)
			if err := testExecutionService.StoreTestMetrics(context.Background(), testMetricsPath, azBlobTestMetrics); err != nil {
				logger.Errorf("failed to store test metrics on azure store, orgID %s, buildID %s, taskID %s, repoID %s, error %v",
					orgID, buildID, taskID, repoID, err)
				return nil, nil, core.TaskError, err
			}
		}
		if len(azBlobTestSuiteMetrics) > 0 {
			testSuiteMetricsPath := fmt.Sprintf("test_suite/%s/%s/%s/%s/%s", orgID, repoID, buildID, taskID, constants.DefaultMetricsFileName)
			if err := testExecutionService.StoreTestMetrics(context.Background(), testSuiteMetricsPath, azBlobTestSuiteMetrics); err != nil {
				logger.Errorf("failed to store test suite metrics on azure store, orgID %s, buildID %s, taskID %s, repoID %s, error %v",
					orgID, buildID, taskID, repoID, err)
				return nil, nil, core.TaskError, err
			}
		}
		if len(failureDetails) > 0 {
			failureDetailsPath := fmt.Sprintf("test/%s/%s/%s/%s/%s", orgID, repoID, buildID, taskID, constants.DefaultFailureDetailsFileName)
			if err := testExecutionService.StoreTestFailures(context.Background(), failureDetailsPath, failureDetails); err != nil {
				logger.Errorf("failed to store test failure details on azure store, orgID %s, buildID %s, taskID %s, repoID %s, error %v",
					orgID, buildID, taskID, repoID, err)
				return nil, nil, core.TaskError, err
			}
		}
	}
	failedTestsCount := 0
	// if any test fails the task, status is set to failed
	for _, t := range testExecutions {
		if t.Status == core.TestFailed {
			taskStatus = core.TaskFailed
			failedTestsCount += 1
		}
	}
	if err := buildStore.IncrementCacheCount(context.Background(), buildID, "failed_test_count", failedTestsCount); err != nil {
		return nil, nil, core.TaskError, err
	}
	return testExecutions, testSuiteExecutions, taskStatus, nil
}

func processTestPayloads(
	reqBody *core.ExecutionResult,
	commitID, taskID, buildID string,
) (testExecutions []*core.TestExecution, azBlobTestMetrics [][]string, failureDetails map[string]string) {
	testExecutions = make([]*core.TestExecution, 0, len(reqBody.TestPayload))
	azBlobTestMetrics = make([][]string, 0, len(reqBody.TestPayload))
	failureDetails = make(map[string]string)

	for idx := range reqBody.TestPayload {
		item := reqBody.TestPayload[idx]
		testExecID := utils.GenerateUUID()
		// skipped tests have pending status
		if item.Status == core.TestPending {
			item.Status = core.TestSkipped
		}
		if !(item.Status == core.TestSkipped || item.Status == core.TestBlocklisted) {
			for _, stat := range item.Stats {
				azBlobTestMetrics = append(azBlobTestMetrics, []string{testExecID, strconv.FormatFloat(stat.CPU, 'f', -1, constants.BitSize64),
					strconv.FormatUint(stat.Memory, constants.Base10), stat.RecordTime.String()})
			}
		}
		testExecutions = append(testExecutions, &core.TestExecution{ID: testExecID,
			TestID:          item.TestID,
			Status:          item.Status,
			BlocklistSource: zero.StringFrom(item.BlocklistSource),
			EndTime:         zero.TimeFrom(item.EndTime),
			StartTime:       zero.TimeFrom(item.StartTime),
			Duration:        item.Duration,
			CommitID:        commitID,
			TaskID:          taskID,
			BuildID:         buildID})

		if item.FailureMessage != "" {
			failureDetails[testExecID] = item.FailureMessage
		}
	}
	return testExecutions, azBlobTestMetrics, failureDetails
}

func processTestSuitePayloads(
	reqBody *core.ExecutionResult,
	commitID, taskID, buildID string,
) (testSuiteExecutions []*core.TestSuiteExecution,
	azBlobTestSuiteMetrics [][]string) {
	testSuiteExecutions = make([]*core.TestSuiteExecution, 0, len(reqBody.TestSuitePayload))
	azBlobTestSuiteMetrics = make([][]string, 0, len(reqBody.TestSuitePayload))
	for idx := range reqBody.TestSuitePayload {
		item := reqBody.TestSuitePayload[idx]
		testSuiteExecID := utils.GenerateUUID()
		// overwrite status for blocklisted test suites;
		if item.Blocklisted {
			item.Status = core.TestSuiteBlocklisted
		} else {
			// skipped test suites have pending status
			if item.Status == core.TestSuitePending {
				item.Status = core.TestSuiteSkipped
			}

			for _, stat := range item.Stats {
				azBlobTestSuiteMetrics = append(azBlobTestSuiteMetrics, []string{
					testSuiteExecID, strconv.FormatFloat(stat.CPU, 'f', -1, constants.BitSize64),
					strconv.FormatUint(stat.Memory, constants.Base10), stat.RecordTime.String()})
			}
		}
		testSuiteExecutions = append(testSuiteExecutions, &core.TestSuiteExecution{
			ID:              testSuiteExecID,
			SuiteID:         item.SuiteID,
			Status:          item.Status,
			BlocklistSource: zero.StringFrom(item.BlocklistSource),
			EndTime:         zero.TimeFrom(item.EndTime),
			StartTime:       zero.TimeFrom(item.StartTime),
			Duration:        item.Duration,
			CommitID:        commitID,
			TaskID:          taskID,
			BuildID:         buildID})
	}
	return testSuiteExecutions, azBlobTestSuiteMetrics
}

func getFlakyTestStatus(execInfo *core.AggregatedFlakyTestExecInfo) core.TestExecutionStatus {
	if execInfo.TotalTransitions >= execInfo.Threshold {
		return core.TestFlaky
	}
	for _, status := range execInfo.Results {
		if status == core.TestBlocklisted {
			return core.TestBlocklisted
		}
		if status == core.TestSkipped {
			return core.TestSkipped
		}
		if status == core.TestPending {
			return core.TestPending
		}
	}
	return core.TestNonFlaky
}
