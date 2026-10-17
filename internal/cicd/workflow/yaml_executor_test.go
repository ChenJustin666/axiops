package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type fakeYAMLResolver struct{ target YAMLTarget }

func (r fakeYAMLResolver) RenderYAML(context.Context, WorkflowRun, YAMLDeployConfig) (YAMLTarget, error) {
	return r.target, nil
}

type fakeYAMLProvider struct{ request YAMLProviderRequest }

func (p *fakeYAMLProvider) Apply(_ context.Context, request YAMLProviderRequest) (YAMLProviderExecution, error) {
	p.request = request
	return YAMLProviderExecution{ExternalID: "apply-1", LogRef: "/logs/apply-1"}, nil
}

type fakeSnapshotStore struct {
	prepared bool
	applied  bool
	snapshot *DeploymentSnapshot
}

func (s *fakeSnapshotStore) Prepare(_ context.Context, snapshot *DeploymentSnapshot) error {
	s.prepared = true
	s.snapshot = snapshot
	return nil
}

type deploymentPreparerFunc func(context.Context, WorkflowRun, YAMLDeployConfig, YAMLTarget) (YAMLTarget, error)

func (f deploymentPreparerFunc) PrepareYAMLDeployment(ctx context.Context, run WorkflowRun, cfg YAMLDeployConfig, target YAMLTarget) (YAMLTarget, error) {
	return f(ctx, run, cfg, target)
}

func TestDeploymentPreparerRunsBeforeSnapshotAndApply(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: registry.example/api:v1\n")
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("fail=%v", fail), func(t *testing.T) {
			provider := &fakeYAMLProvider{}
			snapshots := &fakeSnapshotStore{}
			preparer := deploymentPreparerFunc(func(_ context.Context, run WorkflowRun, cfg YAMLDeployConfig, target YAMLTarget) (YAMLTarget, error) {
				if run.ProjectID != 10 || cfg.ServiceID != 9 || snapshots.prepared || provider.request.RunID != 0 {
					t.Fatal("preparer called in wrong scope/order")
				}
				if fail {
					return target, fmt.Errorf("secret provisioning failed")
				}
				target.Manifest = []byte(strings.Replace(string(target.Manifest), "      containers:", "      imagePullSecrets:\n      - name: managed-pull-secret\n      containers:", 1))
				return target, nil
			})
			node, _ := json.Marshal(Node{Key: "deploy", Type: NodeYAMLDeploy, Config: json.RawMessage(`{"service_id":9,"service_revision_id":12}`)})
			_, err := NewYAMLDeployExecutor(fakeYAMLResolver{target: YAMLTarget{ClusterID: 2, Namespace: "test", Manifest: manifest}}, provider, snapshots).WithDeploymentPreparer(preparer).Execute(context.Background(), ExecutionRequest{Run: WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7}, Task: TaskRun{ID: 22, NodeKey: "deploy", NodeSnapshot: node}})
			if fail {
				if err == nil || snapshots.prepared || provider.request.RunID != 0 {
					t.Fatal("deployment continued after credential failure")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !snapshots.snapshot.VerifyManifestDigest() || string(snapshots.snapshot.RenderedManifest) != string(provider.request.Manifest) || !strings.Contains(string(provider.request.Manifest), "managed-pull-secret") {
				t.Fatal("snapshot does not record exact prepared manifest")
			}
		})
	}
}
func (s *fakeSnapshotStore) MarkApplied(context.Context, int64, string, string) error {
	s.applied = true
	return nil
}
func (p *fakeYAMLProvider) Status(context.Context, string) (YAMLProviderState, error) {
	return YAMLProviderState{Status: RunSucceeded}, nil
}

func TestValidateYAMLManifestRejectsSecretAndWrongNamespace(t *testing.T) {
	secret := []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: token\n")
	if _, err := ValidateYAMLManifest(secret, "test"); err == nil {
		t.Fatal("expected Secret rejection")
	}
	service := []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n  namespace: prod\nspec:\n  selector:\n    app: api\n")
	if _, err := ValidateYAMLManifest(service, "test"); err == nil {
		t.Fatal("expected namespace rejection")
	}
}

func TestYAMLDeployExecutorUsesStableIdempotencyKey(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  selector:\n    matchLabels:\n      app: api\n  template:\n    metadata:\n      labels:\n        app: api\n    spec:\n      containers:\n      - name: api\n        image: example/api:v1\n")
	provider := &fakeYAMLProvider{}
	snapshots := &fakeSnapshotStore{}
	nodeSnapshot, _ := json.Marshal(Node{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, Config: json.RawMessage(`{"service_id":9,"service_revision_id":12}`)})
	result, err := NewYAMLDeployExecutor(fakeYAMLResolver{target: YAMLTarget{ClusterID: 2, Namespace: "test", Manifest: manifest}}, provider, snapshots).Execute(context.Background(), ExecutionRequest{
		Run: WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7}, Task: TaskRun{ID: 22, NodeKey: "deploy", NodeSnapshot: nodeSnapshot},
	})
	if err != nil || result.ProviderRunID != "yaml:apply-1" || provider.request.IdempotencyKey != "workflow:21:deploy" || !snapshots.prepared {
		t.Fatalf("result=%+v request=%+v err=%v", result, provider.request, err)
	}
	reconcile, err := NewYAMLDeployExecutor(nil, provider, snapshots).Reconcile(context.Background(), ExecutionRequest{
		Run:  WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7},
		Task: TaskRun{ID: 22, NodeKey: "deploy", NodeSnapshot: nodeSnapshot, ProviderRunID: result.ProviderRunID},
	})
	if err != nil || reconcile.Status != RunSucceeded || !snapshots.applied {
		t.Fatalf("reconcile=%+v applied=%t err=%v", reconcile, snapshots.applied, err)
	}
}
