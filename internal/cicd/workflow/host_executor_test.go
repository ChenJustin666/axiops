package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeHostResolver struct {
	target HostTarget
	err    error
}

func (f fakeHostResolver) RenderHost(context.Context, WorkflowRun, HostDeployConfig) (HostTarget, error) {
	return f.target, f.err
}

type fakeHostProvider struct {
	requests []HostProviderRequest
	state    HostProviderState
}

func (f *fakeHostProvider) Submit(_ context.Context, request HostProviderRequest) error {
	f.requests = append(f.requests, request)
	return nil
}

func (f *fakeHostProvider) Status(context.Context, int64, string) (HostProviderState, error) {
	return f.state, nil
}

func TestHostDeployExecutorSubmitAndReconcile(t *testing.T) {
	node, _ := json.Marshal(Node{Key: "deploy", Type: NodeHostDeploy, Config: json.RawMessage(`{"service_id":8,"service_revision_id":12}`)})
	provider := &fakeHostProvider{state: HostProviderState{Status: RunSucceeded}}
	executor := NewHostDeployExecutor(fakeHostResolver{target: HostTarget{
		AgentIDs: []int64{3, 4}, ArtifactURL: "https://artifacts.example.com/app.tgz",
		ArtifactSHA256: string(make([]byte, 64)), DeployScript: "deploy", TimeoutSeconds: 60,
	}}, provider)
	request := ExecutionRequest{Run: WorkflowRun{ID: 21, ProjectID: 5}, Task: TaskRun{NodeKey: "deploy", NodeSnapshot: node}}
	result, err := executor.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Status != RunRunning || result.ProviderRunID != "host:21:deploy" || len(provider.requests) != 1 {
		t.Fatalf("unexpected execution result: %+v requests=%d", result, len(provider.requests))
	}
	if provider.requests[0].ProjectID != 5 || provider.requests[0].ServiceID != 8 || provider.requests[0].IdempotencyKey != "workflow:21:deploy" {
		t.Fatalf("unexpected provider request: %+v", provider.requests[0])
	}
	request.Task.ProviderRunID = result.ProviderRunID
	reconciled, err := executor.Reconcile(context.Background(), request)
	if err != nil || reconciled.Status != RunSucceeded {
		t.Fatalf("reconcile: result=%+v err=%v", reconciled, err)
	}
}

func TestHostDeployExecutorRejectsInvalidProviderRunID(t *testing.T) {
	executor := NewHostDeployExecutor(fakeHostResolver{}, &fakeHostProvider{})
	_, err := executor.Reconcile(context.Background(), ExecutionRequest{
		Run: WorkflowRun{ID: 21}, Task: TaskRun{NodeKey: "deploy", ProviderRunID: "host:22:deploy"},
	})
	if err == nil {
		t.Fatal("expected invalid provider run id to fail")
	}
}
