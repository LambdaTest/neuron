package coverage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

// CoverageInput is the input for coverage data from coverage pods.
type CoverageInput struct {
	BuildID       string          `json:"build_id"`
	RepoID        string          `json:"repo_id"`
	CommitID      string          `json:"commit_id"`
	BlobLink      string          `json:"blob_link"`
	TotalCoverage json.RawMessage `json:"total_coverage"`
}

// HandleFind finds the coverage details for a commit.
func HandleFind(coverageStore core.TestCoverageStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoID := c.Query("repoID")
		commitID := c.Query("commitID")

		if repoID == "" || commitID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "repoID and commitID required"})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		coverage, err := coverageStore.FindCommitCoverage(ctx, repoID, commitID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No parent coverage blob found."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"blob_link": coverage.Blob, "parent_commit": coverage.CommitID})
	}
}

// HandleCreate creates the coverage data for a commit.
func HandleCreate(coverageManager core.CoverageManager,
	azureClient core.AzureBlob,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestPayload []CoverageInput
		if err := c.ShouldBindJSON(&requestPayload); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		}
		now := time.Now()
		coverageData := make([]*core.TestCoverage, 0, len(requestPayload))
		for _, item := range requestPayload {
			c := &core.TestCoverage{
				ID:            utils.GenerateUUID(),
				RepoID:        item.RepoID,
				Blob:          azureClient.ReplaceWithCDN(item.BlobLink),
				CommitID:      item.CommitID,
				TotalCoverage: item.TotalCoverage,
				Created:       now,
				Updated:       now,
			}
			coverageData = append(coverageData, c)
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		if len(coverageData) > 0 {
			if err := coverageManager.InsertCoverageData(ctx, coverageData); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

// HandleFindCoverageInfo finds the coverage data for a particular commit
func HandleFindCoverageInfo(userStore core.GitUserStore,
	coverageStore core.TestCoverageStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		commitID := c.Param("sha")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		coverage, err := coverageStore.FindCommitCoverage(ctx, ctxData.RepoID, commitID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("coverage", "commitID"))
				return
			}
			logger.Errorf("error while finding coverage for repoID %s orgID %s commitID %s, error: %v",
				ctxData.RepoID, ctxData.OrgID, commitID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"blob_link": coverage.Blob + "/coverage-merged.json", "total_coverage": coverage.TotalCoverage})
	}
}
