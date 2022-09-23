package ws

import (
	"encoding/json"

	"github.com/LambdaTest/neuron/pkg/core"
)

// constant related to ws
const (
	AuthenticationPassedMsg = "Authentication passed"
)

// wsError create error message
func wsError(content string) core.Message {
	return core.Message{
		Type:    core.MsgError,
		Content: []byte(content),
		Success: false,
	}
}

// wsInfo create info message
func wsInfo(content string) core.Message {
	return core.Message{
		Type:    core.MsgInfo,
		Content: []byte(content),
		Success: true,
	}
}

// CreateTaskMessage creates new task message
func CreateTaskMessage(runner *core.RunnerOptions) core.Message {
	runnerJSON, err := json.Marshal(runner)
	if err != nil {
		return core.Message{}
	}
	return core.Message{
		Type:    core.MsgTask,
		Content: runnerJSON,
		Success: true,
	}
}
