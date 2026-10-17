package workflow

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"
	"time"
)

type healthClient struct {
	fakeKubernetesClient
	live          *unstructured.Unstructured
	diagnostic    string
	diagnosticErr error
	readErr       error
}

func (c *healthClient) ReadResource(context.Context, string, unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.live, c.readErr
}

func (c *healthClient) DiagnoseWorkload(context.Context, string, unstructured.Unstructured) (string, error) {
	return c.diagnostic, c.diagnosticErr
}

type healthFactory struct {
	client  *healthClient
	cluster int64
}

func TestYAMLReadinessIncludesWorkloadDiagnosticsOnTimeout(t *testing.T) {
	live := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "api", "generation": int64(2)},
		"spec":   map[string]interface{}{"replicas": int64(1)},
		"status": map[string]interface{}{"observedGeneration": int64(2), "replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(0)},
	}}
	client := &healthClient{live: live, diagnostic: "pod api-1 phase=Pending container api waiting=ImagePullBackOff"}
	f := &healthFactory{client: client}
	target := YAMLHealthTarget{YAMLTarget: YAMLTarget{ClusterID: 1, Namespace: "qa", Manifest: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n")}, Deadline: time.Now().Add(-time.Second)}
	state, err := NewKubernetesYAMLProvider(f, "test", false).WithHealthTargets(healthStore{target}).Status(context.Background(), "workflow:1:deploy")
	if err != nil || state.Status != RunFailed {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if !strings.Contains(state.ErrorMessage, "ImagePullBackOff") {
		t.Fatalf("diagnostic missing from %q", state.ErrorMessage)
	}
}

func (f *healthFactory) Client(_ context.Context, id int64) (KubernetesResourceApplier, error) {
	f.cluster = id
	return f.client, nil
}

type healthStore struct{ target YAMLHealthTarget }

func (s healthStore) HealthTarget(context.Context, string) (YAMLHealthTarget, error) {
	return s.target, nil
}

func TestYAMLReadinessSurvivesProviderRestart(t *testing.T) {
	live := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "api", "generation": int64(2)},
		"spec":   map[string]interface{}{"replicas": int64(2)},
		"status": map[string]interface{}{"observedGeneration": int64(1), "replicas": int64(2), "updatedReplicas": int64(2), "availableReplicas": int64(2)},
	}}
	f := &healthFactory{client: &healthClient{live: live}}
	target := YAMLHealthTarget{YAMLTarget: YAMLTarget{ClusterID: 99, Namespace: "prod", Manifest: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n")}, Deadline: time.Now().Add(time.Minute)}
	// New provider has no in-memory Apply state, as after a worker restart.
	p := NewKubernetesYAMLProvider(f, "test", false).WithHealthTargets(healthStore{target})
	state, err := p.Status(context.Background(), "workflow:1:deploy")
	if err != nil || state.Status != RunRunning || f.cluster != 99 {
		t.Fatalf("state=%+v cluster=%d err=%v", state, f.cluster, err)
	}
	unstructured.SetNestedField(live.Object, int64(2), "status", "observedGeneration")
	state, err = p.Status(context.Background(), "workflow:1:deploy")
	if err != nil || state.Status != RunSucceeded {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if !strings.Contains(state.LogText, "[succeeded]") || !strings.Contains(state.LogText, "available=2") {
		t.Fatalf("missing success evidence: %s", state.LogText)
	}
	unstructured.SetNestedField(live.Object, int64(1), "status", "availableReplicas")
	target.Deadline = time.Now().Add(-time.Second)
	p.WithHealthTargets(healthStore{target})
	state, err = p.Status(context.Background(), "workflow:1:deploy")
	if err != nil || state.Status != RunFailed {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func TestYAMLDiagnosticFailureDoesNotFailHealthyRollout(t *testing.T) {
	live := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "api"},
		"status": map[string]interface{}{"replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(1)},
	}}
	client := &healthClient{live: live, diagnosticErr: errors.New("forbidden")}
	target := YAMLHealthTarget{YAMLTarget: YAMLTarget{ClusterID: 1, Namespace: "qa", Manifest: []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n")}, Deadline: time.Now().Add(time.Minute)}
	provider := NewKubernetesYAMLProvider(&healthFactory{client: client}, "test", false)
	state, err := provider.checkTarget(context.Background(), target)
	if err != nil || state.Status != RunSucceeded || !strings.Contains(state.LogText, "unavailable: forbidden") {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	// A status read failure cannot leave the deployment running past its deadline.
	client.readErr = errors.New("cluster disconnected")
	target.Deadline = time.Now().Add(-time.Minute)
	state, err = provider.checkTarget(context.Background(), target)
	if err != nil || state.Status != RunFailed || !strings.Contains(state.LogText, "cluster disconnected") {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}
