package yamlconfig

import (
	"context"
	"io"
	"net/http"

	"github.com/LambdaTest/neuron/pkg/constants"
	"github.com/LambdaTest/neuron/pkg/lumber"
	nucleusutils "github.com/LambdaTest/test-at-scale/pkg/utils"

	"github.com/gin-gonic/gin"
)

// HandleValidation validates the request
func HandleValidation(logger lumber.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()
		reqBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Request Body"})
			return
		}
		if _, errV := nucleusutils.ValidateStruct(ctx, reqBytes, constants.DefaultYamlFileName); errV != nil {
			c.JSON(http.StatusOK, gin.H{"error": errV.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Valid"})
	}
}
