package core

import "gopkg.in/guregu/null.v4/zero"

// ResponseMetadata for setting metadata in response
type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

// TestStatus represents the status of a test
type TestStatus struct {
	CommitID   string      `json:"commit_id"`
	AuthorName string      `json:"author_name"`
	Status     zero.String `json:"status"`
	Created    int         `json:"created_at"`
	StartTime  zero.Int    `json:"start_time"`
	EndTime    zero.Int    `json:"end_time"`
	BuildID    string      `json:"build_id"`
	Branch     string      `json:"branch"`
}
