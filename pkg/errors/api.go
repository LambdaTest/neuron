package errors

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents the request payload validation error.
type ValidationError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

var (
	// ErrInvalidToken is returned when the api request token is invalid.
	ErrInvalidToken = New("Invalid or missing token")

	// ErrUnauthorized is returned when the user is not authorized.
	ErrUnauthorized = New("Unauthorized")

	// ErrForbidden is returned when user access is forbidden.
	ErrForbidden = New("Forbidden")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = New("Not Found")

	// ErrInvalidDriver is returned when SCM driver is not defined
	ErrInvalidDriver = New("Invalid Git SCM driver")

	// ErrInvalidCursor is returned when the cursor is invalid in query params
	ErrInvalidCursor = New("Invalid cursor")

	// ErrPerPageVal is returned when the per_page is invalid in query params
	ErrPerPageVal = New("Invalid per_page value")

	// ErrMissingOrgID is returned when orgID not found in the request context.
	ErrMissingOrgID = New("Missing orgID in request context")

	// ErrMissingSecreteKey is returned when orgID not found in the request context.
	ErrMissingSecreteKey = New("Missing lambdatestSecretKey in request context")

	// ErrInvalidLoggerInstance is returned when logger instance is not supported.
	ErrInvalidLoggerInstance = New("Invalid logger instance")

	// ErrInvalidDate is returned when date difference exceeds 6 months
	ErrInvalidDate = New("endDate and startDate time difference limit exceeded")
)

// MissingInReqErr is a error function corresponding to missing request entities.
func MissingInReqErr(field string) error {
	return New(fmt.Sprintf("Missing %s in request body.", field))
}

// EntityNotFoundErr is a error function corresponding to missing entities.
func EntityNotFoundErr(entity, container string) error {
	return New(fmt.Sprintf("%s not found for given %s.", entity, container))
}

// InvalidQueryErr is a error function corresponding to invalid queries.
func InvalidQueryErr(key string) error {
	return New(fmt.Sprintf("Invalid %s in request query parameters.", key))
}

// InvalidInReqErr is a error function corresponding to invalid requests.
func InvalidInReqErr(field string) error {
	return New(fmt.Sprintf("Invalid %s in request body.", field))
}

// MissingInQueryErr is a error function corresponding to missing query params.
func MissingInQueryErr(key string) error {
	return New(fmt.Sprintf("Missing %s in request query parameters.", key))
}

// ValidationErr is a error function corresponding to invalid request payloads.
func ValidationErr(err error) interface{} {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		return validationErr(verr)
	}
	return New(err.Error())
}

func validationErr(verr validator.ValidationErrors) []ValidationError {
	errs := []ValidationError{}
	for _, f := range verr {
		err := f.ActualTag()
		if f.Param() != "" {
			err = fmt.Sprintf("%s=%s", err, f.Param())
		}
		errs = append(errs, ValidationError{Field: f.Field(), Reason: err})
	}
	return errs
}
