package errors

var (
	// ErrSignatureInvalid is returned when the webhook
	// signature is invalid or cannot be calculated.
	ErrSignatureInvalid = New("Invalid webhook signature")

	// ErrUnknownEvent is returned when the webhook event
	// is not recognized by the system.
	ErrUnknownEvent = New("Unknown webhook event")

	// ErrQuery is returned when database query fails
	ErrQuery = New("SQL query failed")

	// ErrEmptyCommit is returned when there is empty commit event
	ErrEmptyCommit = New("Empty commit")

	// ErrPingEvent is returned when database query fails
	ErrPingEvent = New("webhook ping event")

	// ErrNotSupported is returned when database query fails
	ErrNotSupported = New("Event not supported")

	// ErrMarshalJSON is returned when json marshal failed
	ErrMarshalJSON = New("JSON marshal failed")

	// ErrUnMarshalJSON is returned when json unmarshal failed
	ErrUnMarshalJSON = New("JSON unmarshal failed")

	// ErrAzureUpload is returned when azure blob upload fails
	ErrAzureUpload = New("failed to upload blob to azure storage")

	// ErrRedisKeyNotFound is returned when redis key is not found
	ErrRedisKeyNotFound = New("Redis key not found")
)
