package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

type cancellingStore struct {
	fakeTaskStore
	finished  []int64
	stopped   []string
	recovered string
}

func (s *cancellingStore) FinishCancellation(_ context.Context, id int64) error {
	s.finished = append(s.finished, id)
	return nil
}
func (s *cancellingStore) CompleteCancellingTask(_ context.Context, _ int64, key string) error {
	s.stopped = append(s.stopped, key)
	return nil
}
func (s *cancellingStore) RecoverTaskSubmission(context.Context, WorkflowRun, TaskRun) (string, error) {
	return s.recovered, nil
}

type cancelExecutor struct {
	requests int
	state    string
	err      error
}

func (e *cancelExecutor) NodeType() string { return NodeBuild }
func (e *cancelExecutor) Execute(context.Context, ExecutionRequest) (ExecutionResult, error) {
	panic("cancel must never submit")
}
func (e *cancelExecutor) Cancel(context.Context, ExecutionRequest) (ExecutionResult, error) {
	e.requests++
	return ExecutionResult{Status: e.state}, e.err
}

func TestCancellationWaitsForProviderAndDoesNotStartPendingNodes(t *testing.T) {
	store := &cancellingStore{fakeTaskStore: fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunCancelling, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunRunning, ProviderRunID: "build:31"}, {NodeKey: "deploy", NodeType: NodeYAMLDeploy, Status: RunPending}}}}}}
	executor := &cancelExecutor{state: RunRunning}
	registry, _ := NewExecutorRegistry(executor)
	worker := NewWorker(store, registry, time.Minute, 10)
	if err := worker.Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	if len(store.stopped) != 0 || len(store.claimed) != 0 || executor.requests != 1 {
		t.Fatalf("premature cancellation %+v", store)
	}
	executor.state = RunCancelled
	if err := worker.Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	if len(store.stopped) != 1 || len(store.finished) != 2 {
		t.Fatalf("not finalized %+v", store)
	}
}
func TestCancelFailureDoesNotBlockAnotherRun(t *testing.T) {
	store := &cancellingStore{fakeTaskStore: fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunCancelling, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunRunning}}}, {ID: 2, Status: RunPending, Tasks: []TaskRun{{NodeKey: "notify", NodeType: NodeNotify, Status: RunPending}}}}}}
	registry, _ := NewExecutorRegistry(&cancelExecutor{err: errors.New("network")}, fakeExecutor{typeName: NodeNotify})
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err == nil {
		t.Fatal("lost diagnostic")
	}
	if len(store.completed) != 1 || store.completed[0] != "notify:succeeded" {
		t.Fatalf("other run blocked: %+v", store)
	}
}
func TestCancelWaitsForLiveSubmissionLease(t *testing.T) {
	until := time.Now().Add(time.Minute)
	store := &cancellingStore{fakeTaskStore: fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunCancelling, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunRunning, LockedBy: "submitting", LeaseUntil: &until}}}}}}
	executor := &cancelExecutor{state: RunCancelled}
	registry, _ := NewExecutorRegistry(executor)
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	if executor.requests != 0 {
		t.Fatal("cancel raced a live submission")
	}
}

type cancelBuildQueue struct {
	fakeBuildQueue
	requests []BuildCancellationRequest
}

func (q *cancelBuildQueue) CancelWorkflowBuild(_ context.Context, r BuildCancellationRequest) (BuildState, error) {
	q.requests = append(q.requests, r)
	return q.state, nil
}
func TestBuildCancellationRecoversByKeyWithoutSubmitting(t *testing.T) {
	queue := &cancelBuildQueue{fakeBuildQueue: fakeBuildQueue{state: BuildState{Status: "cancelled"}}}
	task := TaskRun{NodeKey: "build", NodeSnapshot: mustJSON(Node{Config: mustJSON(BuildNodeConfig{ServiceID: 9})})}
	result, err := NewBuildExecutor(queue).Cancel(context.Background(), ExecutionRequest{Run: WorkflowRun{ID: 4, ProjectID: 3}, Task: task})
	if err != nil || result.Status != RunCancelled || len(queue.requests) != 1 || queue.requests[0].TriggerKey != "workflow:4:build" || queue.triggered.ServiceID != 0 {
		t.Fatalf("result=%+v error=%v queue=%+v", result, err, queue)
	}
}
func TestBuildSubmissionKeySurvivesLeaseReclaim(t *testing.T) {
	queue := &fakeBuildQueue{}
	request := ExecutionRequest{Run: WorkflowRun{ID: 7}, Task: TaskRun{NodeKey: "build", Attempt: 1, NodeSnapshot: mustJSON(Node{Config: mustJSON(BuildNodeConfig{ServiceID: 9})})}}
	executor := NewBuildExecutor(queue)
	if _, err := executor.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	first := queue.triggered.TriggerKey
	request.Task.Attempt = 8
	if _, err := executor.Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if first == "" || first != queue.triggered.TriggerKey {
		t.Fatal("lease reclaim changed submission identity")
	}
}

type leaseTestStore struct {
	fakeTaskStore
	renewals int
	release  bool
	state    string
	err      error
}

func (s *leaseTestStore) RenewTaskLease(context.Context, int64, string, string, time.Duration) (string, error) {
	s.renewals++
	return s.state, s.err
}
func (s *leaseTestStore) ReleaseTaskLease(context.Context, int64, string, string) error {
	s.release = true
	return nil
}

type waitingExecutor struct{}

func (waitingExecutor) NodeType() string { return NodeTest }
func (waitingExecutor) Execute(ctx context.Context, _ ExecutionRequest) (ExecutionResult, error) {
	<-ctx.Done()
	return ExecutionResult{}, ctx.Err()
}
func TestLeaseCheckPreventsExecutionAfterCancel(t *testing.T) {
	s := &leaseTestStore{state: RunCancelling}
	_, err := NewWorker(s, nil, time.Second, 10).executeWithLease(context.Background(), "w", waitingExecutor{}, ExecutionRequest{})
	if !errors.Is(err, errTaskCancelled) {
		t.Fatalf("err=%v", err)
	}
}
func TestSynchronousNodeHonorsTimeout(t *testing.T) {
	start := time.Now().Add(-950 * time.Millisecond)
	_, err := NewWorker(&fakeTaskStore{}, nil, time.Minute, 10).executeWithLease(context.Background(), "w", waitingExecutor{}, ExecutionRequest{Task: TaskRun{StartedAt: &start, NodeSnapshot: mustJSON(Node{TimeoutSeconds: 1})}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v", err)
	}
}
