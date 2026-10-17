package workflow

import (
	"context"
	"testing"
	"time"
)

type fakeTaskStore struct {
	runs      []WorkflowRun
	claimed   []string
	completed []string
	submitted []string
	progress  []string
	skipped   []string
}

func (s *fakeTaskStore) ListActiveRuns(context.Context, int) ([]WorkflowRun, error) {
	return s.runs, nil
}
func (s *fakeTaskStore) ClaimTask(_ context.Context, _ int64, key, _ string, _ time.Duration) (bool, error) {
	s.claimed = append(s.claimed, key)
	return true, nil
}
func (s *fakeTaskStore) CompleteTask(_ context.Context, _ int64, key, _, status, _ string) error {
	s.completed = append(s.completed, key+":"+status)
	return nil
}
func (s *fakeTaskStore) MarkTaskSubmitted(_ context.Context, _ int64, key, _ string, result ExecutionResult) error {
	s.submitted = append(s.submitted, key+":"+result.ProviderRunID)
	return nil
}
func (s *fakeTaskStore) CompleteSubmittedTask(_ context.Context, _ int64, key, status, _ string) error {
	s.completed = append(s.completed, key+":"+status)
	return nil
}
func (s *fakeTaskStore) AppendTaskLog(_ context.Context, _ int64, key, message string) error {
	s.progress = append(s.progress, key+":"+message)
	return nil
}
func (s *fakeTaskStore) SkipTask(_ context.Context, _ int64, key string) error {
	s.skipped = append(s.skipped, key)
	return nil
}

type fakeExecutor struct {
	typeName string
	err      error
}

func (e fakeExecutor) NodeType() string { return e.typeName }
func (e fakeExecutor) Execute(context.Context, ExecutionRequest) (ExecutionResult, error) {
	return ExecutionResult{Status: RunSucceeded}, e.err
}

type fakeAsyncExecutor struct{ result ExecutionResult }

func (e fakeAsyncExecutor) NodeType() string { return NodeBuild }
func (e fakeAsyncExecutor) Execute(context.Context, ExecutionRequest) (ExecutionResult, error) {
	return ExecutionResult{Status: RunRunning, ProviderRunID: "build:31"}, nil
}
func (e fakeAsyncExecutor) Reconcile(context.Context, ExecutionRequest) (ExecutionResult, error) {
	return e.result, nil
}

func TestWorkerExecutesReadyNodesAndRecordsResult(t *testing.T) {
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunPending, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunPending}}}}}
	registry, err := NewExecutorRegistry(fakeExecutor{typeName: NodeBuild})
	if err != nil {
		t.Fatal(err)
	}
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.claimed) != 1 || len(store.completed) != 1 || store.completed[0] != "build:succeeded" {
		t.Fatalf("claimed=%v completed=%v", store.claimed, store.completed)
	}
}

func TestWorkerFailsTaskWithoutExecutor(t *testing.T) {
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunPending, Tasks: []TaskRun{{NodeKey: "deploy", NodeType: NodeYAMLDeploy, Status: RunPending}}}}}
	registry, _ := NewExecutorRegistry()
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.completed) != 1 || store.completed[0] != "deploy:failed" {
		t.Fatalf("completed=%v", store.completed)
	}
}

func TestWorkerReconcilesSubmittedTask(t *testing.T) {
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{{
		NodeKey: "build", NodeType: NodeBuild, Status: RunRunning, ProviderRunID: "build:31",
	}}}}}
	registry, _ := NewExecutorRegistry(fakeAsyncExecutor{result: ExecutionResult{Status: RunSucceeded}})
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.completed) != 1 || store.completed[0] != "build:succeeded" {
		t.Fatalf("completed=%v", store.completed)
	}
}

func TestWorkerPublishesRunningTaskDiagnostic(t *testing.T) {
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{{
		NodeKey: "deploy", NodeType: NodeBuild, Status: RunRunning, ProviderRunID: "build:31",
	}}}}}
	registry, _ := NewExecutorRegistry(fakeAsyncExecutor{result: ExecutionResult{Status: RunRunning, LogText: "pod api phase=Pending"}})
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.progress) != 1 || store.progress[0] != "deploy:pod api phase=Pending" {
		t.Fatalf("progress=%v", store.progress)
	}
}

func TestWorkerReclaimsExpiredUnsubmittedTask(t *testing.T) {
	expired := time.Now().Add(-time.Minute)
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{{
		NodeKey: "deploy", NodeType: NodeYAMLDeploy, Status: RunRunning, LockedBy: "dead-worker", LeaseUntil: &expired,
	}}}}}
	registry, _ := NewExecutorRegistry(fakeExecutor{typeName: NodeYAMLDeploy})
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker-2"); err != nil {
		t.Fatal(err)
	}
	if len(store.claimed) != 1 || store.claimed[0] != "deploy" || len(store.completed) != 1 {
		t.Fatalf("claimed=%v completed=%v", store.claimed, store.completed)
	}
}

func TestWorkerSkipsEntireDeploymentPhaseAfterBatchBuildFailure(t *testing.T) {
	server := mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 1})})
	gateway := mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 2})})
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{
		{NodeKey: "build-server", NodeType: NodeBuild, Status: RunFailed},
		{NodeKey: "deploy-server", NodeType: NodeYAMLDeploy, NodeSnapshot: server, Status: RunPending, DependsOn: []string{"build-server"}},
		{NodeKey: "build-gateway", NodeType: NodeBuild, Status: RunSucceeded},
		{NodeKey: "deploy-gateway", NodeType: NodeYAMLDeploy, NodeSnapshot: gateway, Status: RunPending, DependsOn: []string{"build-gateway"}},
	}}}}
	registry, _ := NewExecutorRegistry(fakeExecutor{typeName: NodeYAMLDeploy})
	worker := NewWorker(store, registry, time.Minute, 10)
	if err := worker.Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.skipped) != 2 || store.skipped[0] != "deploy-server" || store.skipped[1] != "deploy-gateway" || len(store.claimed) != 0 {
		t.Fatalf("skipped=%v claimed=%v", store.skipped, store.claimed)
	}
	store.runs[0].Tasks[0].Status = RunSucceeded
	store.runs[0].Tasks[1].Status = RunFailed
	if err := worker.Tick(context.Background(), "worker-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.claimed) != 1 || store.claimed[0] != "deploy-gateway" {
		t.Fatalf("gateway was not submitted after server deployment failure: %v", store.claimed)
	}
}
