package middleware

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/gin-gonic/gin"
)

const (
	defaultPerPageLimit = 10
	maxPerPageLimit     = 100
	cursorDelimiter     = ":"
	cursorTypeOffset    = "offset"
	cursorTypeNext      = "next_cursor"
)

var strDefaultPerPageLimit = strconv.Itoa(defaultPerPageLimit)

// HandlePage set page parameters for paginated apis
func HandlePage() gin.HandlerFunc {
	return func(c *gin.Context) {
		perPage := c.DefaultQuery("per_page", strDefaultPerPageLimit)
		nextCursor := c.Query("next_cursor")

		if nextCursor != "" {
			key, value, err := validateCursor(nextCursor)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, errs.ErrInvalidCursor)
				return
			}
			c.Set(key, value)
		}

		limit, err := strconv.Atoi(perPage)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, errs.ErrPerPageVal)
			return
		}
		if limit < 1 {
			limit = defaultPerPageLimit
		}
		if limit > maxPerPageLimit {
			limit = maxPerPageLimit
		}
		c.Set("limit", limit)
		c.Next()
	}
}

func validateCursor(cursor string) (cursorType, cursorValue string, err error) {
	val, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", err
	}

	value := string(val)
	i := strings.Index(value, cursorDelimiter)

	switch value[:i] {
	case cursorTypeOffset:
		return cursorTypeOffset, value[i+1:], nil
	case cursorTypeNext:
		return cursorTypeNext, value[i+1:], nil
	default:
		return "", "", errs.ErrInvalidCursor
	}
}
