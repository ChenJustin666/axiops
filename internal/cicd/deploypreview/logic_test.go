package deploypreview

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type testPolicyStore []authz.Policy

func (p testPolicyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return p, nil
}

type fakePreviewRepository struct {
	target    *Target
	inventory []byte
}

func (f fakePreviewRepository) ResolveTarget(context.Context, int64, int64, int64) (*Target, error) {
	return f.target, nil
}

func (f fakePreviewRepository) LoadLatestHelmInventory(context.Context, int64, int64, string) ([]byte, error) {
	return f.inventory, nil
}

type fakeYAMLResolver struct{ target workflow.YAMLTarget }

func (f fakeYAMLResolver) RenderYAML(context.Context, workflow.WorkflowRun, workflow.YAMLDeployConfig) (workflow.YAMLTarget, error) {
	return f.target, nil
}

type fakeHelmResolver struct{ target workflow.HelmTarget }

func (f fakeHelmResolver) RenderHelm(context.Context, workflow.WorkflowRun, workflow.HelmDeployConfig) (workflow.HelmTarget, error) {
	return f.target, nil
}

type fakeHelmRenderer struct{ manifest []byte }

func (f fakeHelmRenderer) RenderPreview(context.Context, workflow.HelmRequest) ([]byte, error) {
	return f.manifest, nil
}

type fakeKubernetesPreviewer struct {
	changes []workflow.YAMLResourceChange
	called  bool
}

func (f *fakeKubernetesPreviewer) Preview(context.Context, workflow.YAMLProviderRequest) ([]workflow.YAMLResourceChange, error) {
	f.called = true
	return append([]workflow.YAMLResourceChange(nil), f.changes...), nil
}

func (f *fakeKubernetesPreviewer) PreviewHelm(context.Context, int64, string, []byte) ([]workflow.YAMLResourceChange, error) {
	f.called = true
	return append([]workflow.YAMLResourceChange(nil), f.changes...), nil
}

func previewPolicies() authz.Checker {
	return authz.New(testPolicyStore{
		{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "cluster", Action: "view", Effect: "allow", ScopeType: authz.ScopeCluster, ScopeID: 2, Environment: "*"},
	})
}

func testTarget(deliveryType string) *Target {
	return &Target{
		ProjectID: 10, EnvironmentID: 3, ServiceID: 4, ServiceRevisionID: 8, ClusterID: 2,
		EnvironmentCode: "test", EnvironmentType: "k8s_namespace", BindingStatus: "validated", EnvironmentStatus: "active",
		ServiceCode: "payment-api", DeliveryType: deliveryType, Namespace: "payment-test",
	}
}

func TestYAMLPreviewRequiresAllScopesAndSummarizes(t *testing.T) {
	previewer := &fakeKubernetesPreviewer{changes: []workflow.YAMLResourceChange{
		{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "payment-test", Name: "api", Action: "update"},
		{APIVersion: "v1", Kind: "Service", Namespace: "payment-test", Name: "api", Action: "unchanged"},
	}}
	logic := &Logic{
		repo: fakePreviewRepository{target: testTarget(workflow.DeliveryTypeK8sYAML)}, authorizer: previewPolicies(),
		yaml:      fakeYAMLResolver{target: workflow.YAMLTarget{ClusterID: 2, Namespace: "payment-test", Manifest: []byte("manifest")}},
		previewer: previewer,
	}
	result, code := logic.Preview(context.Background(), 7, 3, Request{ServiceID: 4})
	if code != errcode.CodeSuccess || !previewer.called {
		t.Fatalf("code=%d called=%v", code, previewer.called)
	}
	if result.ServiceRevisionID != 8 || result.Summary.Update != 1 || result.Summary.Unchanged != 1 || result.Summary.Delete != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestPreviewRejectsMissingClusterScopeBeforeRendering(t *testing.T) {
	previewer := &fakeKubernetesPreviewer{}
	logic := &Logic{
		repo: fakePreviewRepository{target: testTarget(workflow.DeliveryTypeK8sYAML)},
		authorizer: authz.New(testPolicyStore{
			{Resource: "environment", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
			{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "*"},
		}),
		yaml: fakeYAMLResolver{}, previewer: previewer,
	}
	_, code := logic.Preview(context.Background(), 7, 3, Request{ServiceID: 4})
	if code != errcode.CodeNoPermission || previewer.called {
		t.Fatalf("code=%d called=%v", code, previewer.called)
	}
}

func TestHelmPreviewAddsDeletesAndReturnsOnlySnapshotValues(t *testing.T) {
	previous, _ := json.Marshal([]workflow.YAMLResourceChange{
		{APIVersion: "v1", Kind: "ConfigMap", Namespace: "payment-test", Name: "api"},
		{APIVersion: "v1", Kind: "Service", Namespace: "payment-test", Name: "api"},
	})
	previewer := &fakeKubernetesPreviewer{changes: []workflow.YAMLResourceChange{
		{APIVersion: "v1", Kind: "ConfigMap", Namespace: "payment-test", Name: "api", Action: "unchanged"},
	}}
	logic := &Logic{
		repo: fakePreviewRepository{target: testTarget(workflow.DeliveryTypeHelm), inventory: previous}, authorizer: previewPolicies(),
		helm: fakeHelmResolver{target: workflow.HelmTarget{
			ClusterID: 2, Namespace: "payment-test", ReleaseName: "payment-api", ChartPath: "deploy/chart",
			Chart: []byte("chart"), Values: map[string]interface{}{"password": "runtime-secret"},
			SnapshotValues: map[string]interface{}{"password": map[string]interface{}{"credential_id": int64(9)}}, TimeoutSeconds: 60,
		}},
		renderer: fakeHelmRenderer{manifest: []byte("rendered")}, previewer: previewer,
	}
	result, code := logic.Preview(context.Background(), 7, 3, Request{ServiceID: 4})
	if code != errcode.CodeSuccess || result.Summary.Delete != 1 || len(result.Changes) != 2 {
		t.Fatalf("code=%d result=%+v", code, result)
	}
	encoded, _ := json.Marshal(result)
	if string(encoded) == "" || contains(string(encoded), "runtime-secret") {
		t.Fatalf("runtime secret leaked: %s", encoded)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
