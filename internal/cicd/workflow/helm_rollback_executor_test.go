package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

func appliedHelmSnapshot(t *testing.T) *DeploymentSnapshot {
	t.Helper()
	snapshot, err := NewHelmDeploymentSnapshot(
		WorkflowRun{ID: 20, ProjectID: 10, EnvironmentID: 7},
		TaskRun{ID: 22, NodeKey: "deploy"},
		HelmDeployConfig{ServiceID: 9, ServiceRevisionID: 12, ReleaseName: "api"},
		HelmTarget{ClusterID: 2, Namespace: "test", ReleaseName: "api", SnapshotValues: map[string]interface{}{}},
		[]byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n"),
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.ID = 30
	snapshot.Status = "applied"
	snapshot.HelmRevision = 4
	return snapshot
}

func TestHelmRollbackExecutorUsesAuditedRevision(t *testing.T) {
	provider := &fakeHelmProvider{}
	executor := NewHelmRollbackExecutor(fakeSnapshotReader{snapshot: appliedHelmSnapshot(t)}, provider)
	node, _ := json.Marshal(Node{Key: "helm-rollback", Type: NodeHelmRollback, TimeoutSeconds: 300, Config: mustJSON(HelmRollbackConfig{SnapshotID: 30})})
	result, err := executor.Execute(context.Background(), ExecutionRequest{
		Run:  WorkflowRun{ID: 40, ProjectID: 10, EnvironmentID: 7},
		Task: TaskRun{NodeKey: "helm-rollback", NodeSnapshot: node},
	})
	if err != nil || result.Status != RunSucceeded || provider.rollback.Revision != 4 || provider.rollback.ReleaseName != "api" || provider.rollback.IdempotencyKey != "workflow:40:helm-rollback" {
		t.Fatalf("result=%+v rollback=%+v err=%v", result, provider.rollback, err)
	}
}

func TestHelmRollbackExecutorRejectsCrossProjectSnapshot(t *testing.T) {
	executor := NewHelmRollbackExecutor(fakeSnapshotReader{snapshot: appliedHelmSnapshot(t)}, &fakeHelmProvider{})
	node, _ := json.Marshal(Node{Key: "helm-rollback", Type: NodeHelmRollback, Config: mustJSON(HelmRollbackConfig{SnapshotID: 30})})
	_, err := executor.Execute(context.Background(), ExecutionRequest{
		Run: WorkflowRun{ID: 40, ProjectID: 99, EnvironmentID: 7}, Task: TaskRun{NodeKey: "helm-rollback", NodeSnapshot: node},
	})
	if err == nil {
		t.Fatal("cross-project Helm rollback was accepted")
	}
}
