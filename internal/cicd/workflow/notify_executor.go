package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type notifyNodeConfig struct {
	Message string `json:"message,omitempty"`
}

// NotifyExecutor marks the workflow notification stage complete. The final
// run state and its subscribed deliveries are committed atomically by the
// workflow repository, then sent asynchronously by the notification worker.
type NotifyExecutor struct{}

func NewNotifyExecutor() *NotifyExecutor { return &NotifyExecutor{} }

func (e *NotifyExecutor) NodeType() string { return NodeNotify }

func (e *NotifyExecutor) Execute(_ context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if len(request.Task.NodeSnapshot) == 0 {
		return ExecutionResult{}, fmt.Errorf("invalid notify node snapshot")
	}
	var node struct {
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(request.Task.NodeSnapshot, &node); err != nil {
		return ExecutionResult{}, fmt.Errorf("invalid notify node snapshot")
	}
	var config notifyNodeConfig
	if len(node.Config) > 0 {
		if err := json.Unmarshal(node.Config, &config); err != nil {
			return ExecutionResult{}, fmt.Errorf("invalid notify node configuration")
		}
	}
	if len(strings.TrimSpace(config.Message)) > 1024 {
		return ExecutionResult{}, fmt.Errorf("notify message cannot exceed 1024 characters")
	}
	return ExecutionResult{Status: RunSucceeded}, nil
}
