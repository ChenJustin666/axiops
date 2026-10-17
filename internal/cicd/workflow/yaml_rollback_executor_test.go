package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type fakeSnapshotReader struct{ snapshot *DeploymentSnapshot }

func (r fakeSnapshotReader) FindApplied(context.Context, int64) (*DeploymentSnapshot, error) {
	return r.snapshot, nil
}

func TestYAMLRollbackStopsBeforeApplyWhenDependencyValidationFails(t *testing.T) {
	manifest := []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  namespace: test\n")
	snapshot, err := NewYAMLDeploymentSnapshot(WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7}, TaskRun{ID: 22, NodeKey: "deploy"}, YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12}, YAMLTarget{ClusterID: 2, Namespace: "test", Manifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ID = 30
	snapshot.Status = "applied"
	provider := &fakeYAMLProvider{}
	executor := NewYAMLRollbackExecutor(fakeSnapshotReader{snapshot}, provider).WithDeploymentPreparer(deploymentPreparerFunc(func(context.Context, WorkflowRun, YAMLDeployConfig, YAMLTarget) (YAMLTarget, error) {
		return YAMLTarget{}, fmt.Errorf("required Secret is missing")
	}))
	_, err = executor.Execute(context.Background(), ExecutionRequest{Run: WorkflowRun{ID: 40, ProjectID: 10, EnvironmentID: 7}, Task: TaskRun{ID: 41, NodeKey: "rollback", NodeSnapshot: mustJSON(Node{Type: NodeYAMLRollback, Config: mustJSON(YAMLRollbackConfig{SnapshotID: 30})})}})
	if err == nil || provider.request.RunID != 0 {
		t.Fatal("rollback applied despite missing dependency")
	}
}

func TestYAMLRollbackExecutorUsesAuditedSnapshot(t *testing.T) {
	manifest := []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  namespace: test\n")
	snapshot, err := NewYAMLDeploymentSnapshot(
		WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7}, TaskRun{ID: 22, NodeKey: "deploy"},
		YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12},
		YAMLTarget{ClusterID: 2, Namespace: "test", Manifest: manifest},
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ID = 30
	snapshot.Status = "applied"
	nodeSnapshot, _ := json.Marshal(Node{Key: "rollback", Type: NodeYAMLRollback, Config: json.RawMessage(`{"snapshot_id":30}`)})
	provider := &fakeYAMLProvider{}
	result, err := NewYAMLRollbackExecutor(fakeSnapshotReader{snapshot: snapshot}, provider).Execute(context.Background(), ExecutionRequest{
		Run:  WorkflowRun{ID: 40, ProjectID: 10, EnvironmentID: 7},
		Task: TaskRun{ID: 41, NodeKey: "rollback", NodeSnapshot: nodeSnapshot},
	})
	if err != nil || result.ProviderRunID != "yaml-rollback:apply-1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if provider.request.IdempotencyKey != "workflow:40:rollback:rollback:30" || string(provider.request.Manifest) != string(manifest) {
		t.Fatalf("request=%+v", provider.request)
	}
}

func TestYAMLRollbackExecutorRejectsTamperedSnapshot(t *testing.T) {
	snapshot := &DeploymentSnapshot{
		ID: 30, ProjectID: 10, EnvironmentID: 7, ClusterID: 2, Namespace: "test", Provider: NodeYAMLDeploy,
		ManifestDigest: "invalid", RenderedManifest: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n"), Status: "applied",
	}
	nodeSnapshot, _ := json.Marshal(Node{Key: "rollback", Type: NodeYAMLRollback, Config: json.RawMessage(`{"snapshot_id":30}`)})
	_, err := NewYAMLRollbackExecutor(fakeSnapshotReader{snapshot: snapshot}, &fakeYAMLProvider{}).Execute(context.Background(), ExecutionRequest{
		Run: WorkflowRun{ID: 40, ProjectID: 10, EnvironmentID: 7}, Task: TaskRun{ID: 41, NodeKey: "rollback", NodeSnapshot: nodeSnapshot},
	})
	if err == nil {
		t.Fatal("tampered snapshot was accepted")
	}
}
