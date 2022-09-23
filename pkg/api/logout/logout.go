package logout

import (
	"fmt"
	"net/http"
	"time"

	apiutils "github.com/LambdaTest/neuron/pkg/api/utils"
	"github.com/LambdaTest/neuron/pkg/lumber"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/gin-gonic/gin"
)

// HandleLogout creates an http.HandlerFunc that handles
// session termination.
func HandleLogout(
	session core.Session,
	redisDB core.RedisDB,
	logger lumber.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxData, statusCode, err := apiutils.ExtractAndValidateData(c, nil, false)
		if err != nil {
			c.AbortWithStatusJSON(statusCode, err)
			return
		}
		if ctxData.Expiry == 0 || ctxData.JwtID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing expiry or jwt id"})
		}
		sub := ctxData.Expiry - time.Now().Unix()
		key := fmt.Sprintf("%s%s", core.JwtIDPrefix, ctxData.JwtID)

		if sub > 0 {
			resp := redisDB.Client().Set(c, key, ctxData.UserID, time.Duration(sub)*time.Second)
			if _, err := resp.Result(); err != nil {
				logger.Errorf("Error while blocklisting JWT Token, error %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, err.Error())
				return
			}
		}
		// delete session cookie
		http.SetCookie(c.Writer, session.DeleteCookie())
		c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
	}
}
