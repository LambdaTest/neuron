package utils

import (
	"errors"
	"net/http"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/gin-gonic/gin"
)

// GetTokenErrResponse sets proper api err response for given err
func GetTokenErrResponse(c *gin.Context, err error) {
	if errors.Is(err, errs.ErrSecretNotFound) {
		c.JSON(http.StatusForbidden, err)
		return
	}
	if errors.Is(err, errs.ErrTypeAssertionFailed) {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Type assertion failed when parsing map."})
		return
	}
	c.JSON(http.StatusInternalServerError, errs.GenericErrorMessage)
}
