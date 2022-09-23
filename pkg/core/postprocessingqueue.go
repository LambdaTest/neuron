package core

// PostProcessingQueuePayload represents the payload for item in post processing queue.
type PostProcessingQueuePayload struct {
	OrgID          string `json:"org_id"`
	BuildID        string `json:"build_id"`
	BaseCommitID   string `json:"base_commit_id"`
	RepoID         string `json:"repo_id"`
	PayloadAddress string `json:"payload_address"`
}
