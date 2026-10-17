package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

type TestBuildSpec struct {
	SourceBuildID  int64  `json:"-"`
	Image          string `json:"image"`
	Command        string `json:"command"`
	WorkingDir     string `json:"working_dir,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// Test jobs share the durable build queue, isolated checkout, logging,
// cancellation, capacity limits and cleanup. They do not push an image.
type IsolatedTestExecutor struct{ *BuildExecutor }

func NewIsolatedTestExecutor(queue BuildQueue) *IsolatedTestExecutor {
	return &IsolatedTestExecutor{NewBuildExecutor(queue)}
}
func (*IsolatedTestExecutor) NodeType() string { return NodeTest }
func (e *IsolatedTestExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	var node Node
	if json.Unmarshal(request.Task.NodeSnapshot, &node) != nil {
		return ExecutionResult{}, fmt.Errorf("invalid test node snapshot")
	}
	var config struct {
		BuildNodeConfig
		TestBuildSpec
	}
	if json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 || strings.TrimSpace(config.Image) == "" || strings.TrimSpace(config.Command) == "" || len(config.Command) > 16384 {
		return ExecutionResult{}, fmt.Errorf("测试节点需要服务、执行镜像和测试命令；命令在独立 Job 中运行")
	}
	dir := path.Clean(config.WorkingDir)
	if path.IsAbs(dir) || dir == ".." || strings.HasPrefix(dir, "../") || strings.ContainsAny(dir, "\\\x00\r\n") {
		return ExecutionResult{}, fmt.Errorf("测试目录必须是仓库内相对路径")
	}
	config.WorkingDir = dir
	config.TimeoutSeconds = node.TimeoutSeconds
	// Only ancestor builds can select the code tested; a parallel build for the
	// same service must not silently redirect this test's source.
	ancestors := make(map[string]bool)
	var visit func([]string)
	visit = func(keys []string) {
		for _, key := range keys {
			if !ancestors[key] {
				ancestors[key] = true
				if task, ok := taskByKey(request.Run.Tasks, key); ok {
					visit(task.DependsOn)
				}
			}
		}
	}
	visit(request.Task.DependsOn)
	var sourceTasks []TaskRun
	for _, task := range request.Run.Tasks {
		if ancestors[task.NodeKey] {
			sourceTasks = append(sourceTasks, task)
		}
	}
	var err error
	config.SourceBuildID, err = matchingBuildID(sourceTasks, config.ServiceID)
	if err != nil {
		return ExecutionResult{}, err
	}
	id, err := e.queue.TriggerBuild(ctx, BuildTriggerRequest{WorkflowRunID: request.Run.ID, NodeKey: request.Task.NodeKey, TriggerKey: workflowBuildKey(request.Run, request.Task), ServiceID: config.ServiceID, Branch: config.Branch, CommitID: config.CommitID, Test: &config.TestBuildSpec}, request.Run.TriggerUserID)
	if err != nil {
		return ExecutionResult{}, err
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: fmt.Sprintf("build:%d", id), LogRef: fmt.Sprintf("/api/v1/builds/%d/logs", id)}, nil
}
