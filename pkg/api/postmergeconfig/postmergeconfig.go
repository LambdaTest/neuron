package postmergeconfig

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// HandleList returns postmergeconfig for a repo
func HandleList(postMergeConfigStore core.PostMergeConfigStore,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c,
			map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, cd.RepoName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", cd.RepoName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		postMergeConfig, err := postMergeConfigStore.FindAllPostMergeConfig(ctx, repo.ID)
		if err != nil {
			logger.Errorf("error while finding postmergeconfig %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		for ind := range postMergeConfig {
			postMergeConfig[ind].Repo = cd.RepoName
			postMergeConfig[ind].Org = cd.OrgName
		}

		if len(postMergeConfig) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"message": "No postmerge config."})
			return
		}

		c.JSON(http.StatusOK, postMergeConfig)
	}
}

// HandleCreate creates new postmergeconfig
func HandleCreate(
	postMergeConfigStore core.PostMergeConfigStore,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		postMergeConfig := &core.PostMergeConfig{}
		if err := c.ShouldBindJSON(postMergeConfig); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		if err = validateRequestBody(postMergeConfig); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, postMergeConfig.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", postMergeConfig.Org, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, postMergeConfig.Repo)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", postMergeConfig.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		now := time.Now()
		postMergeConfig.ID = utils.GenerateUUID()
		postMergeConfig.RepoID = repo.ID
		postMergeConfig.UpdatedAt = now
		postMergeConfig.CreatedAt = now

		err = postMergeConfigStore.Create(ctx, postMergeConfig)
		if err != nil {
			if errors.Is(err, errs.ErrDupeKey) {
				logger.Errorf("error while creating postmergeconfig config in store, error: %v", err)
				c.JSON(http.StatusConflict, errs.ErrConfigAlreadyExists)
				return
			}
			logger.Errorf("error while creating postmergeconfig config in store, error: %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

// HandleUpdate updates existing postmergeconfig for a branch
func HandleUpdate(
	postMergeConfigStore core.PostMergeConfigStore,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		postMergeConfig := &core.PostMergeConfig{}
		if err := c.ShouldBindJSON(postMergeConfig); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		if postMergeConfig.ID == "" {
			c.JSON(http.StatusBadRequest, errs.MissingInReqErr("ID"))
			return
		}

		if err = validateRequestBody(postMergeConfig); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, postMergeConfig.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, postMergeConfig.Repo)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", postMergeConfig.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		postMergeConfig.RepoID = repo.ID
		postMergeConfig.UpdatedAt = time.Now()

		if err := postMergeConfigStore.Update(ctx, postMergeConfig); err != nil {
			if errors.Is(err, errs.ErrDupeKey) {
				logger.Errorf("error while updating postmergeconfig config in store, error: %v", err)
				c.JSON(http.StatusConflict, errs.ErrConfigAlreadyExists)
				return
			}
			if errors.Is(err, errs.ErrNotFound) {
				logger.Errorf("error while creating postmergeconfig config in store, error: %v", err)
				c.JSON(http.StatusNotFound, errs.ErrConfigNotFound)
				return
			}
			logger.Errorf("error while updating postmergeconfig config in store, error: %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

func validateRequestBody(req *core.PostMergeConfig) error {
	if req.Repo == "" {
		return errs.MissingInReqErr("repo")
	}
	if req.Org == "" {
		return errs.MissingInReqErr("org")
	}
	if req.Branch == "" {
		return errs.MissingInReqErr("branch")
	}
	if req.StrategyName == "" {
		return errs.MissingInReqErr("strategyName")
	}
	if req.StrategyName != core.AfterNCommitStrategy {
		return errs.InvalidInReqErr("strategyName")
	}
	threshold, err := strconv.Atoi(req.Threshold)
	if err != nil || threshold < 1 {
		return errs.New("Valid threshold must be greater than zero")
	}
	return nil
}
