package core

import "context"

// Scheduler will runs the tasks in regular intervals.
type Scheduler interface {
	//  Run starts the scheduler on startup.
	Run(ctx context.Context)
}

// SchedulerManager manages operations for stale builds
type SchedulerManager interface {
	//  StopStaleBuilds marks stale builds as error
	StopStaleBuilds(ctx context.Context) error
}
