package flakyconfig

import (
	"context"
	"errors"
	"net/http"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/utils"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

// HandleList returns flakyconfig for a repo
func HandleList(flakyConfigStore core.FlakyConfigStore,
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
		flakyConfig, err := flakyConfigStore.FindAllFlakyConfig(ctx, repo.ID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Flaky config", "repo"))
				return
			}
			logger.Errorf("error while finding flakyconfig %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, flakyConfig)
	}
}

// HandleCreate creates new flakyconfig
func HandleCreate(
	flakyConfigStore core.FlakyConfigStore,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		flakyConfig := &core.FlakyConfig{}
		if err := c.ShouldBindJSON(flakyConfig); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		if err = validateRequestBody(flakyConfig); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, flakyConfig.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", flakyConfig.Org, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, flakyConfig.Repo)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", flakyConfig.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		err = flakyConfigStore.FindIfActiveConfigExists(ctx, repo.ID, flakyConfig.Branch, flakyConfig.ConfigType)
		if err == nil {
			// means config exists in table
			c.JSON(http.StatusConflict, errs.ErrConfigAlreadyExists)
			return
		}
		if !errors.Is(err, errs.ErrRowsNotFound) {
			logger.Errorf("failed to find repoID for repo: %s error: %v", flakyConfig.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		now := time.Now()
		flakyConfig.ID = utils.GenerateUUID()
		flakyConfig.RepoID = repo.ID
		flakyConfig.UpdatedAt = now
		flakyConfig.CreatedAt = now
		err = flakyConfigStore.Create(ctx, flakyConfig)
		if err != nil {
			logger.Errorf("error while creating flakyconfig config in store, error: %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

// HandleUpdate updates existing flakyconfig for a branch
func HandleUpdate(
	flakyConfigStore core.FlakyConfigStore,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		flakyConfig := &core.FlakyConfig{}
		if err := c.ShouldBindJSON(flakyConfig); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		if flakyConfig.ID == "" {
			c.JSON(http.StatusBadRequest, errs.MissingInReqErr("ID"))
			return
		}

		if err = validateRequestBody(flakyConfig); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, flakyConfig.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		repo, err := repoStore.Find(ctx, orgID, flakyConfig.Repo)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "repo"))
				return
			}
			logger.Errorf("failed to find repoID for repo: %s error: %v", flakyConfig.Repo, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		flakyConfig.RepoID = repo.ID
		flakyConfig.UpdatedAt = time.Now()

		if err := flakyConfigStore.Update(ctx, flakyConfig); err != nil {
			if errors.Is(err, errs.ErrDupeKey) {
				logger.Errorf("error while updating flakyconfig config in store, error: %v", err)
				c.JSON(http.StatusConflict, errs.ErrConfigAlreadyExists)
				return
			}
			if errors.Is(err, errs.ErrNotFound) {
				logger.Errorf("error while creating flakyconfig config in store, error: %v", err)
				c.JSON(http.StatusNotFound, errs.ErrConfigNotFound)
				return
			}
			logger.Errorf("error while updating flakyconfig config in store, error: %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}

func validateRequestBody(req *core.FlakyConfig) error {
	if req.Repo == "" {
		return errs.MissingInReqErr("repo")
	}
	if req.Org == "" {
		return errs.MissingInReqErr("org")
	}
	// TODO: add checks if branch valid or not
	if req.Branch == "" {
		return errs.MissingInReqErr("branch")
	}
	if req.AlgoName == "" {
		return errs.MissingInReqErr("algoName")
	}
	if req.ConfigType == "" {
		return errs.MissingInQueryErr("config_type")
	}
	if req.AlgoName != core.RunningXTimes {
		return errs.InvalidInReqErr("algoName")
	}
	if !(req.ConfigType == core.FlakyConfigPreMerge || req.ConfigType == core.FlakyConfigPostMerge) {
		return errs.InvalidInReqErr("config_type")
	}
	if req.Threshold < 1 {
		return errs.New("valid threshold should greater than 0")
	}
	if req.ConsecutiveRuns < 1 {
		return errs.New("valid consecutive runs should be greater than 0")
	}
	return nil
}
