package errors

var (
	// ErrTimeoutExceeded is returned when graceful timeout period exceeds.
	ErrTimeoutExceeded = New("Timeout exceeded")
	// ErrInvalidEnvironemt is returned when the env is incorrect.
	ErrInvalidEnvironemt = New("Invalid Environment")
	// ErrInvalidQueuePayload is returned when type assertion fails in queue producer.
	ErrInvalidQueuePayload = New("Invalid Queue Payload")
	// GenericUserFacingBEErrRemark is generic error message for user facing errors.
	GenericUserFacingBEErrRemark = New("Unexpected error")
	// GenericTaskTimeoutErrorRemark is generic error message for task timeout errors.
	GenericTaskTimeoutErrorRemark = New("Timed Out")
	// GenericTaskAbortedErrorRemark is generic error message for task abort errors.
	GenericTaskAbortedErrorRemark = New("Aborted")
	// GenericJobTimeoutErrRemark is a generic remark after job timeout
	GenericJobTimeoutErrRemark = New("Job timeout")
	// GenericErrorMessage is generic error message returned to UI
	GenericErrorMessage = New("Unexpected error. Please try again later.")
	// ErrUnknownContainer is returned when the azure container name is invalid.
	ErrUnknownContainer = New("Unknown azure container")
	// ErrAzureConfig is returned when missing values in azure blob config
	ErrAzureConfig = New("missing values in azure blob config")
	// ErrConfigAlreadyExists is returned when config id exists
	ErrConfigAlreadyExists = New("Config already exists")
	// ErrConfigNotFound is returned when no config found
	ErrConfigNotFound = New("config not found")
	// ErrPostMergeThresholdNotMet is returned when postmerge threshold is not met
	ErrPostMergeThresholdNotMet = New("postmerge threshold is not met")
	// ErrUnSupportedPostMergeStrategy is returned when user provide unsupported postmerge strategy
	ErrUnSupportedPostMergeStrategy = New("unsupported postmerge strategy")
	// ErrUnsupportedGitProvider is returned when try to integrate unsupported provider repo
	ErrUnsupportedGitProvider = New("unsupported gitprovider")
	// ErrTypeAssertionFailed is returned when type assertion fails for a var
	ErrTypeAssertionFailed = New("type assertion failed")
	// ErrMissingBuildData  indicates build data missing while setting claim  in JWT token.
	ErrMissingBuildData = New("Missing build data while creating token")
	// ErrMissingBuildIDClaim indicates build_id claim missing in JWT token.
	ErrMissingBuildIDClaim = New("Missing build_id in token")
	// ErrMissingRepoIDClaim indicates repo_id claim missing in JWT token.
	ErrMissingRepoIDClaim = New("Missing repo_id in token")
	// ErrMissingOrgIDClaim indicates org_id claim missing in JWT token.
	ErrMissingOrgIDClaim = New("Missing org_id in token")
	// ErrParsingSecret indicates that the secret cannot be parsed
	ErrParsingSecret = New("Unparseable secret")
)

// Error represents a json-encoded API error.
type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new error message.
func New(text string) error {
	return &Error{Message: text}
}

// ErrSkipRetry is returned when retry attempt needs to be skipped
type ErrSkipRetry struct {
	Err error
}

// Error gives a human-readable description of the error.
func (e *ErrSkipRetry) Error() string {
	return e.Err.Error()
}
