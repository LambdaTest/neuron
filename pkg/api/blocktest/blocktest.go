package blocktest

import (
	"context"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

//TODO: changes in blocktest table for suites and test files.

// HandleFind returns list block tests
func HandleFind(blockTestStore core.BlockTestStore, taskStore core.TaskStore, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		repoID := c.Query("repoID")
		branch := c.Query("branch")
		taskID := c.Query("taskID")
		if repoID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "RepoID missing in query parameters."})
			return
		}
		if branch == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "branch missing in query parameters."})
			return
		}
		if taskID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "taskID missing in query parameters."})
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		statusFilter := ""
		task, err := taskStore.Find(ctx, taskID)
		if err != nil {
			logger.Errorf("error while finding task for taskID %s error %v", taskID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		// Post and Pre merge builds will skip blocklist and quarantined
		// FTM tasks will run quarantined tests if they are part of impacted test set. Because Quarantined tests
		// may fixed by dev and to identify the test is fixed or not we need to run test in FTM.
		if task.Type == core.FlakyTask {
			statusFilter = string(core.TestBlocklisted)
		}

		blockTests, err := blockTestStore.FindBlockTest(ctx, repoID, branch, statusFilter)
		if err != nil {
			logger.Errorf("error while finding block listed tests repoID %s, branch %s, taskID %s, error: %v",
				repoID, branch, taskID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		if len(blockTests) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"message": "No block tests found."})
			return
		}
		c.JSON(http.StatusOK, blockTests)
	}
}
