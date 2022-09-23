package middleware

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/core"
	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
)

const (
	orgKey          = "org"
	orgIDKey        = "orgID"
	repoKey         = "repo"
	repoIDKey       = "repoID"
	skipAuthKey     = "skipAuth"
	userDataKey     = "userData"
	gitProviderKey  = "git_provider"
	synapseProxyKey = "Lambdatest-SecretKey"
)

// HandleJWTVerification returns a middleware that checks
// if the JWT in the request is valid.
func HandleJWTVerification(
	session core.Session,
	redisDB core.RedisDB,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// skip if skipAuthKey is set
		if c.GetBool(skipAuthKey) {
			c.Next()
			return
		}
		userData, err := session.Authorize(c)
		if err != nil {
			logger.Errorf("failed to find user details in cookie %v", err)
			return
		}
		key := fmt.Sprintf("%s%s", core.JwtIDPrefix, userData.JwtID)

		// if jwtID exists in redis then it is blocklisted
		resp := redisDB.Client().Exists(c, key)

		n, err := resp.Result()
		if err != nil {
			logger.Errorf("error while finding JWT ID in redis %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
			return
		}

		// JWT token is blocklisted
		if n == 1 {
			logger.Debugf("Token with ID %s is invalidated", userData.JwtID)
			c.AbortWithStatusJSON(http.StatusForbidden, errs.ErrInvalidJWTToken)
			return
		}
		c.Set(userDataKey, userData)
		c.Next()
	}
}

// HandleJWTVerificationInternal handles internal-nucleus authentication and authorisation
func HandleJWTVerificationInternal(internalJWT core.Session, logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildData, err := internalJWT.AuthorizeInternal(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, errs.ErrInvalidJWTToken)
			return
		}
		repoID := c.Query("repoID")
		buildID := c.Query("buildID")
		orgID := c.Query("orgID")
		if repoID != buildData.RepoID || buildID != buildData.BuildID || orgID != buildData.OrgID {
			logger.Errorf("token data does not match with requested data")
			c.AbortWithStatusJSON(http.StatusForbidden, errs.ErrInvalidJWTToken)
			return
		}
		c.Next()
	}
}

// HandleAuthentication checks if the user is authenticated and belongs the given organization
func HandleAuthentication(
	userOrgStore core.UserOrgStore,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// skip if skipAuthKey is set
		if c.GetBool(skipAuthKey) {
			c.Next()
			return
		}
		contextValue, exists := c.Get(userDataKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusNotFound, errs.ErrNotFound)
			return
		}
		cookieData, ok := contextValue.(*core.UserData)
		if !ok {
			c.AbortWithStatusJSON(http.StatusNotFound, errs.ErrNotFound)
			return
		}
		orgID := c.GetString(orgIDKey)
		if orgID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.ErrMissingOrgID)
			return
		}
		err := userOrgStore.FindIfExists(c, cookieData.UserID, orgID)
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, errs.EntityNotFoundErr("User", "org"))
				return
			}
			logger.Errorf("error while finding user %s for orgID %s, %v", cookieData.UserID, orgID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Next()
	}
}

// HandleAuthenticationSynapse checks if the synapse is authenticated and belongs the given organization
func HandleAuthenticationSynapse(
	orgStore core.OrganizationStore,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		encryptedKey := c.Request.Header.Get(synapseProxyKey)
		secretKeyByte, err := base64.StdEncoding.DecodeString(encryptedKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, errs.ErrInvalidAuthHeader)
			return
		}
		if len(secretKeyByte) == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.ErrMissingToken)
			return
		}

		_, err = orgStore.FindBySecretKey(c, string(secretKeyByte))
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.AbortWithStatusJSON(http.StatusForbidden, errs.ErrForbidden)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		c.Next()
	}
}

// HandleRepoValidation returns a middleware that checks if an active repository is public or private
func HandleRepoValidation(
	repostore core.RepoStore,
	restrictRoute bool,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgName := c.Query(orgKey)
		if orgName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.MissingInQueryErr(orgKey))
			return
		}
		repoName := c.Query(repoKey)
		if repoName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.MissingInQueryErr(repoKey))
			return
		}
		gitProvider := c.Query(gitProviderKey)
		if gitProvider == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.MissingInQueryErr(gitProviderKey))
			return
		}
		// TODO: can add caching here
		r, err := repostore.FindActiveByName(c, repoName, orgName, core.SCMDriver(gitProvider))
		if err != nil {
			if errors.Is(err, errs.ErrRowsNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, errs.ErrNotFound)
				return
			}
			logger.Errorf("error while finding repository %s/%s, %v", orgName, repoName, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, errs.GenericErrorMessage)
			return
		}
		// skip auth if repo is public, route is not restricted and API has GET method
		skipAuth := !r.Private && !restrictRoute && c.Request.Method == http.MethodGet
		c.Set(skipAuthKey, skipAuth)
		// set orgID and repoID in request context
		c.Set(orgIDKey, r.OrgID)
		c.Set(repoIDKey, r.ID)
		c.Set(gitProviderKey, gitProvider)
		c.Next()
	}
}
