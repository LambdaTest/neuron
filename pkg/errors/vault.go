package errors

// ErrNamespaceNotFound is returned when a vault namespace is not found.
var ErrNamespaceNotFound = New("Namespace not found")

// ErrSecretNotFound is returned when a vault secret is not found in path.
var ErrSecretNotFound = New("Secrets not found")
