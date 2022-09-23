package testlist

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/guregu/null.v4/zero"
)

type testListInput struct {
	Tests           []testInput      `json:"tests" binding:"omitempty,min=0,dive"`
	ImpactedTests   []string         `json:"impactedTests"`
	TestSuites      []testSuiteInput `json:"testSuites" binding:"omitempty,min=0,dive"`
	ExecuteAllTests bool             `json:"executeAllTests"`
	Parallelism     int              `json:"parallelism"`
	RepoID          string           `json:"repoID" binding:"required"`
	BuildID         string           `json:"buildID" binding:"required"`
	CommitID        string           `json:"commitID" binding:"required"`
	SplitMode       core.SplitMode   `json:"splitMode"`
	TaskID          string           `json:"taskID" binding:"required"`
	OrgID           string           `json:"orgID" binding:"required"`
	Branch          string           `json:"branch" binding:"required"`
	Tier            core.Tier        `json:"tier"  binding:"required,oneof=xsmall small medium large xlarge"`
	ContainerImage  string           `json:"containerImage"`
	SubModule       string           `json:"subModule"`
}

type testInput struct {
	TestID      string `json:"testID" binding:"required"`
	Detail      string `json:"_detail"`
	SuiteID     string `json:"suiteID"`
	Title       string `json:"title"`
	Filelocator string `json:"locator" binding:"required"`
}

type testSuiteInput struct {
	ID            string `json:"suiteID" binding:"required"`
	ParentSuiteID string `json:"parentSuiteID"`
	Name          string `json:"suiteName"`
	TotalTests    int    `json:"totalTests"`
}

// HandleCreate inserts test suite, test and test dag data in database
func HandleCreate(
	discoveryStore core.DiscoveryStore,
	buildStore core.BuildStore,
	testSplitter core.TestSplitter,
	gitStatusService core.GitStatusService,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqBody := new(testListInput)
		if err := c.ShouldBindJSON(reqBody); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, errs.ValidationErr(err))
			return
		}
		buildID := reqBody.BuildID
		orgID := reqBody.OrgID
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		if err := buildStore.IncrementCacheCount(ctx, buildID, core.ProcessedSubModule, 1); err != nil {
			logger.Errorf("error while incrementing key %s for builID %s in buildCache, error %v",
				core.ProcessedSubModule, buildID, err)
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		if len(reqBody.Tests) == 0 {
			logger.Infof("no tests discovered in orgID %s commitID %s, repoID %s, buildID %s, taskID %s",
				orgID, reqBody.CommitID, reqBody.RepoID, buildID, reqBody.TaskID)
			c.JSON(http.StatusOK, nil)
			return
		}
		testData, err := extractTestData(reqBody)
		if err != nil {
			logger.Errorf("error while extracting test data, marshal error: %v, orgID %s, buildID %s, repoID %s",
				orgID, buildID, reqBody.RepoID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		suiteData, err := extractSuiteData(reqBody)
		if err != nil {
			logger.Errorf("error while extracting suite data, marshal error %v, orgID %s, buildID %s, repoID %s",
				orgID, buildID, reqBody.RepoID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		commitDiscovery := &core.CommitDiscovery{ID: utils.GenerateUUID(),
			RepoID:    reqBody.RepoID,
			CommitID:  reqBody.CommitID,
			TestIDs:   testData.rawTestIDs,
			SuiteIDs:  suiteData.rawSuiteIDs,
			SubModule: reqBody.SubModule,
		}

		if cerr := discoveryStore.Create(ctx, testData.tests, testData.testBranches,
			suiteData.testSuites, suiteData.testSuiteBranches, commitDiscovery); cerr != nil {
			logger.Errorf("error while running queries in transaction for test discovery, orgID %s, buildID %s, repoID %s, error: %v",
				orgID, buildID, reqBody.RepoID, cerr)
			c.JSON(http.StatusInternalServerError, gin.H{"message": cerr.Error()})
			return
		}
		// if no impacted tests skip execution.
		if len(testData.impactedTestIDs) == 0 {
			logger.Debugf("no impacted tests found in orgID %s commitID %s, repoID %s, buildID %s, taskID %s",
				orgID, reqBody.CommitID, reqBody.RepoID, buildID, reqBody.TaskID)
			c.JSON(http.StatusOK, nil)
			return
		}
		buildUpdateMap := map[string]interface{}{
			core.ContainerImage:     reqBody.ContainerImage,
			core.ImpactedTestExists: true,
		}

		/*
			Following things will be updated in build cache
			1.  container Image (to keep track of custom image to be used in test / flaky exec)
			2.  Impacted tests exists (keep field in build hash to prevent build being marked as stopped.)
		*/
		if err := buildStore.UpdateBuildCache(ctx, buildID, buildUpdateMap, true); err != nil {
			logger.Errorf("error while updating build cache, orgID %s, buildID %s, repoID %s, error: %v",
				orgID, buildID, reqBody.RepoID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		go func() {
			if buildCache, err := scheduleExecTasks(reqBody, testData.impactedTestIDs,
				testSplitter, buildStore, logger); err != nil {
				now := time.Now()
				build := &core.Build{
					ID:      buildID,
					Remark:  zero.StringFrom(errs.GenericUserFacingBEErrRemark.Error()),
					Status:  core.BuildError,
					EndTime: zero.TimeFrom(now),
					Updated: now,
				}

				if buildCache != nil {
					go func() {
						if errC := gitStatusService.UpdateGitStatus(context.Background(),
							buildCache.GitProvider,
							buildCache.RepoSlug,
							buildID,
							buildCache.TargetCommitID,
							buildCache.TokenPath,
							buildCache.InstallationTokenPath,
							core.BuildError,
							buildCache.BuildTag); errC != nil {
							logger.Errorf("error while updating status on GitHub for commitID: %s, buildID: %s, orgID: %s, err: %v",
								buildCache.TargetCommitID, buildID, orgID, errC)
						}
					}()
				}
				logger.Debugf("marking buildID %s as stopped with status %s, orgID %s", build.ID, build.Status, orgID)
				// mark build as stopped and delete cache from redis.
				if err := buildStore.MarkStopped(context.Background(), build, orgID); err != nil {
					logger.Errorf("failed to mark buildID %s as completed, orgID %s, error %v", build.ID, orgID, err)
				}
			}
		}()
		c.JSON(http.StatusOK, nil)
	}
}

func scheduleExecTasks(reqBody *testListInput,
	impactedTestIDs []string,
	testSplitter core.TestSplitter,
	buildStore core.BuildStore,
	logger lumber.Logger) (*core.BuildCache, error) {
	buildID := reqBody.BuildID
	orgID := reqBody.OrgID
	build := &core.Build{ID: buildID, Tier: reqBody.Tier, Updated: time.Now()}
	ctx := context.Background()
	if err := buildStore.UpdateTier(ctx, build); err != nil {
		logger.Errorf("failed in updating tier info in build table, buildID %s, orgID %s, err %v", buildID, orgID, err)
		return nil, err
	}
	execAll := len(impactedTestIDs) == len(reqBody.Tests)
	buildCache, err := buildStore.GetBuildCache(ctx, buildID)
	if err != nil {
		logger.Errorf("error in finding build cache for buildID %s, %v", buildID, err)
		return nil, err
	}
	// keep test splitting default
	if reqBody.SplitMode == "" {
		reqBody.SplitMode = core.TestSplit
	}
	// call splitter.
	if err := testSplitter.Split(ctx,
		orgID, buildID, reqBody.TaskID,
		impactedTestIDs, execAll, reqBody.Parallelism,
		reqBody.Branch, reqBody.SplitMode, reqBody.SubModule); err != nil {
		logger.Errorf("error while splitting tests, orgID %s, buildID %s, repoID %s, error: %v",
			orgID, buildID, reqBody.RepoID, err)
		return buildCache, err
	}
	return buildCache, nil
}

//FIXME: DebutCommit: may not be correct in concurrent run
func extractTestData(reqBody *testListInput) (data *testData,
	err error) {
	tests := make([]*core.Test, 0, len(reqBody.Tests))
	testBranches := make([]*core.TestBranch, 0, len(reqBody.Tests))
	testIDs := make([]string, 0, len(reqBody.Tests))
	now := time.Now()
	for _, test := range reqBody.Tests {
		tests = append(tests, &core.Test{ID: test.TestID,
			Name:        test.Title,
			TestSuiteID: zero.StringFrom(test.SuiteID),
			RepoID:      reqBody.RepoID,
			DebutCommit: reqBody.CommitID,
			TestLocator: test.Filelocator,
			SubModule:   reqBody.SubModule,
			Updated:     now,
			Created:     now})
		testIDs = append(testIDs, test.TestID)
		testBranches = append(testBranches, &core.TestBranch{ID: utils.GenerateUUID(),
			TestID:     test.TestID,
			RepoID:     reqBody.RepoID,
			BranchName: reqBody.Branch,
			Updated:    now,
			Created:    now})
	}
	var impactedTestIDs []string
	if len(reqBody.ImpactedTests) > 0 {
		impactedTestIDs = reqBody.ImpactedTests
	} else if reqBody.ExecuteAllTests {
		// if no diff and no impacted tests then we have to run all discovered tests.
		impactedTestIDs = testIDs
	} else {
		// if diff exists and no impacted tests then we have no impacted tests.
		impactedTestIDs = []string{}
	}
	rawTestIDs, err := json.Marshal(testIDs)

	return &testData{
		tests:           tests,
		testBranches:    testBranches,
		rawTestIDs:      rawTestIDs,
		impactedTestIDs: impactedTestIDs,
	}, err
}

func extractSuiteData(reqBody *testListInput) (data *suiteData,
	err error) {
	testSuites := make([]*core.TestSuite, 0, len(reqBody.TestSuites))
	testSuiteBranches := make([]*core.TestSuiteBranch, 0, len(reqBody.TestSuites))
	suiteIDs := make([]*testSuitesObject, 0, len(reqBody.TestSuites))
	now := time.Now()
	for _, suite := range reqBody.TestSuites {
		testSuites = append(testSuites, &core.TestSuite{ID: suite.ID,
			Name:          suite.Name,
			RepoID:        reqBody.RepoID,
			DebutCommit:   reqBody.CommitID,
			ParentSuiteID: zero.StringFrom(suite.ParentSuiteID),
			Updated:       now,
			Created:       now,
			SubModule:     reqBody.SubModule,
			TotalTests:    suite.TotalTests})
		testSuiteBranches = append(testSuiteBranches, &core.TestSuiteBranch{ID: utils.GenerateUUID(),
			TestSuiteID: suite.ID,
			RepoID:      reqBody.RepoID,
			BranchName:  reqBody.Branch,
			Updated:     now,
			Created:     now})
		suiteIDs = append(suiteIDs, &testSuitesObject{suite.ID, suite.TotalTests})
	}
	rawSuiteIDs, err := json.Marshal(suiteIDs)
	return &suiteData{
		testSuites:        testSuites,
		testSuiteBranches: testSuiteBranches,
		rawSuiteIDs:       rawSuiteIDs,
	}, err
}

type testSuitesObject struct {
	ID         string `json:"suite_id"`
	TotalTests int    `json:"total_tests"`
}

type testData struct {
	tests           []*core.Test
	testBranches    []*core.TestBranch
	rawTestIDs      []byte
	impactedTestIDs []string
}

type suiteData struct {
	testSuites        []*core.TestSuite
	testSuiteBranches []*core.TestSuiteBranch
	rawSuiteIDs       []byte
}
