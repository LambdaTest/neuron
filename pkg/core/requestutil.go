package core

import "context"

// Requests is a util interface for making API Requests
type Requests interface {
	// MakeAPIRequest is an utility function for making API requests
	MakeAPIRequest(ctx context.Context, httpMethod, endpoint string, body []byte, token string) ([]byte, error)
}
