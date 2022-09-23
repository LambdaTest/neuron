package synapse

import "time"

// Websocket connection options
const (
	PingInterval           = 5 * time.Second
	PongWait               = 10 * time.Second
	WsWait                 = 10 * time.Second
	DeRegisterWaitDuration = 10 * time.Minute
	MaxMessageSize         = 4096
)

// const related to task mode
const (
	TestExecutionMode = "testExecution"
	CoverageMode      = "coverageMode"
	BuildID           = "build-id"
	Repo              = "repo"
	Mode              = "mode"
	JobID             = "job-id"
	ID                = "id"
)
