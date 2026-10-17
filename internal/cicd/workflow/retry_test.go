package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

type retryStore struct {
	fakeTaskStore
	retries int
}

func (s *retryStore) ScheduleTaskRetry(context.Context, WorkflowRun, TaskRun, string) (bool, error) {
	s.retries++
	return true, nil
}
func TestRetryKeepsWorkflowOpenOnlyForSafeBuildFailure(t *testing.T) {
	for _, retryable := range []bool{false, true} {
		task := TaskRun{NodeKey: "build", NodeType: NodeBuild, Status: RunRunning, ProviderRunID: "build:31", NodeSnapshot: mustJSON(Node{RetryLimit: 1})}
		store := &retryStore{fakeTaskStore: fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{task}}}}}
		registry, _ := NewExecutorRegistry(fakeAsyncExecutor{ExecutionResult{Status: RunFailed, Retryable: retryable}})
		if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err != nil {
			t.Fatal(err)
		}
		if retryable && (store.retries != 1 || len(store.completed) != 0) {
			t.Fatal("retry prematurely completed workflow")
		}
		if !retryable && (store.retries != 0 || len(store.completed) != 1) {
			t.Fatal("unsafe retry accepted")
		}
	}
}
func TestRetryGenerationIsDistinctFromLeaseAttemptAndBounded(t *testing.T) {
	task := TaskRun{NodeKey: "build", NodeType: NodeBuild, NodeSnapshot: mustJSON(Node{RetryLimit: 1, TimeoutSeconds: 60}), Attempt: 9}
	first := workflowBuildKey(WorkflowRun{ID: 1}, task)
	task.RetryCount = 1
	if first == workflowBuildKey(WorkflowRun{ID: 1}, task) || retryAllowed(task) {
		t.Fatal("retry generation or limit ignored")
	}
	task.RetryCount = 0
	start := time.Now().Add(-time.Hour)
	task.StartedAt = &start
	if retryAllowed(task) {
		t.Fatal("timed-out node retried")
	}
	if retryDelay(10) > 3*time.Minute {
		t.Fatal("retry backoff unbounded")
	}
}
func TestRetryBackoffDoesNotClaimPendingTask(t *testing.T) {
	later := time.Now().Add(time.Minute)
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunRunning, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunPending, NextRetryAt: &later}}}}}
	registry, _ := NewExecutorRegistry(fakeExecutor{typeName: NodeBuild})
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	if len(store.claimed) != 0 {
		t.Fatal("retry started before its backoff")
	}
}

type recoveringQueue struct {
	fakeBuildQueue
	found int64
	err   error
}

func (q *recoveringQueue) FindWorkflowBuild(context.Context, int64, string) (int64, error) {
	return q.found, q.err
}
func TestRecoverFindsCommittedBuildEvenAfterNodeDeadline(t *testing.T) {
	q := &recoveringQueue{found: 31}
	start := time.Now().Add(-time.Hour)
	task := TaskRun{NodeKey: "build", StartedAt: &start, NodeSnapshot: mustJSON(Node{TimeoutSeconds: 1, Config: mustJSON(BuildNodeConfig{ServiceID: 9})})}
	result, err := NewWorker(&fakeTaskStore{}, nil, time.Minute, 10).executeWithLease(context.Background(), "w", NewBuildExecutor(q), ExecutionRequest{Task: task})
	if err != nil || result.ProviderRunID != "build:31" || q.triggered.ServiceID != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
func TestUncertainSubmissionIsNotMarkedFailed(t *testing.T) {
	store := &fakeTaskStore{runs: []WorkflowRun{{ID: 1, Status: RunPending, Tasks: []TaskRun{{NodeKey: "build", NodeType: NodeBuild, Status: RunPending}}}}}
	registry, _ := NewExecutorRegistry(fakeExecutor{typeName: NodeBuild, err: ErrSubmissionUncertain})
	err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "w")
	if !errors.Is(err, ErrSubmissionUncertain) || len(store.completed) != 0 {
		t.Fatal("ambiguous submission treated as terminal")
	}
}
func TestBuildTimeoutWaitsForProviderCancellation(t *testing.T) {
	q := &cancelBuildQueue{fakeBuildQueue: fakeBuildQueue{state: BuildState{Status: "running"}}}
	start := time.Now().Add(-time.Hour)
	req := ExecutionRequest{Task: TaskRun{NodeKey: "b", StartedAt: &start, ProviderRunID: "build:31", NodeSnapshot: mustJSON(Node{TimeoutSeconds: 1, Config: mustJSON(BuildNodeConfig{ServiceID: 9})})}}
	result, err := NewBuildExecutor(q).Reconcile(context.Background(), req)
	if err != nil || result.Status != RunRunning || len(q.requests) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	q.state.Status = "cancelled"
	result, err = NewBuildExecutor(q).Reconcile(context.Background(), req)
	if err != nil || result.Status != RunFailed || result.Retryable {
		t.Fatal("timeout must fail after confirmed stop")
	}
}
