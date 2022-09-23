package core

import (
	"time"

	"context"

	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4/zero"
)

// Status specifies the status of a Task
type Status string

// Task Status values.
const (
	TaskRunning    Status = "running"
	TaskFailed     Status = "failed"
	TaskPassed     Status = "passed"
	TaskInitiating Status = "initiating"
	TaskAborted    Status = "aborted"
	TaskError      Status = "error"
	TaskSkipped    Status = "skipped"
)

// TaskType specifies the type of a Task
type TaskType string

// Task Type values.
const (
	DiscoveryTask TaskType = "discover"
	ExecutionTask TaskType = "execute"
	FlakyTask     TaskType = "flaky"
)

// Task represents the containers spawned for each build .
type Task struct {
	ID             string                  `db:"id" json:"task_id"`
	RepoID         string                  `db:"repo_id" json:"-"`
	BuildID        string                  `db:"build_id" json:"-"`
	Created        time.Time               `db:"created_at" json:"created_at,omitempty"`
	Updated        time.Time               `db:"updated_at" json:"-"`
	Status         Status                  `db:"status" json:"status,omitempty"`
	StartTime      zero.Time               `db:"start_time" json:"start_time,omitempty"`
	EndTime        zero.Time               `db:"end_time" json:"end_time,omitempty"`
	Tier           Tier                    `db:"tier" json:"tier,omitempty"`
	Type           TaskType                `db:"type" json:"type"`
	TestLocators   zero.String             `db:"test_locators" json:"-"`
	Remark         zero.String             `json:"remark" db:"remark"`
	ContainerImage string                  `db:"container_image" json:"container_image,omitempty"`
	SubModule      string                  `db:"submodule" json:"submodule,omitempty"`
	Label          string                  `json:"label"`
	TestsMeta      *ExecutionMeta          `json:"execution_meta,omitempty"`
	FlakyMeta      *FlakyExecutionMetadata `json:"flaky_execution_meta,omitempty"`
}

// TaskMeta contains additional info of the tasks executed
type TaskMeta struct {
	TotalBuilds     int     `json:"total_builds_executed"`
	Initiating      int     `json:"tasks_initiating"`
	Running         int     `json:"tasks_running"`
	Failed          int     `json:"tasks_failed"`
	Aborted         int     `json:"tasks_aborted"`
	AvgTaskDuration float64 `json:"avg_task_duration"`
	Passed          int     `json:"passed"`
	Error           int     `json:"error"`
	Total           int     `json:"total"`
	Skipped         int     `json:"skipped"`
}

// TaskStore defines datastore operation for working with Task
type TaskStore interface {
	// Create persists a new task in the datastore.
	Create(ctx context.Context, task ...*Task) error
	// CreateInTx persists a new task in the datastore and executes the statements within the specified transaction
	CreateInTx(ctx context.Context, tx *sqlx.Tx, task ...*Task) error
	// Find the task in datastore by id.
	Find(ctx context.Context, taskID string) (*Task, error)
	// UpdateInTx  persists changes to the task in the datastore and executes the statements within the specified transaction.
	UpdateInTx(ctx context.Context, tx *sqlx.Tx, task *Task) error
	// UpdateByBuildID update the status of all tasks for a build.
	UpdateByBuildID(ctx context.Context, status Status, taskType TaskType, remark, buildID string) error
	// UpdateByBuildIDInTx update the status of all tasks for a build and executes the statements within the specified transaction
	UpdateByBuildIDInTx(ctx context.Context, tx *sqlx.Tx, status Status, remark, buildID string) error
	// StopStaleTasksInTx marks the tasks as error after timeout within specified transaction.
	StopStaleTasksInTx(ctx context.Context, tx *sqlx.Tx, timeout time.Duration) (int64, error)
	// FetchTask fetches all the task which are there for a particular buildID
	FetchTask(ctx context.Context, buildID string, offset, limit int) ([]*Task, error)
	// FetchTaskHavingStatus fetches all the tasks which are in given state for a particular buildID
	FetchTaskHavingStatus(ctx context.Context, buildID string, status Status) ([]*Task, error)
}
