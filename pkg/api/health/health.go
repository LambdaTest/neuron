package health

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler for health API
func Handler(signalCtx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		select {
		// If we receive a sigterm/sigint, we return a 500 code,
		// so that the readines k8s probe fails and the pod is removed from traffic
		case <-signalCtx.Done():
			c.Data(http.StatusInternalServerError, gin.MIMEPlain, []byte(http.StatusText(http.StatusInternalServerError)))
		default:
			c.Data(http.StatusOK, gin.MIMEPlain, []byte(http.StatusText(http.StatusOK)))
		}
	}
}
