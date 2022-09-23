package core

import "context"

// TaskRunner is the interface which defines the implementation intiating task based on runner type
type TaskRunner interface {
	// ScheduleTask schedules task on synapse or k8s depending on the runner type
	ScheduleTask(ctx context.Context, r *RunnerOptions, buildID, jobID, taskID, customDockerImage string) <-chan error
}
