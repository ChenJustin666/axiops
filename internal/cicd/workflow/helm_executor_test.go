package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeHelmResolver struct{ target HelmTarget }

func (r fakeHelmResolver) RenderHelm(context.Context, WorkflowRun, HelmDeployConfig) (HelmTarget, error) {
	return r.target, nil
}

type fakeHelmProvider struct {
	request  HelmRequest
	rollback HelmRollbackRequest
}

func (p *fakeHelmProvider) Render(context.Context, HelmRequest) ([]byte, error) {
	return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n"), nil
}

func (p *fakeHelmProvider) UpgradeInstall(_ context.Context, request HelmRequest) (HelmExecution, error) {
	p.request = request
	return HelmExecution{ExternalID: "release-1", Completed: true, Revision: 2}, nil
}

type fakeHelmSnapshotStore struct {
	prepared *DeploymentSnapshot
	applied  int
}

func (s *fakeHelmSnapshotStore) Prepare(_ context.Context, snapshot *DeploymentSnapshot) error {
	s.prepared = snapshot
	return nil
}

func (s *fakeHelmSnapshotStore) MarkHelmApplied(_ context.Context, _ int64, _ string, _ string, revision int) error {
	s.applied = revision
	return nil
}
func (p *fakeHelmProvider) Status(context.Context, string) (HelmState, error) {
	return HelmState{Status: RunSucceeded, Revision: 2}, nil
}
func (p *fakeHelmProvider) Rollback(_ context.Context, request HelmRollbackRequest) (HelmExecution, error) {
	p.rollback = request
	return HelmExecution{ExternalID: "rollback-1", Completed: true, Revision: 3}, nil
}

func TestHelmExecutorForcesAtomicWaitAndStableKey(t *testing.T) {
	provider := &fakeHelmProvider{}
	snapshots := &fakeHelmSnapshotStore{}
	snapshot, _ := json.Marshal(Node{Type: NodeHelmDeploy, Config: mustJSON(HelmDeployConfig{ServiceID: 9, ServiceRevisionID: 12, ReleaseName: "api"})})
	result, err := NewHelmDeployExecutor(fakeHelmResolver{HelmTarget{ClusterID: 2, Namespace: "test", ReleaseName: "api", ChartPath: "chart"}}, provider, snapshots).Execute(context.Background(), ExecutionRequest{Run: WorkflowRun{ID: 21, ProjectID: 4, EnvironmentID: 5}, Task: TaskRun{ID: 6, NodeKey: "deploy", NodeSnapshot: snapshot}})
	if err != nil || result.ProviderRunID != "helm:release-1" || !provider.request.Atomic || !provider.request.Wait || provider.request.IdempotencyKey != "workflow:21:deploy" {
		t.Fatalf("result=%+v request=%+v err=%v", result, provider.request, err)
	}
	if snapshots.prepared == nil || snapshots.applied != 2 || snapshots.prepared.HelmReleaseName != "api" {
		t.Fatalf("snapshot=%+v applied=%d", snapshots.prepared, snapshots.applied)
	}
}

func TestMergeHelmValuesPrecedenceAndDeepMerge(t *testing.T) {
	base := map[string]interface{}{"image": map[string]interface{}{"repository": "old", "tag": "v1"}, "replicas": 1}
	env := map[string]interface{}{"image": map[string]interface{}{"tag": "v2"}}
	override := map[string]interface{}{"replicas": 3}
	got := MergeHelmValues(base, env, override)
	image := got["image"].(map[string]interface{})
	if image["repository"] != "old" || image["tag"] != "v2" || got["replicas"] != 3 {
		t.Fatalf("merged=%+v", got)
	}
	image["tag"] = "changed"
	if base["image"].(map[string]interface{})["tag"] != "v1" {
		t.Fatal("merge mutated input")
	}
}
