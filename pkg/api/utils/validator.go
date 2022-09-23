package utils

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/gin-gonic/gin"
)

const maxTimeSpan = 180
const defaultStartTime = "2006-01-02T15:04:05Z"
const parsedDefaultStartTime = "2006-01-02 15:04:05 +0000 UTC"

// ContextData represent the data added to gin context from query paramse
type ContextData struct {
	*core.UserData
	RepoName        string `validate:"repo"`
	OrgName         string `validate:"org"`
	RepoID          string
	OrgID           string
	ExecID          string    `validate:"exec_id"`
	IntervalType    string    `validate:"type"`
	StartDate       time.Time `validate:"start_date"`
	EndDate         time.Time `validate:"end_date"`
	Limit           int
	Offset          int
	NextCursor      string
	TaskID          string `validate:"task_id"`
	BuildID         string `validate:"build_id"`
	LogsTag         string `validate:"logs_tag"`
	GitProviderType string `validate:"git_provider"`
}

//nolint:funlen,gocyclo,exhaustive
// ExtractAndValidateData returns and validates the incoming data in context if it exists.
func ExtractAndValidateData(c *gin.Context, requiredParams map[string]struct{}, paginationRequired bool) (*ContextData, int, error) {
	var cd ContextData
	cd.OrgID = c.GetString("orgID")
	cd.RepoID = c.GetString("repoID")
	// check data in context only if auth is not skipped
	if !c.GetBool("skipAuth") {
		contextValue, exists := c.Get("userData")
		if !exists {
			return nil, http.StatusUnauthorized, errs.ErrUnauthorized
		}
		cookieData, ok := contextValue.(*core.UserData)
		if !ok {
			return nil, http.StatusUnauthorized, errs.ErrUnauthorized
		}
		cd.UserData = cookieData
	}

	statusCode, perr := paginationRequiredParams(c, &cd, paginationRequired)
	if perr != nil {
		return nil, statusCode, perr
	}

	if requiredParams == nil {
		return &cd, http.StatusOK, nil
	}
	var err error

	t := reflect.TypeOf(cd)
	v := reflect.ValueOf(&cd).Elem()
	// Iterate over all available fields and read the tag value
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Get the field tag value
		tag := field.Tag.Get("validate")

		// extract values from query params and add in struct
		if _, exists := requiredParams[tag]; exists {
			switch v.Field(i).Kind() {
			case reflect.String:
				val := c.Query(tag)
				if val == "" {
					return nil, http.StatusBadRequest, errs.MissingInQueryErr(tag)
				}
				// if type is tag validate it
				if tag == "type" {
					val, err = validateInterval(val)
					if err != nil {
						return nil, http.StatusBadRequest, errs.InvalidQueryErr(tag)
					}
				}
				if tag == "logs_tag" {
					val, err = validateLogsTag(val)
					if err != nil {
						return nil, http.StatusBadRequest, errs.InvalidQueryErr(tag)
					}
				}
				v.Field(i).SetString(val)
			case reflect.Struct:
				val := c.Query(tag)
				if val == "" {
					val = defaultStartTime
				}
				if strings.HasSuffix(tag, "date") {
					date, err := time.Parse(time.RFC3339, val)
					if err != nil {
						return nil, http.StatusBadRequest, errs.InvalidQueryErr(tag)
					}
					v.Field(i).Set(reflect.ValueOf(date))
				}
			default:
				return nil, http.StatusBadRequest, errors.New("struct data type not supported")
			}
		}
	}
	// case where both start date and end date is same or null
	if cd.EndDate.String() == cd.StartDate.String() && cd.StartDate.String() == parsedDefaultStartTime {
		return nil, http.StatusBadRequest, errs.ErrInvalidDate
	}
	if cd.EndDate.String() != "" && cd.StartDate.String() != parsedDefaultStartTime &&
		(cd.EndDate.Sub(cd.StartDate).Hours()/24) > maxTimeSpan {
		return nil, http.StatusBadRequest, errs.ErrInvalidDate
	}
	return &cd, http.StatusOK, nil
}

// validateInterval validates the interval
func validateInterval(intervalType string) (string, error) {
	intervalType = strings.ToUpper(intervalType)
	switch intervalType {
	case "WEEK", "DAY", "MONTH":
	default:
		return "", errors.New("invalid type")
	}
	return intervalType, nil
}

// validateOffset validates the offset value
func validateOffset(offsetStr string) (int, error) {
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, err
	}
	switch offset {
	case 0, 1:
		offset = 0
	}
	return offset, nil
}

func paginationRequiredParams(c *gin.Context,
	cd *ContextData, paginationRequired bool) (int, error) {
	if paginationRequired {
		limit := c.GetInt("limit")
		if limit == 0 {
			return http.StatusBadRequest, errs.MissingInQueryErr("limit")
		}
		cd.Limit = limit

		offset := c.GetString("offset")
		if offset != "" {
			offset, err := validateOffset(offset)
			if err != nil {
				return http.StatusBadRequest, errs.InvalidQueryErr("offset")
			}
			cd.Offset = offset
		}

		nextCursor := c.GetString("next_cursor")
		if nextCursor != "" {
			cd.NextCursor = nextCursor
		}
	}
	return 0, nil
}

// validateLogsTag validates the type of log
func validateLogsTag(logsTag string) (string, error) {
	switch logsTag {
	case "prerun", "execution", "postrun":
	default:
		return "", errors.New("invalid type")
	}
	return logsTag, nil
}
