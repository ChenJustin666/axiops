package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeBuildQueue struct {
	triggered BuildTriggerRequest
	state     BuildState
}

func (q *fakeBuildQueue) TriggerBuild(_ context.Context, req BuildTriggerRequest, _ int64) (int64, error) {
	q.triggered = req
	return 31, nil
}
func (q *fakeBuildQueue) BuildState(context.Context, int64) (BuildState, error) { return q.state, nil }

func TestBuildExecutorSubmitsExistingBuildQueue(t *testing.T) {
	queue := &fakeBuildQueue{}
	nodeSnapshot, _ := json.Marshal(Node{Key: "build", Name: "Build", Type: NodeBuild, Config: json.RawMessage(`{"service_id":9,"branch":"main"}`)})
	result, err := NewBuildExecutor(queue).Execute(context.Background(), ExecutionRequest{
		Run:  WorkflowRun{TriggerUserID: 7},
		Task: TaskRun{NodeSnapshot: nodeSnapshot},
	})
	if err != nil || result.Status != RunRunning || result.ProviderRunID != "build:31" || queue.triggered.ServiceID != 9 {
		t.Fatalf("result=%+v request=%+v err=%v", result, queue.triggered, err)
	}
}

func TestBuildExecutorReconcilesTerminalState(t *testing.T) {
	queue := &fakeBuildQueue{state: BuildState{Status: "success"}}
	result, err := NewBuildExecutor(queue).Reconcile(context.Background(), ExecutionRequest{Task: TaskRun{ProviderRunID: "build:31"}})
	if err != nil || result.Status != RunSucceeded {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
