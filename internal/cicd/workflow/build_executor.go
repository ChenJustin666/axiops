package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type BuildTriggerRequest struct {
	Test           *TestBuildSpec
	RetryOfBuildID int64
	WorkflowRunID  int64
	NodeKey        string
	TriggerKey     string
	ServiceID      int64
	Branch         string
	CommitID       string
}

// A database/network failure may hide a committed submission. Recover its
// durable key before deciding whether to create work or mark the node failed.
var ErrSubmissionUncertain = errors.New("submission outcome is uncertain")

func (e *BuildExecutor) Recover(ctx context.Context, request ExecutionRequest) (ExecutionResult, bool, error) {
	if e == nil || e.queue == nil {
		return ExecutionResult{}, false, fmt.Errorf("build queue is not configured")
	}
	queue, ok := e.queue.(interface {
		FindWorkflowBuild(context.Context, int64, string) (int64, error)
	})
	if !ok {
		return ExecutionResult{}, false, nil
	}
	config, err := buildConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, false, err
	}
	id, err := queue.FindWorkflowBuild(ctx, config.ServiceID, workflowBuildKey(request.Run, request.Task))
	if err != nil {
		return ExecutionResult{}, false, fmt.Errorf("%w: %v", ErrSubmissionUncertain, err)
	}
	if id == 0 {
		return ExecutionResult{}, false, nil
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: fmt.Sprintf("build:%d", id), LogRef: fmt.Sprintf("/api/v1/builds/%d/logs", id)}, true, nil
}

type BuildCancellationRequest struct {
	ProjectID  int64
	ServiceID  int64
	BuildID    int64
	TriggerKey string
}

func workflowBuildKey(run WorkflowRun, task TaskRun) string {
	if task.RetryCount > 0 {
		return fmt.Sprintf("workflow:%d:%s:retry:%d", run.ID, task.NodeKey, task.RetryCount)
	}
	return fmt.Sprintf("workflow:%d:%s", run.ID, task.NodeKey)
}

func (e *BuildExecutor) Cancel(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.queue == nil {
		return ExecutionResult{}, fmt.Errorf("build queue is not configured")
	}
	queue, ok := e.queue.(interface {
		CancelWorkflowBuild(context.Context, BuildCancellationRequest) (BuildState, error)
	})
	if !ok {
		return ExecutionResult{}, fmt.Errorf("build queue cancellation is unavailable")
	}
	config, err := buildConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	var id int64
	if request.Task.ProviderRunID != "" {
		id, err = parseBuildRunID(request.Task.ProviderRunID)
		if err != nil {
			return ExecutionResult{}, err
		}
	}
	state, err := queue.CancelWorkflowBuild(ctx, BuildCancellationRequest{ProjectID: request.Run.ProjectID, ServiceID: config.ServiceID, BuildID: id, TriggerKey: workflowBuildKey(request.Run, request.Task)})
	if err != nil {
		return ExecutionResult{}, err
	}
	if state.Status == "cancelled" || state.Status == "success" || state.Status == "failed" || state.Status == "timeout" {
		return ExecutionResult{Status: RunCancelled, LogText: "[cancel] 构建已停止；已有日志和产物记录保留"}, nil
	}
	return ExecutionResult{Status: RunRunning, LogText: "[cancel] 已请求停止构建，等待执行资源退出"}, nil
}

type BuildState struct {
	Retryable    bool
	Status       string
	ErrorMessage string
}

type BuildQueue interface {
	TriggerBuild(context.Context, BuildTriggerRequest, int64) (int64, error)
	BuildState(context.Context, int64) (BuildState, error)
}

type BuildNodeConfig struct {
	ServiceID int64  `json:"service_id"`
	Branch    string `json:"branch"`
	CommitID  string `json:"commit_id"`
}

type BuildExecutor struct{ queue BuildQueue }

func NewBuildExecutor(queue BuildQueue) *BuildExecutor { return &BuildExecutor{queue: queue} }

func (e *BuildExecutor) NodeType() string { return NodeBuild }

func (e *BuildExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.queue == nil {
		return ExecutionResult{}, fmt.Errorf("build queue is not configured")
	}
	config, err := buildConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	var retryOf int64
	if request.Task.RetryCount > 0 {
		if len(request.Task.RetryHistory) == 0 {
			return ExecutionResult{}, fmt.Errorf("retry history is missing")
		}
		retryOf, err = parseBuildRunID(request.Task.RetryHistory[len(request.Task.RetryHistory)-1].ProviderRunID)
		if err != nil {
			return ExecutionResult{}, err
		}
	}
	buildID, err := e.queue.TriggerBuild(ctx, BuildTriggerRequest{RetryOfBuildID: retryOf, WorkflowRunID: request.Run.ID, NodeKey: request.Task.NodeKey, ServiceID: config.ServiceID, Branch: config.Branch, CommitID: config.CommitID, TriggerKey: workflowBuildKey(request.Run, request.Task)}, request.Run.TriggerUserID)
	if err != nil {
		return ExecutionResult{}, err
	}
	// The existing build queue owns kube_job submission and retries. The
	// workflow task remains running until Reconcile observes a terminal state.
	return ExecutionResult{Status: RunRunning, ProviderRunID: fmt.Sprintf("build:%d", buildID), LogRef: fmt.Sprintf("/api/v1/builds/%d/logs", buildID)}, nil
}

func (e *BuildExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	buildID, err := parseBuildRunID(request.Task.ProviderRunID)
	if err != nil {
		return ExecutionResult{}, err
	}
	state, err := e.queue.BuildState(ctx, buildID)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case "success":
		return ExecutionResult{Status: RunSucceeded}, nil
	case "failed", "timeout":
		result := ExecutionResult{Status: RunFailed, ErrorMessage: state.ErrorMessage, Retryable: state.Retryable}
		if !state.Retryable && retryAllowed(request.Task) {
			result.LogText = "[retry] 未取得完整 Commit 或构建配置快照，停止自动重试，避免使用变化后的代码"
		}
		return result, nil
	case "cancelled":
		if buildNodeTimedOut(request.Task) {
			return ExecutionResult{Status: RunFailed, ErrorMessage: "构建节点已超时，执行资源已停止"}, nil
		}
		return ExecutionResult{Status: RunCancelled}, nil
	case "queued", "dispatching", "running", "cancelling":
		if buildNodeTimedOut(request.Task) {
			_, err := e.Cancel(ctx, request)
			return ExecutionResult{Status: RunRunning, LogText: "[timeout] 构建节点已超时，等待取消构建并保存日志"}, err
		}
		return ExecutionResult{Status: RunRunning}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported build status %q", state.Status)
	}
}

func buildNodeTimedOut(task TaskRun) bool {
	var node Node
	return task.StartedAt != nil && json.Unmarshal(task.NodeSnapshot, &node) == nil && node.TimeoutSeconds > 0 && time.Since(*task.StartedAt) >= time.Duration(node.TimeoutSeconds)*time.Second
}

func buildConfig(snapshot json.RawMessage) (BuildNodeConfig, error) {
	var node Node
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &node) != nil {
		return BuildNodeConfig{}, fmt.Errorf("invalid build node snapshot")
	}
	var config BuildNodeConfig
	if len(node.Config) == 0 || json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 {
		return BuildNodeConfig{}, fmt.Errorf("build node requires service_id")
	}
	return config, nil
}

func parseBuildRunID(value string) (int64, error) {
	raw, ok := strings.CutPrefix(value, "build:")
	if !ok {
		return 0, fmt.Errorf("invalid build provider run id %q", value)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid build provider run id %q", value)
	}
	return id, nil
}
