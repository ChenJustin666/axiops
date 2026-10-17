package workflow

import (
	"context"
	"testing"
)

func TestIsolatedTestBindsAncestorBuildAndUsesDurableQueue(t *testing.T) {
	queue := &fakeBuildQueue{}
	build := TaskRun{NodeKey: "build", NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:42", NodeSnapshot: mustJSON(Node{Config: mustJSON(BuildNodeConfig{ServiceID: 9})})}
	parallel := build
	parallel.NodeKey = "parallel"
	parallel.ProviderRunID = "build:43"
	task := TaskRun{NodeKey: "test", DependsOn: []string{"build"}, NodeSnapshot: mustJSON(Node{TimeoutSeconds: 90, Config: mustJSON(map[string]interface{}{"service_id": 9, "image": "golang:1.24", "command": "go test ./...", "working_dir": "module"})})}
	result, err := NewIsolatedTestExecutor(queue).Execute(context.Background(), ExecutionRequest{Run: WorkflowRun{ID: 1, Tasks: []TaskRun{build, parallel}}, Task: task})
	if err != nil || result.Status != RunRunning || queue.triggered.Test == nil || queue.triggered.Test.SourceBuildID != 42 || queue.triggered.Test.TimeoutSeconds != 90 {
		t.Fatalf("%+v %+v %v", result, queue.triggered, err)
	}
}
func TestIsolatedTestRejectsMissingImageAndEscapingDirectory(t *testing.T) {
	for _, spec := range []map[string]interface{}{
		{"service_id": 9, "command": "test"},
		{"service_id": 9, "image": "golang:1.24", "command": "test", "working_dir": "../other"},
		{"service_id": 9, "image": "golang:1.24", "command": "test", "working_dir": "/etc"},
	} {
		q := &fakeBuildQueue{}
		_, err := NewIsolatedTestExecutor(q).Execute(context.Background(), ExecutionRequest{Task: TaskRun{NodeSnapshot: mustJSON(Node{Config: mustJSON(spec)})}})
		if err == nil || q.triggered.ServiceID != 0 {
			t.Fatal("invalid test launched work")
		}
	}
}
