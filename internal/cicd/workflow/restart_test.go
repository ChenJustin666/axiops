package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

type restartTestFactory struct{ url string }

func (f restartTestFactory) RESTConfig(context.Context, int64) (*rest.Config, error) {
	return &rest.Config{Host: f.url}, nil
}

func restartTestObject() *unstructured.Unstructured {
	var object unstructured.Unstructured
	if err := json.Unmarshal([]byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app","namespace":"qa","uid":"uid-1","resourceVersion":"17","generation":3},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"},"annotations":{"keep":"yes"}},"spec":{"containers":[{"name":"app","image":"example.invalid/app:v1"}]}}},"status":{"observedGeneration":3,"replicas":1,"availableReplicas":1,"updatedReplicas":1}}`), &object); err != nil {
		panic(err)
	}
	return &object
}

func restartTestRequest(o *unstructured.Unstructured) ExecutionRequest {
	start := time.Now()
	g := RestartGuard{ServiceID: 9, SnapshotID: 10, WorkflowID: 11, WorkflowVersion: 1, Kind: o.GetKind(), Name: o.GetName(), UID: string(o.GetUID()), Generation: o.GetGeneration(), SpecHash: restartSpecHash(o)}
	f := restartContext{Target: &EnvironmentTarget{ID: 7, ProjectID: 19, Code: "qa", ClusterID: 2, Namespace: "qa", EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"}, Guard: g, Actor: &RunActor{UserID: 5, Username: "tester"}}
	return ExecutionRequest{Run: WorkflowRun{ID: 21, ProjectID: 19, WorkflowID: 11, EnvironmentID: 7, TriggerUserID: 5, ContextSnapshot: mustJSON(f)}, Task: TaskRun{NodeKey: "restart", NodeType: NodeWorkloadRestart, NodeSnapshot: mustJSON(Node{Key: "restart", Type: NodeWorkloadRestart, Config: mustJSON(g)}), StartedAt: &start}}
}

func TestRestartPermissionIndependentAndEnvironmentScoped(t *testing.T) {
	f, _ := restartFrozen(restartTestRequest(restartTestObject()))
	base := policyStore{}
	for _, resource := range []string{"environment", "service", "workflow"} {
		base = append(base, authz.Policy{Resource: resource, Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19})
	}
	base = append(base, authz.Policy{Resource: "deployment", Action: "deploy", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19})
	l := &Logic{authorizer: authz.New(base)}
	if code := l.authorizeRestart(context.Background(), 5, f.Target, f.Guard); code != errcode.CodeNoPermission {
		t.Fatalf("deploy permission permitted restart: %d", code)
	}
	base = append(base, authz.Policy{Resource: "deployment", Action: "restart", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 9, Environment: "qa"})
	l.authorizer = authz.New(base)
	if code := l.authorizeRestart(context.Background(), 5, f.Target, f.Guard); code != 0 {
		t.Fatalf("service scoped restart denied: %d", code)
	}
	f.Target.Code = "prod"
	if code := l.authorizeRestart(context.Background(), 5, f.Target, f.Guard); code != errcode.CodeNoPermission {
		t.Fatalf("qa permission leaked into production: %d", code)
	}
	f.Target.Code = "qa"
	base = append(base, authz.Policy{Resource: "deployment", Action: "restart", Effect: "deny", ScopeType: authz.ScopeCluster, ScopeID: 2})
	l.authorizer = authz.New(base)
	if code := l.authorizeRestart(context.Background(), 5, f.Target, f.Guard); code != errcode.CodeNoPermission {
		t.Fatalf("explicit deny ignored: %d", code)
	}
}

func TestRestartStrategiesAndNoConfigMutation(t *testing.T) {
	for _, test := range []struct {
		name    string
		change  func(*unstructured.Unstructured)
		blocked bool
	}{
		{"rolling", func(*unstructured.Unstructured) {}, false},
		{"paused", func(o *unstructured.Unstructured) { _ = unstructured.SetNestedField(o.Object, true, "spec", "paused") }, true},
		{"stopped", func(o *unstructured.Unstructured) {
			_ = unstructured.SetNestedField(o.Object, int64(0), "spec", "replicas")
		}, true},
		{"recreate", func(o *unstructured.Unstructured) {
			_ = unstructured.SetNestedField(o.Object, "Recreate", "spec", "strategy", "type")
		}, true},
		{"ondelete", func(o *unstructured.Unstructured) {
			o.SetKind("StatefulSet")
			_ = unstructured.SetNestedField(o.Object, "OnDelete", "spec", "updateStrategy", "type")
		}, true},
		{"partition", func(o *unstructured.Unstructured) {
			o.SetKind("StatefulSet")
			_ = unstructured.SetNestedField(o.Object, int64(1), "spec", "updateStrategy", "rollingUpdate", "partition")
		}, true},
		{"daemon-no-node", func(o *unstructured.Unstructured) { o.SetKind("DaemonSet") }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			o := restartTestObject()
			test.change(o)
			if err := validateRestartWorkload(o); (err != nil) != test.blocked {
				t.Fatalf("blocked=%v err=%v", test.blocked, err)
			}
		})
	}
	o := restartTestObject()
	before := restartSpecHash(o)
	patch, err := restartPatch(o, "operation-1")
	if err != nil {
		t.Fatal(err)
	}
	var ops []map[string]interface{}
	_ = json.Unmarshal(patch, &ops)
	if len(ops) != 3 || ops[0]["path"] != "/metadata/uid" || ops[1]["path"] != "/metadata/resourceVersion" || ops[2]["path"] != "/spec/template/metadata/annotations" {
		t.Fatalf("unsafe patch: %s", patch)
	}
	annotations := ops[2]["value"].(map[string]interface{})
	if annotations["keep"] != "yes" {
		t.Fatal("existing annotations lost")
	}
	_ = unstructured.SetNestedField(o.Object, "operation-1", "spec", "template", "metadata", "annotations", restartAnnotation)
	if restartSpecHash(o) != before {
		t.Fatal("marker changed config fingerprint")
	}
	_ = unstructured.SetNestedField(o.Object, int64(2), "spec", "replicas")
	if restartSpecHash(o) == before {
		t.Fatal("replica drift not detected")
	}
}

func TestRestartRecoveryAndObservedGeneration(t *testing.T) {
	o := restartTestObject()
	r := restartTestRequest(o)
	f, err := restartFrozen(r)
	if err != nil {
		t.Fatal(err)
	}
	_ = unstructured.SetNestedField(o.Object, restartMarker(r), "spec", "template", "metadata", "annotations", restartAnnotation)
	o.SetGeneration(4)
	getCount, writeCount := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			writeCount++
			t.Errorf("recovery wrote to Kubernetes: %s", req.Method)
		} else {
			getCount++
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(o.Object)
	}))
	defer server.Close()
	e := NewWorkloadRestartExecutor(nil, restartTestFactory{server.URL}, authz.New(policyStore{}))
	result, err := e.Execute(context.Background(), r)
	if err != nil || result.Status != RunRunning || result.ProviderRunID != restartMarker(r) {
		t.Fatalf("recovery=%+v err=%v", result, err)
	}
	if getCount != 1 || writeCount != 0 {
		t.Fatal("recovery not read only")
	}
	if result := observeRestart(o, r, f); result.Status != RunRunning {
		t.Fatalf("old generation marked ready: %+v", result)
	}
	_ = unstructured.SetNestedSlice(o.Object, []interface{}{map[string]interface{}{"type": "Progressing", "status": "False", "reason": "ProgressDeadlineExceeded", "message": "previous rollout"}}, "status", "conditions")
	if result := observeRestart(o, r, f); result.Status != RunRunning {
		t.Fatalf("old failure condition completed new rollout: %+v", result)
	}
	unstructured.RemoveNestedField(o.Object, "status", "conditions")
	_ = unstructured.SetNestedField(o.Object, int64(4), "status", "observedGeneration")
	if result := observeRestart(o, r, f); result.Status != RunSucceeded {
		t.Fatalf("new generation not ready: %+v", result)
	}
	o.SetGeneration(5)
	if result := observeRestart(o, r, f); result.Status != RunFailed {
		t.Fatal("external restart/update not detected")
	}
	o.SetGeneration(4)
	o.SetUID("replacement")
	if result := observeRestart(o, r, f); result.Status != RunFailed {
		t.Fatal("replacement not detected")
	}
	start := time.Now().Add(-11 * time.Minute)
	r.Task.StartedAt = &start
	if result, err := e.Reconcile(context.Background(), r); err != nil || result.Status != RunFailed {
		t.Fatalf("timeout=%+v %v", result, err)
	}
}

func TestRestartStatefulAndDaemonReadiness(t *testing.T) {
	for _, kind := range []string{"StatefulSet", "DaemonSet"} {
		t.Run(kind, func(t *testing.T) {
			o := restartTestObject()
			o.SetKind(kind)
			if kind == "DaemonSet" {
				_ = unstructured.SetNestedField(o.Object, int64(1), "status", "desiredNumberScheduled")
				_ = unstructured.SetNestedField(o.Object, int64(1), "status", "updatedNumberScheduled")
				_ = unstructured.SetNestedField(o.Object, int64(1), "status", "numberAvailable")
			} else {
				_ = unstructured.SetNestedField(o.Object, int64(1), "status", "readyReplicas")
				_ = unstructured.SetNestedField(o.Object, "old", "status", "currentRevision")
				_ = unstructured.SetNestedField(o.Object, "new", "status", "updateRevision")
			}
			r := restartTestRequest(o)
			f, err := restartFrozen(r)
			if err != nil {
				t.Fatal(err)
			}
			_ = unstructured.SetNestedField(o.Object, restartMarker(r), "spec", "template", "metadata", "annotations", restartAnnotation)
			o.SetGeneration(4)
			if result := observeRestart(o, r, f); result.Status != RunRunning {
				t.Fatalf("old observed generation completed: %+v", result)
			}
			_ = unstructured.SetNestedField(o.Object, int64(4), "status", "observedGeneration")
			if kind == "StatefulSet" {
				if result := observeRestart(o, r, f); result.Status != RunRunning {
					t.Fatalf("mixed revisions completed: %+v", result)
				}
				_ = unstructured.SetNestedField(o.Object, "new", "status", "currentRevision")
			}
			if result := observeRestart(o, r, f); result.Status != RunSucceeded {
				t.Fatalf("ready rollout not completed: %+v", result)
			}
		})
	}
}

func TestRestartRevokedPermissionDoesNotPatch(t *testing.T) {
	o := restartTestObject()
	r := restartTestRequest(o)
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			writes++
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(o.Object)
	}))
	defer server.Close()
	e := NewWorkloadRestartExecutor(nil, restartTestFactory{server.URL}, authz.New(policyStore{}))
	if _, err := e.Execute(context.Background(), r); err == nil {
		t.Fatal("revoked user executed restart")
	}
	if writes != 0 {
		t.Fatal("denied user patched workload")
	}
	if result, err := e.Cancel(context.Background(), r); err != nil || result.Status != RunCancelled {
		t.Fatalf("cancel before submission: %+v %v", result, err)
	}
	if writes != 0 {
		t.Fatal("cancel wrote to Kubernetes")
	}
}
