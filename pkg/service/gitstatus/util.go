package gitstatus

import (
	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/drone/go-scm/scm"
)

func createStatus(status core.BuildStatus) scm.State {
	switch status {
	case core.BuildInitiating:
		return scm.StatePending
	case core.BuildRunning:
		return scm.StateRunning
	case core.BuildPassed:
		return scm.StateSuccess
	case core.BuildFailed:
		return scm.StateFailure
	case core.BuildAborted:
		return scm.StateCanceled
	case core.BuildError:
		return scm.StateError
	default:
		return scm.StateUnknown
	}
}

func createDesc(status core.BuildStatus) string {
	switch status {
	case core.BuildInitiating:
		return core.BuildPendingDesc
	case core.BuildRunning:
		return core.BuildRunningDesc
	case core.BuildPassed:
		return core.BuildPassedDesc
	case core.BuildFailed:
		return core.BuildFailedDesc
	case core.BuildAborted:
		return core.BuildAbortedDesc
	case core.BuildError:
		return core.BuildErrorDesc
	default:
		return core.BuildUnknownDesc
	}
}
