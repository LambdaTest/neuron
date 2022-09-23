package core

import (
	"context"
	"io"

	"github.com/drone/go-scm/scm"
)

// List of container names
const (
	PayloadContainer = iota
	MetricsContainer
	CacheContainer
	LogsContainer
	CoverageContainer
)

// EventBlob represents the azure blob storage object
type EventBlob struct {
	EventID                    string       `json:"event_id"`
	BuildID                    string       `json:"build_id"`
	RepoID                     string       `json:"repo_id"`
	OrgID                      string       `json:"org_id"`
	RepoSlug                   string       `json:"repo_slug"`
	ForkSlug                   string       `json:"fork_slug,omitempty"`
	RepoLink                   string       `json:"repo_link"`
	BaseCommit                 string       `json:"build_base_commit"`
	TargetCommit               string       `json:"build_target_commit"`
	GitProvider                SCMDriver    `json:"git_provider"`
	PrivateRepo                bool         `json:"private_repo"`
	EventType                  EventType    `json:"event_type"`
	DiffURL                    string       `json:"diff_url,omitempty"`
	PullRequestNumber          int          `json:"pull_request_number,omitempty"`
	Commits                    []scm.Commit `json:"commits,omitempty"`
	TasFileName                string       `json:"tas_file_name"`
	ParentCommitCoverageExists bool         `json:"parent_commit_coverage_exists"`
	EventBody                  []byte       `json:"-"`
	RawToken                   []byte       `json:"-"`
	BranchName                 string       `json:"branch_name"`
	LicenseTier                Tier         `json:"license_tier"`
	CollectCoverage            bool         `json:"collect_coverage"`
}

// AzureBlob defines operation for working with azure store
type AzureBlob interface {
	// UploadBytes uploads a buffer in blocks to a block blob.
	UploadBytes(ctx context.Context, path string, rawBytes []byte, containerID int, mimeType string) (string, error)
	// UploadStream copies the file held in io.Reader to the Blob at blockBlobURL
	UploadStream(ctx context.Context, path string, reader io.Reader, containerID int, mimeType string) (string, error)
	// GenerateSasURL generates Azure SAS URL
	GenerateSasURL(ctx context.Context, blobPath string, containerID int) (string, error)
	// DownloadStream downloads the data from the blob return result in io.Reader
	DownloadStream(ctx context.Context, path string, containerID int) (io.ReadCloser, error)
	// ReplaceWithCDN replaces the azure URL with Azure CDN URL.
	ReplaceWithCDN(urlStr string) string
}
