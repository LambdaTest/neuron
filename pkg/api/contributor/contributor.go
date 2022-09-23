package contributor

import (
	"context"
	"errors"
	"math"
	"net/http"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// HandleLists lists the all the authors of a repo
func HandleLists(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		author := c.Query("author")
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		commits, err := commitStore.FindAuthors(ctx, cd.RepoName, cd.OrgID, branchName, author, cd.NextCursor, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Author", "repository"))
				return
			}
			logger.Errorf("failed to find commit for repoID %s, orgID %s, error: %v", cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if len(commits) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeCursor(commits[len(commits)-1].Author.Name)
			commits = commits[:len(commits)-1]
		}
		c.JSON(http.StatusOK, gin.H{"authors": commits, "response_metadata": responseMetadata})
	}
}

// HandleCommitActivityData lists the commit activity for each author.
func HandleCommitActivityData(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		//TODO: optimize
		authors, nextAuthor, err := commitStore.FindAuthorCommitActivity(ctx, cd.RepoName, cd.OrgID, cd.NextCursor, cd.Limit)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Commit activity", "author"))
				return
			}
			logger.Errorf("failed to find commit for repoID %s, orgID %s, error: %v", cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		if nextAuthor != "" {
			responseMetadata.NextCursor = utils.EncodeCursor(nextAuthor)
		}

		c.JSON(http.StatusOK, gin.H{"authors": authors, "response_metadata": responseMetadata})
	}
}

// HandleListGraph lists the graph of all the contributors of a repo
func HandleListGraph(
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		graphs, err := commitStore.FindContributorGraph(ctx, cd.RepoName, cd.OrgID, branchName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Graph", "repository"))
				return
			}
			logger.Errorf("failed to find graph for repoID %s, orgID %s, error: %v",
				cd.RepoID, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, gin.H{"graph": graphs})
	}
}

// HandleAuthorDetails lists the details of author for that repo
func HandleAuthorDetails(
	testStore core.TestStore,
	commitStore core.GitCommitStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}, "start_date": {},
			"end_date": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		authorName := c.Query("author")
		if authorName == "" {
			c.JSON(http.StatusBadRequest, "author name can't be empty")
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		authorStats := &core.AuthorStats{}
		var totalTransitions, blocklistedTests int
		g, errCtx := errgroup.WithContext(ctx)
		g.Go(func() error {
			stats, sErr := commitStore.FindAuthorStats(errCtx, cd.RepoName, cd.OrgID, branchName, authorName, cd.StartDate, cd.EndDate)
			if sErr != nil && sErr != errs.ErrRowsNotFound {
				return sErr
			}
			authorStats = stats
			return nil
		})
		g.Go(func() error {
			transitions, tErr := commitStore.FindTransitions(errCtx, cd.RepoName, cd.OrgID, branchName, authorName, cd.StartDate, cd.EndDate)
			if tErr != nil && tErr != errs.ErrRowsNotFound {
				return tErr
			}
			totalTransitions = transitions
			return nil
		})
		g.Go(func() error {
			blocklistedTest, bErr := testStore.FindBlocklistedTests(ctx, cd.RepoName, cd.OrgID, branchName, authorName, cd.StartDate, cd.EndDate)
			if bErr != nil && bErr != errs.ErrRowsNotFound {
				return bErr
			}
			blocklistedTests = blocklistedTest
			return nil
		})
		if err = g.Wait(); err != nil {
			if errors.Is(err, ctx.Err()) {
				logger.Errorf("context canceled while fetching details from database for orgID %s, repoName %s, error %v",
					cd.OrgID, cd.RepoName, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			} else {
				if errors.Is(err, errs.ErrRowsNotFound) {
					c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Author Stats", "Author"))
					return
				}
				logger.Errorf("error while fetching info from database for orgID %s, repoName %s, error %v",
					cd.OrgID, cd.RepoName, err)
				c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			}
			return
		}
		var averageTransition float64
		if authorStats.CommitCount == 0 {
			averageTransition = float64(0)
		} else {
			averageTransition = (float64(totalTransitions) / float64(authorStats.CommitCount)) * 100
		}
		authorStats.AverageTransition = math.Round(averageTransition) / constants.DefaultCredits
		authorStats.BlocklistedTests = blocklistedTests

		c.JSON(http.StatusOK, gin.H{"author_stats": authorStats})
	}
}
