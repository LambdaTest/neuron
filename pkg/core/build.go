package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// const related to build cache
const (
	ContainerImage     string = "container_image"
	ImpactedTestExists string = "impacted_test_exists"
	TotalFlakyTasks    string = "total_flaky_tasks"
	FlakyTasksPassed   string = "flaky_tasks_passed"
	FlakyTasksFailed   string = "flaky_tasks_failed"
	FlakyTasksError    string = "flaky_tasks_error"
	FlakyTasksAborted  string = "flaky_tasks_aborted"
	TotalSubModule     string = "total_submodule"
	ProcessedSubModule string = "processed_submodule"
	TotalExecTasks     string = "total_exec_tasks"
	ExecTasksPassed    string = "exec_tasks_passed" //nolint:gosec
	ExecTasksFailed    string = "exec_tasks_failed"
	ExecTasksError     string = "exec_tasks_error"
	ExecTasksAborted   string = "exec_tasks_aborted"
	BuildFailedRemarks string = "build_failed_reamrks"
)

// BuildStatus specifies the status of a build
type BuildStatus string

// Build Status values.
const (
	BuildRunning    BuildStatus = "running"
	BuildFailed     BuildStatus = "failed"
	BuildPassed     BuildStatus = "passed"
	BuildInitiating BuildStatus = "initiating"
	BuildAborted    BuildStatus = "aborted"
	BuildError      BuildStatus = "error"
)

// Build Description values.
const (
	BuildPendingDesc string = "pending."
	BuildRunningDesc string = "running."
	BuildFailedDesc  string = "Failed"
	BuildPassedDesc  string = "Passed"
	BuildErrorDesc   string = "errored out."
	BuildAbortedDesc string = "was aborted."
	BuildUnknownDesc string = "in unknown state."
)

// Build abort tag for redis
const (
	AbortedBuild string = "aborted"
)

// BuildTag is used to tag build as Flaky, premerge, postmerge
type BuildTag string

// Build Tag values
const (
	PostMergeTag BuildTag = "postmerge"
	PreMergeTag  BuildTag = "premerge"
)

// BuildExecutionStatus specifies the counts of status
type BuildExecutionStatus struct {
	Total      int `json:"total_builds_executed,omitempty"`
	Passed     int `json:"builds_passed"`
	Skipped    int `json:"builds_skipped"`
	Failed     int `json:"builds_failed"`
	Aborted    int `json:"builds_aborted"`
	Error      int `json:"builds_error"`
	Running    int `json:"builds_running"`
	Initiating int `json:"builds_initiating"`
}

// BuildWiseTimeSaved represents information related to build wise time saved
type BuildWiseTimeSaved struct {
	BuildID           string  `json:"build_id" db:"build_id"`
	TimeAllTests      int     `json:"time_all_tests_ms" db:"time_all_tests_ms"`
	TimeImpactedTests int     `json:"time_impacted_tests_ms" db:"time_impacted_tests_ms"`
	TimeSaved         int     `json:"time_saved_ms" db:"time_saved_ms"`
	PercentTimeSaved  float64 `json:"percent_time_saved" db:"percent_time_saved"`
	NetTimeSaved      int     `json:"net_time_saved_ms" db:"net_time_saved_ms"`
}

// Build represents the user's builds.
type Build struct {
	BuildNum          int                     `json:"build_num" db:"build_num"`
	ID                string                  `json:"id" db:"id"`
	BaseCommit        string                  `json:"base_commit" db:"base_commit"`
	CommitID          string                  `json:"commit_id,omitempty" db:"commit_id"`
	CommitAuthor      string                  `json:"commit_author,omitempty"`
	CommitMessage     *zero.String            `json:"commit_message,omitempty"`
	RepoID            string                  `json:"repo_id,omitempty" db:"repo_id"`
	EventID           string                  `json:"event_id,omitempty" db:"event_id"`
	Actor             string                  `json:"-" db:"actor"`
	PayloadAddress    string                  `json:"-" db:"payload_address"`
	Created           time.Time               `json:"created_at" db:"created_at"`
	Updated           time.Time               `json:"-" db:"updated_at"`
	StartTime         zero.Time               `json:"start_time" db:"start_time"`
	EndTime           zero.Time               `json:"end_time" db:"end_time"`
	Status            BuildStatus             `json:"status,omitempty" db:"status"`
	TimeAllTests      int                     `json:"time_all_tests_ms,omitempty" db:"time_all_tests_ms,omitempty"`
	TimeImpactedTests int                     `json:"time_impacted_tests_ms,omitempty" db:"time_impacted_tests_ms,omitempty"`
	Meta              *ExecutionMeta          `json:"execution_meta,omitempty"`
	FlakyMeta         *FlakyExecutionMetadata `json:"flaky_execution_meta,omitempty"`
	ExecutionStatus   BuildExecutionStatus    `json:"build_execution_status,omitempty"`
	BuildsList        []*BuildsCreated        `json:"builds_list,omitempty"`
	Tag               BuildTag                `json:"build_tag,omitempty" db:"tag"`
	Branch            string                  `json:"branch" db:"branch_name"`
	Tier              Tier                    `json:"tier,omitempty" db:"tier"`
	Remark            zero.String             `json:"remark" db:"remark"`
	JobView           JobView                 `json:"job_view" db:"job_view"`
}

// TimeSavedData defines information related to time saved by not running unimpacted tests
type TimeSavedData struct {
	PercentTimeSaved         float64 `json:"time_saved_percent"`
	TimeTakenByAllTests      int     `json:"time_taken_all_tests_ms"`
	TimeTakenByImpactedTests int     `json:"time_taken_impacted_tests_ms"`
	TotalTests               int     `json:"total_tests"`
	TotalImpactedTests       int     `json:"total_impacted_tests"`
}

// MttfAndMttr defines data related to mean time to failure and mean time to recovery of jobs
type MttfAndMttr struct {
	MTTF      int64       `json:"mttf_in_sec" db:"-"`
	MTTR      int64       `json:"mttr_in_sec" db:"-"`
	Status    BuildStatus `json:"-" db:"status"`
	TimeStamp time.Time   `json:"-" db:"created_at"`
}

// BuildStore defines datastore operation for working with build
type BuildStore interface {
	// CreateInTx persists a new build to the datastore and executes the statement within supplied transaction.
	CreateInTx(ctx context.Context, tx *sqlx.Tx, build *Build) error
	// FindByRepo returns the builds of a repo.
	FindByRepo(ctx context.Context, repoName, orgID, buildID, branchName, statusFilter,
		searchID string, authorsNames, tags []string, lastSeenTime time.Time, limit int) ([]*Build, error)
	// FindByBuildID returns the build from buildID.
	FindByBuildID(ctx context.Context, buildID string) (*Build, error)
	// FindByCommit returns the build by commitID.
	FindByCommit(ctx context.Context, commitID, repoName, orgID, branchName string, lastSeenTime time.Time, limit int) ([]*Build, error)
	// Update persists build changes to the datastore.
	Update(ctx context.Context, build *Build) error
	// MarkStarted  marks build as started in the datastore.
	MarkStarted(ctx context.Context, build *Build) error
	// MarkStopped   marks build as stopped in the datastore.
	MarkStopped(ctx context.Context, build *Build, orgID string) error
	// MarkStoppedInTx  marks build as stopped in the datastore and executes the query in given transaction.
	MarkStoppedInTx(ctx context.Context, tx *sqlx.Tx, build *Build, orgID string) error
	// FindBuildStatus return the builds information in a selected date range
	FindBuildStatus(ctx context.Context, repoName, orgID, branchName string, startDate, endDate time.Time) ([]*Build, error)
	// FindBuildStatusFailed return the builds information in a selected date range of failed builds.
	FindBuildStatusFailed(ctx context.Context, repoName, orgID, branchName string, startDate, endDate time.Time, limit int) ([]*Build, error)
	// CheckBuildCache check if build cache exits in redis.
	CheckBuildCache(ctx context.Context, buildID string) error
	// StoreBuildCache store build details in redis cache.
	StoreBuildCache(ctx context.Context, buildID string, buildCache *BuildCache) error
	// GetBuildCache get build details from redis cache.
	GetBuildCache(ctx context.Context, buildID string) (*BuildCache, error)
	// DeleteBuildCache deletes the build details from redis cache.
	DeleteBuildCache(ctx context.Context, buildID string) error
	// UpdateExecTaskCount updates the count of exec tasks in build cache.
	UpdateExecTaskCount(ctx context.Context, buildID string, count int) error
	// UpdateFlakyExecutionCount updates the flaky execution count  in build cache.
	UpdateFlakyExecutionCount(ctx context.Context, buildID string, executionCount int) error
	// StopStaleBuildsInTx marks the builds as error after timeout within specified transaction.
	StopStaleBuildsInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error)
	// FindTests returns the tests in a build.
	FindTests(ctx context.Context, buildID, repoName, orgID string, offset, limit int) ([]*Test, error)
	// FindLastBuildCommitSha returns commitID for last passed build
	FindLastBuildCommitSha(ctx context.Context, repoID, branch string) (string, error)
	// FindBuildMeta returns the meta for the builds
	FindBuildMeta(ctx context.Context, repoName, orgID, branch string) ([]*BuildExecutionStatus, error)
	// FindBaseBuildCommitForRebuildPostMerge return basecommit id for latest build on given commitID
	FindBaseBuildCommitForRebuildPostMerge(ctx context.Context, repoID,
		branch, commitID string) (string, error)
	// UpdateTier updates yml tier for given build within specified transaction
	UpdateTier(ctx context.Context, build *Build) error
	// CheckIfCommitIDBuilt checks if given commitID was ever built on TAS for given repo. Useful in deciding first ever commit
	CheckIfCommitIDBuilt(ctx context.Context, commitID, repoID string) (bool, error)
	// CheckIfPassedBuildExistsInTx checks if given repoID has any successful build
	// and executes the query in specified transaction
	CheckIfPassedBuildExistsInTx(ctx context.Context, tx *sqlx.Tx, repoID string) (bool, error)
	// CountPassedFailedBuildsForRepo returns count of builds which have passed or failed for a given repoID
	CountPassedFailedBuildsForRepo(ctx context.Context, repoID string) (int, error)
	// FindCommitDiff finds the diff of commits for a particular job
	FindCommitDiff(ctx context.Context, orgName, repoName, gitProvider, buildID string) (string, error)
	// UpdateBuildCache updates the build cache based on provided map
	UpdateBuildCache(ctx context.Context, buildID string, updateMap map[string]interface{}, skipKeycheck bool) error
	// IncrementCacheCount updates count of a field in buildCache
	IncrementCacheCount(ctx context.Context, buildID, field string, count int) error
	// FindLastBuildTier finds the tier of the last build that had run for a given repoID
	FindLastBuildTierInTx(ctx context.Context, tx *sqlx.Tx, repoID string) (Tier, error)
	// FindMttfAndMttr finds the mean time to failure and mean time to repair for a repo and branch
	FindMttfAndMttr(ctx context.Context, repoID, branchName string, startDate, endDate time.Time) (*MttfAndMttr, error)
	// FindJobsWithTestsStatus finds the tests status with in the `limit` builds
	FindJobsWithTestsStatus(ctx context.Context, repoName, orgID, branchName string, limit int) ([]*Build, error)
	// UpdateTestsRuntimes updates all_tests_runtime and impacted_tests_runtime columns
	UpdateTestsRuntimes(ctx context.Context, build *Build) error
	// FindBuildWiseTimeSaved returns build-wise time saved data in a given duration
	FindBuildWiseTimeSaved(ctx context.Context, repoID, branchName string, minCreatedAt time.Time,
		maxCreatedAt time.Time, limit, offset int) ([]*BuildWiseTimeSaved, error)
}

// BuildMonitor represents the build monitoring service.
type BuildMonitor interface {
	// FindAndUpdate find and update the build based on the task status.
	FindAndUpdate(ctx context.Context, orgID string, task *Task) error
	// FindTimeSavedData is a utility func which returns time-saved
	// data in a given build by not running unimpacted tests
	FindTimeSavedData(ctx context.Context, buildID, commitID, repoID string) (*TimeSavedData, error)
}

// BuildMeta contains additional info of the builds executed
type BuildMeta struct {
	Total       int     `json:"total_builds_executed"`
	Error       int     `json:"builds_error"`
	Initiating  int     `json:"builds_initiating"`
	Passed      int     `json:"builds_passed"`
	Running     int     `json:"builds_running"`
	Failed      int     `json:"builds_failed"`
	Aborted     int     `json:"builds_aborted"`
	AvgDuration float64 `json:"avg_builds_duration"`
}

// BuildService performs common functionalities related to Build.
type BuildService interface {
	// ParseGitEvent parses the git scm webhook event.
	ParseGitEvent(ctx context.Context, gitEvent *GitEvent, repo *Repository, buildTag BuildTag) (*ParserResponse, error)
	// CreateBuild creates a new build.
	CreateBuild(ctx context.Context, payload *ParserResponse, driver SCMDriver, isNewGitEvent bool) (int, error)
}

// BuildCache details related to a particular build, is cached in redis.
type BuildCache struct {
	TokenPath             string    `redis:"token_path"`
	InstallationTokenPath string    `redis:"installation_token_path"`
	SecretPath            string    `redis:"secret_path"`
	GitProvider           SCMDriver `redis:"git_provider"`
	RepoSlug              string    `redis:"repo_slug"`
	PullRequestNumber     int       `redis:"pull_request_number"`
	PayloadAddress        string    `redis:"payload_address"`
	BaseCommitID          string    `redis:"base_commit_id"`
	TargetCommitID        string    `redis:"target_commit_id"`
	RepoID                string    `redis:"repo_id"`
	CollectCoverage       bool      `redis:"collect_coverage"`
	Tier                  Tier      `redis:"tier"`
	BuildTag              BuildTag  `redis:"tag"`
	Branch                string    `redis:"branch"`
	TotalExecTasks        int       `redis:"total_exec_tasks"`
	ExecTasksPassed       int       `redis:"exec_tasks_passed"`
	ExecTasksFailed       int       `redis:"exec_tasks_failed"`
	ExecTasksError        int       `redis:"exec_tasks_error"`
	ExecTasksAborted      int       `redis:"exec_tasks_aborted"`
	FlakyConsecutiveRuns  int       `redis:"flaky_consecutive_runs"`
	ImpactedTestExists    bool      `redis:"impacted_test_exists"`
	ContainerImage        string    `redis:"container_image"`
	FailedTestCount       int       `redis:"failed_test_count"`
	FlakyTestCount        int       `redis:"flaky_test_count"`
	TotalFlakyTasks       int       `redis:"total_flaky_tasks"`
	FlakyTasksPassed      int       `redis:"flaky_tasks_passed"`
	FlakyTasksFailed      int       `redis:"flaky_tasks_failed"`
	FlakyTasksError       int       `redis:"flaky_tasks_error"`
	FlakyTasksAborted     int       `redis:"flaky_tasks_aborted"`
	TotalSubModule        int       `redis:"total_submodule"`
	ProcessedSubModule    int       `redis:"processed_submodule"`
	BuildFailedRemarks    string    `redis:"build_failed_reamrks"`
	Aborted               bool      `redis:"aborted"`
}

// BuildsCreated is wrapper for buildID and created_at date
type BuildsCreated struct {
	Created  int    `json:"created_at"`
	BuildID  string `json:"build_id"`
	BuildTag string `json:"build_tag"`
}
