package user

import (
	"context"
	"errors"
	"net/http"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/utils"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

type userResponse struct {
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	UserID   string `json:"user_id"`
}

// UserInfoDetails is a request body of user information
type UserInfoDetails struct {
	Org             string `json:"org"`
	UserDescription string `json:"user_description"`
	Experience      string `json:"experience"`
	TeamSize        string `json:"team_size"`
}

// HandleUserData facilitates new repository import
func HandleUserData(
	userStore core.GitUserStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		user, err := userStore.FindByID(ctx, cd.UserData.UserID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				logger.Errorf("user not found in DB, userID: %s", cd.UserData.UserID)
				c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
				return
			}
			logger.Errorf("failed to find user %v, userID: %s, err", err.Error(), cd.UserData.UserID)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, &userResponse{
			UserName: user.Username,
			Email:    user.Email,
			UserID:   user.ID,
		})
	}
}

// HandleUserInfo adds user information provided by user to Db
func HandleUserInfo(
	userStore core.GitUserStore,
	userInfoStore core.UserInfoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userInfo UserInfoDetails
		if err := c.ShouldBindJSON(&userInfo); err != nil {
			logger.Errorf("error while binding json %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}

		cd, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		_, orgID, err := userStore.FindByOrg(ctx, cd.UserID, userInfo.Org)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user for org %s, %v", cd.OrgName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		info := &core.UserInfo{
			ID:              utils.GenerateUUID(),
			UserID:          cd.UserID,
			UserDescription: userInfo.UserDescription,
			Experience:      constants.UserExperience[userInfo.Experience],
			TeamSize:        constants.UserTeamSize[userInfo.TeamSize],
			OrgID:           orgID,
			Created:         time.Now(),
		}

		if err := userInfoStore.Create(ctx, info); err != nil {
			logger.Errorf("failed to insert user info, error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Info added Successfully."})
	}
}

// HandleUserInfoData gets user info data from DB
func HandleUserInfoData(
	userStore core.GitUserStore,
	userInfoStore core.UserInfoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
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
		userInfoData, err := userInfoStore.Find(ctx, ctxData.UserID, orgID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "Query"))
				return
			}
			logger.Errorf("error while finding data for userID %s, org %s, %v", ctxData.UserID, ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"is_active": userInfoData.IsActive, "user_info": &UserInfoDetails{
			Org:             ctxData.OrgName,
			UserDescription: userInfoData.UserDescription,
			Experience:      userInfoData.Experience,
			TeamSize:        userInfoData.TeamSize,
		}})
	}
}

// HandleUserUsageInfo gets TAS usage information for the given user
func HandleUserUsageInfo(
	userStore core.GitUserStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
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
		listUserUsage, err := userStore.FindUserTasUsageInfo(ctx, orgID, ctxData.UserID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.JSON(http.StatusNotFound, errs.EntityNotFoundErr("Usage", "org"))
				return
			}
			logger.Errorf("error while finding data for userID %s, org %s, %v", ctxData.UserID, ctxData.OrgName, err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"User_Usages": listUserUsage})
	}
}

// HandleUserInfoDemoData stores users data that are on TAS site
func HandleUserInfoDemoData(
	userDemoStore core.UserDemoStore,
	emailStore core.EmailNotificationManager,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userInfo *core.UserInfoDemoDetails
		if err := c.ShouldBindJSON(&userInfo); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, errs.ValidationErr(err))
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		go func() {
			err := emailStore.UserDemoNotification(context.Background(), userInfo)
			if err != nil {
				logger.Errorf("error while sending mail to TAS team, %v", err)
			}
		}()
		userInfo.ID = utils.GenerateUUID()
		userInfo.Name = userInfo.FirstName + " " + userInfo.LastName
		if err := userDemoStore.Create(ctx, userInfo); err != nil {
			logger.Errorf("error while inserting user data, %v", err)
			c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Info added successfully"})
	}
}

// HandleMarkActive marks active flag in user_info db
func HandleMarkActive(
	userStore core.GitUserStore,
	userInfoStore core.UserInfoStore,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		cd, statusCode, err := apiutils.ExtractAndValidateData(c, map[string]struct{}{"org": {}}, false)
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
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		info := &core.UserInfo{
			ID:       utils.GenerateUUID(),
			UserID:   cd.UserID,
			OrgID:    orgID,
			Created:  time.Now(),
			IsActive: true,
		}
		if err := userInfoStore.UpdateActiveUser(ctx, info); err != nil {
			logger.Errorf("failed to mark active flag for the particular user, error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "User marked as active."})
	}
}

// HandleEligibilityInfo sends email to TAS team incase of open-source user setup
func HandleEligibilityInfo(
	emailStore core.EmailNotificationManager,
	logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var openSourceUser *core.OpenSourceUserInfo
		if err := c.ShouldBindJSON(&openSourceUser); err != nil {
			logger.Errorf("error while binding json, error: %v", err)
			c.JSON(http.StatusBadRequest, errs.ValidationErr(err))
			return
		}

		err := emailStore.OpenSourceUserNotification(context.Background(), openSourceUser)
		if err != nil {
			logger.Errorf("error while sending mail to TAS team, %v", err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Notification mail sent successfully"})
	}
}
