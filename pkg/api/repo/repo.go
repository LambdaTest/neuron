package repo

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/drone/go-scm/scm"

	"github.com/LambdaTest/neuron/pkg/constants"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/LambdaTest/neuron/pkg/utils"
	"github.com/gin-gonic/gin"
)

// RepoInput is request body when creating a new repository.
type RepoInput struct {
	Name        string       `json:"name"`
	Link        string       `json:"link"`
	TasFileName string       `json:"tas_file_name"`
	Namespace   string       `json:"namespace,omitempty"`
	JobView     core.JobView `json:"job_view,omitempty"`
}

// HandleList lists the user repositories
func HandleList(
	repoService core.RepositoryService,
	userStore core.GitUserStore,
	tokenHandler core.GitTokenHandler,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err.Error())
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		user, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("failed to find user with userID: %s, error: %v", cd.UserID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		t, _, err := tokenHandler.GetToken(ctx, true, cd.GitProvider, utils.GetRunnerNamespaceFromOrgID(orgID), cd.OrgName, user.Mask, user.ID)
		if err != nil {
			logger.Errorf("Error while getting token from token handler GitProvider %s, OrgName %s, UserID %s, error: %v",
				cd.GitProvider, cd.OrgName, user.ID, err)
			apiutils.GetTokenErrResponse(c, err)
			return
		}

		user.Oauth = *t

		repos, next, err := repoService.List(ctx, cd.Offset, c.GetInt("limit"), cd.UserID, orgID, cd.OrgName, user)
		if err != nil {
			logger.Errorf("failed to find repositories %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		if next > 0 {
			responseMetadata.NextCursor = utils.EncodeOffset(next)
		}
		c.JSON(http.StatusOK, gin.H{"repositories": repos, "response_metadata": responseMetadata})
	}
}

// HandleCreate creates a new repository
func HandleCreate(
	repoStore core.RepoStore,
	userStore core.GitUserStore,
	hookService core.HookService,
	repoService core.RepositoryService,
	tokenHandler core.GitTokenHandler,
	emailNotificationManager core.EmailNotificationManager,
	postMergeConfigStore core.PostMergeConfigStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		repoDetails := new(RepoInput)
		if err = c.ShouldBindJSON(repoDetails); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		if err = validateRequestBody(repoDetails); err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		user, orgID, err := userStore.FindByOrg(ctx, cd.UserID, repoDetails.Namespace)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "organization"))
				return
			}
			logger.Errorf("error while finding user %s %v", cd.UserID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		_, err = repoStore.FindIfActive(ctx, orgID, repoDetails.Name)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				now := time.Now()
				repo := &core.Repository{ID: utils.GenerateUUID(),
					OrgID:       orgID,
					Name:        repoDetails.Name,
					Namespace:   repoDetails.Namespace,
					TasFileName: repoDetails.TasFileName,
					Secret:      utils.GenerateUUID(),
					Admin:       user.ID,
					Active:      true,
					Mask:        utils.RandString(constants.BitSize16),
					JobView:     repoDetails.JobView,
					Updated:     now,
					Created:     now,
				}

				t, _, tokenErr := tokenHandler.GetToken(ctx, true, user.GitProvider, utils.GetRunnerNamespaceFromOrgID(repo.OrgID),
					repo.Namespace, user.Mask, user.ID)
				if tokenErr != nil {
					logger.Errorf("error while getting token for repository %s, %v", repoDetails.Link, err)
					apiutils.GetTokenErrResponse(c, err)
					return
				}
				r, findErr := repoService.Find(ctx, scm.Join(repo.Namespace, repo.Name), t, user.GitProvider)
				if findErr != nil {
					if errors.Is(findErr, errs.ErrNotFound) {
						c.JSON(http.StatusNotFound, findErr)
						return
					}
					logger.Errorf("error while finding repository %s, %v", repoDetails.Link, findErr)
					c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
					return
				}
				// check repo visibility
				repo.Private = r.Private
				repo.Link = r.Link
				if err = hookService.Create(ctx, t, user.GitProvider, repo); err != nil {
					logger.Errorf("error while creating webhook %v, for repository %s", err, repoDetails.Link)
					c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
					return
				}
				if err = repoStore.Create(ctx, repo); err != nil {
					logger.Errorf("error while creating repository %s %v", repoDetails.Link, err)
					c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
					return
				}

				if err = createDefaultPostMergeConfig(ctx, orgID, repo.Name, repoStore, postMergeConfigStore); err != nil {
					if !errors.Is(err, errs.ErrDupeKey) {
						logger.Errorf("error while creating postmergeconfig config in store, error: %v", err)
						c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
						return
					}
					// Can happen due to repo reimport after deactivation. Therefore, log and consume.
					logger.Errorf("error while creating postmergeconfig config in store, error: %v", err)
				}
				// send email notifcation to repo_admin
				repoNotificationErr := emailNotificationManager.SendRepoNotification(ctx, repo)
				// not returning error because email notification failures are not critical
				if repoNotificationErr != nil {
					logger.Errorf("failed to send  import notification for repo %s  error, %+v", repo.Name, repoNotificationErr)
				}
				c.JSON(http.StatusOK, gin.H{"message": "Repo enabled successfully."})
				return
			}
			logger.Errorf("error while finding repository %s %v", repoDetails.Link, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusConflict, gin.H{"message": "Repo already active."})
	}
}

func validateRequestBody(req *RepoInput) error {
	if req.Name == "" {
		return errs.MissingInReqErr("name")
	}
	if req.Namespace == "" {
		return errs.MissingInReqErr("namespace")
	}
	if req.TasFileName == "" {
		req.TasFileName = constants.DefaultYamlFileName
	}
	if req.JobView == "" {
		req.JobView = core.FtmOnlyView
	}
	return nil
}

// HandleListActive return active repos for a user (Handling graph data also)
func HandleListActive(
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		searchText := c.Query("text")

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		repos, err := repoStore.FindAllActive(ctx, orgID, searchText, cd.Offset, cd.Limit+1)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No active repos found for given organization."})
				return
			}
			logger.Errorf("error while finding active repos for userID: %s, org: %s, %v", cd.UserID, cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		responseMetadata := new(core.ResponseMetadata)
		// set the element as next_cursor value
		if len(repos) == cd.Limit+1 {
			responseMetadata.NextCursor = utils.EncodeOffset(cd.Offset + cd.Limit)
			// remove last element to return len==limit
			repos = repos[:len(repos)-1]
		}

		c.JSON(http.StatusOK, gin.H{"repositories": repos, "response_metadata": responseMetadata})
	}
}

// HandleListRepoSettings is the handler for listing repo settings.
func HandleListRepoSettings(
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, ctxData.UserID, ctxData.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		repo, err := repoStore.Find(ctx, orgID, ctxData.RepoName)
		if err != nil {
			logger.Errorf("error while fetching info of repo: %s, org: %s, %v", ctxData.RepoName, ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, &core.RepoSettingsInfo{
			Strict:         repo.Strict,
			ConfigFileName: repo.TasFileName,
			JobView:        repo.JobView,
		})
	}
}

// HandleCreateRepoSettings is the handler for creating repo settings.
func HandleCreateRepoSettings(
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var configInfo core.RepoSettingsInfo
		if err := c.ShouldBindJSON(&configInfo); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, ctxData.UserID, configInfo.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", configInfo.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		if err := repoStore.UpdateStatus(ctx, configInfo, orgID, ctxData.UserID); err != nil {
			logger.Errorf("error while updating info of repo: %s, org: %s, %v", configInfo.RepoName, configInfo.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Info added Successfully."})
	}
}

// HandleListBranch lists the branches of a repository
func HandleListBranch(
	branchStore core.BranchStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		listBranch, err := branchStore.FindByRepo(ctx, ctxData.RepoID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Branch", "Repo"))
				return
			}
			logger.Errorf("error while finding data for repoID %s, orgID %s, error: %v", ctxData.RepoID, ctxData.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"branches": listBranch})
	}
}

// HandleBadgeData returns the badge data of a repository
func HandleBadgeData(
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		gitProvider := c.Query("git_provider")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		badgeData, err := repoStore.FindBadgeData(ctx, ctxData.RepoName, ctxData.OrgName, branchName, "", gitProvider)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Badge data", "Repo"))
				return
			}
			logger.Errorf("error while finding data for repoName %s, orgID %s, error: %v", ctxData.RepoName, ctxData.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"badge": badgeData})
	}
}

// HandleBadgeTimeSaved finds the time saved in latestBuildID.
func HandleBadgeTimeSaved(
	buildStore core.BuildStore,
	testExecutionStore core.TestExecutionStore,
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		branchName := c.Query("branch")
		gitProvider := c.Query("git_provider")

		badgeData, err := repoStore.FindBadgeData(ctx, cd.RepoName, cd.OrgName, branchName, "", gitProvider)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Badge data", "Repo"))
				return
			}
			logger.Errorf("error while finding data for repoName %s, orgID %s, error: %v", cd.RepoName, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		totalTests, timeAllTests, cerr := testExecutionStore.FindTimeByRunningAllTests(ctx, badgeData.BuildID, "", cd.RepoID)
		if cerr != nil || timeAllTests == 0 {
			if errors.Is(cerr, errs.ErrRowsNotFound) || timeAllTests == 0 {
				c.JSON(http.StatusOK, &core.Badge{
					BuildStatus: badgeData.BuildStatus})
				return
			}
			logger.Errorf("error while finding data for repoName %s, orgID %s, error: %v", cd.RepoName, cd.OrgID, cerr)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		durMillisecond := time.Microsecond * time.Duration(0)
		if badgeData.TotalTests == 0 {
			c.JSON(http.StatusOK, &core.Badge{
				BuildStatus:      badgeData.BuildStatus,
				Duration:         badgeData.Duration,
				TotalTests:       badgeData.TotalTests,
				DiscoveredTests:  totalTests,
				Passed:           badgeData.Passed,
				Failed:           badgeData.Failed,
				Skipped:          badgeData.Skipped,
				PercentTimeSaved: 100,
				TimeSaved:        durMillisecond.String(),
			})
			return
		}

		percentTimeSaved := 0.0
		if totalTests > badgeData.TotalTests {
			timeDifference := utils.Max(0, (timeAllTests - badgeData.ExecutionTime))
			percentTimeSaved = (float64(timeDifference) / float64(timeAllTests)) * 100
			durMillisecond = time.Millisecond * time.Duration(timeDifference)
		}

		c.JSON(http.StatusOK, &core.Badge{
			BuildStatus:      badgeData.BuildStatus,
			Duration:         badgeData.Duration,
			TotalTests:       badgeData.TotalTests,
			DiscoveredTests:  totalTests,
			Passed:           badgeData.Passed,
			Failed:           badgeData.Failed,
			Skipped:          badgeData.Skipped,
			PercentTimeSaved: percentTimeSaved,
			TimeSaved:        durMillisecond.String(),
		})
	}
}

// HandleRemoveRepo is the handler for deactivating repo.
func HandleRemoveRepo(
	hookService core.HookService,
	userStore core.GitUserStore,
	repoStore core.RepoStore,
	tokenHandler core.GitTokenHandler,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		user, orgID, err := userStore.FindByOrg(ctx, ctxData.UserID, ctxData.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		repo := &core.Repository{
			Name:      ctxData.RepoName,
			Namespace: ctxData.OrgName,
			OrgID:     orgID,
		}
		t, _, tokenErr := tokenHandler.GetToken(ctx, true, user.GitProvider, utils.GetRunnerNamespaceFromOrgID(repo.OrgID),
			repo.Namespace, user.Mask, user.ID)
		if tokenErr != nil {
			logger.Errorf("error while getting token for repository %s, %v", ctxData.RepoName, err)
			apiutils.GetTokenErrResponse(c, err)
			return
		}
		if err = hookService.Delete(ctx, user, repo, t); err != nil {
			logger.Errorf("error while deleting webhook %v, for repository %s", err, ctxData.RepoName)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		if err = repoStore.RemoveRepo(ctx, ctxData.RepoName, orgID, ctxData.UserID); err != nil {
			logger.Errorf("error while updating info of repo: %s, org: %s, %v", ctxData.RepoName, ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Repo deactivated Successfully."})
	}
}

// HandleListBranches lists the branches of a repositories
func HandleListBranches(
	repoService core.RepositoryService,
	userStore core.GitUserStore,
	tokenHandler core.GitTokenHandler,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, true)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err.Error())
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		user, _, err := userStore.FindByOrg(ctx, cd.UserID, cd.OrgName)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("failed to find user with userID: %s, error: %v", cd.UserID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		t, _, tokenErr := tokenHandler.GetToken(ctx, false, user.GitProvider, "", "", user.Mask, user.ID)
		if tokenErr != nil {
			logger.Errorf("error while getting token for repository %s, %v", cd.RepoName, err)
			apiutils.GetTokenErrResponse(c, err)
			return
		}

		user.Oauth = *t
		repoSlug := cd.OrgName + "/" + cd.RepoName

		branches, next, err := repoService.ListBranches(ctx, cd.Offset, cd.Limit, repoSlug, user)
		if err != nil {
			logger.Errorf("failed to find branches for repositories %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		responseMetadata := new(core.ResponseMetadata)
		if next > 0 {
			responseMetadata.NextCursor = utils.EncodeOffset(next)
		}
		c.JSON(http.StatusOK, gin.H{"branches": branches, "response_metadata": responseMetadata})
	}
}

func createDefaultPostMergeConfig(ctx context.Context,
	orgID, repoName string,
	repoStore core.RepoStore,
	postMergeConfigStore core.PostMergeConfigStore) error {
	repo, err := repoStore.Find(ctx, orgID, repoName)
	if err != nil {
		return err
	}
	now := time.Now()
	// Creating a default postmerge config.
	// Not doing in repo tx because this can still be added from repo settings page, if errored
	postMergeConfig := &core.PostMergeConfig{
		ID:           utils.GenerateUUID(),
		RepoID:       repo.ID,
		Org:          orgID,
		Branch:       "*",
		IsActive:     true,
		StrategyName: core.AfterNCommitStrategy,
		Threshold:    "1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return postMergeConfigStore.Create(ctx, postMergeConfig)
}

// HandleBadgeFlakyTests finds the flaky tests in latestBuildID.
func HandleBadgeFlakyTests(
	repoStore core.RepoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"repo": {}, "org": {}}, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		branchName := c.Query("branch")
		gitProvider := c.Query("git_provider")

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		badgeData, err := repoStore.FindBadgeDataFlaky(ctx, cd.RepoName, cd.OrgName, branchName, gitProvider)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Badge data", "Repo"))
				return
			}
			logger.Errorf("error while finding data for repoName %s, orgID %s, error: %v", cd.RepoName, cd.OrgID, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}

		c.JSON(http.StatusOK, &core.BadgeFlaky{
			BuildStatus:   badgeData.BuildStatus,
			FlakyTests:    badgeData.FlakyTests,
			NonFlakyTests: badgeData.NonFlakyTests,
			Quarantined:   badgeData.Quarantined,
			BuildID:       badgeData.BuildID,
		})
	}
}
