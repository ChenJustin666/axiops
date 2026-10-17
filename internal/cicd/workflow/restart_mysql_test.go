package workflow

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func testMySQLWorkloadRestart(t *testing.T, db *sql.DB) {
	ctx := actorContext(7, "restart-tester")
	repo := NewRepository(db)
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO app_service(id,project_id,name,code,status) VALUES(1201,19,'restart-app','restart-app','active')`)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status) VALUES(1201,19,'Restart QA','restart-qa','k8s_namespace',2,'restart-qa','validated','active')`)
	env, err := repo.FindEnvironmentTarget(ctx, 1201)
	if err != nil {
		t.Fatal(err)
	}
	node := Node{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, Config: mustJSON(YAMLDeployConfig{ServiceID: 1201, ServiceRevisionID: 1})}
	wf := &Workflow{ProjectID: 19, TargetEnvironmentID: 1201, Name: "Restart QA", Code: "restart-qa", DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{node}}
	wid, err := repo.Create(ctx, wf)
	if err != nil {
		t.Fatal(err)
	}
	origin := &WorkflowRun{WorkflowID: wid, ProjectID: 19, EnvironmentID: 1201, DefinitionVersion: 1, Status: RunSucceeded, TriggerType: "manual", ContextSnapshot: mustJSON(map[string]interface{}{"target": env})}
	id, err := repo.CreateRun(ctx, origin, []TaskRun{{NodeKey: node.Key, NodeType: node.Type, NodeSnapshot: mustJSON(node), Status: RunSucceeded}})
	if err != nil {
		t.Fatal(err)
	}
	origin, err = repo.FindRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	o := restartTestObject()
	o.SetNamespace(env.Namespace)
	manifest, _ := json.Marshal(o.Object)
	snapshot, err := NewYAMLDeploymentSnapshot(*origin, origin.Tasks[0], YAMLDeployConfig{ServiceID: 1201, ServiceRevisionID: 1}, YAMLTarget{ClusterID: 2, Namespace: env.Namespace, Manifest: manifest})
	if err != nil {
		t.Fatal(err)
	}
	store := NewSQLDeploymentSnapshotStore(db)
	if err := store.Prepare(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkApplied(ctx, origin.ID, node.Key, NodeYAMLDeploy); err != nil {
		t.Fatal(err)
	}
	policies := policyStore{}
	for _, resource := range []string{"environment", "service", "workflow"} {
		policies = append(policies, authz.Policy{Resource: resource, Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19})
	}
	policies = append(policies, authz.Policy{Resource: "deployment", Action: "restart", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 1201, Environment: env.Code})
	var mu sync.Mutex
	writes := 0
	loseReply := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path != "/apis/apps/v1/namespaces/restart-qa/deployments/app" {
			t.Errorf("wrong target: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PATCH" {
			writes++
			raw, _ := io.ReadAll(r.Body)
			patch, err := jsonpatch.DecodePatch(raw)
			if err != nil {
				t.Error(err)
				w.WriteHeader(422)
				return
			}
			before, _ := json.Marshal(o.Object)
			after, err := patch.Apply(before)
			if err != nil {
				w.WriteHeader(409)
				return
			}
			var next unstructured.Unstructured
			if err := json.Unmarshal(after, &next); err != nil {
				t.Error(err)
				return
			}
			next.SetGeneration(o.GetGeneration() + 1)
			next.SetResourceVersion("18")
			o = &next
			if loseReply {
				w.WriteHeader(504)
				_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Timeout","code":504}`))
				return
			}
		} else if r.Method != "GET" {
			t.Errorf("unexpected mutation %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(o.Object)
	}))
	defer server.Close()
	factory := restartTestFactory{server.URL}
	checker := authz.New(policies)
	logic := NewLogic(db, checker).WithRestartFactory(factory)
	preview, code, message := logic.PreviewRestart(ctx, 7, env.ID, "deployment", "app")
	if code != 0 {
		t.Fatalf("preview %d %s", code, message)
	}
	if preview.ServiceID != 1201 || preview.Namespace != env.Namespace || len(preview.Images) != 1 {
		t.Fatalf("bad preview %+v", preview)
	}
	denied := NewLogic(db, authz.New(policies[:len(policies)-1])).WithRestartFactory(factory)
	if _, code, _ := denied.PreviewRestart(ctx, 7, env.ID, "deployment", "app"); code != errcode.CodeNoPermission {
		t.Fatalf("read-only account not denied: %d", code)
	}
	if writes != 0 {
		t.Fatal("preview mutated Kubernetes")
	}
	// Authenticated HTTP handler must preserve denial and require a fresh token.
	handler := NewHandler(denied)
	httpReq := httptest.NewRequest(http.MethodPost, "/environments/1201/restarts", strings.NewReader(`{"kind":"Deployment","name":"app","expected_token":"`+preview.Token+`","request_key":"restart-http-denied"}`)).WithContext(ctx)
	httpReq.SetPathValue("id", "1201")
	recorder := httptest.NewRecorder()
	handler.SubmitRestart(recorder, httpReq)
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || response.Code != errcode.CodeNoPermission {
		t.Fatalf("handler denial code=%d err=%v", response.Code, err)
	}
	if writes != 0 {
		t.Fatal("denied handler mutated Kubernetes")
	}
	req := RestartRequest{Kind: "Deployment", Name: "app", ExpectedToken: preview.Token, RequestKey: "restart-mysql-duplicate-1"}
	var ids [2]int64
	var codes [2]int
	var messages [2]string
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); ids[i], codes[i], messages[i] = logic.SubmitRestart(ctx, 7, env.ID, req) }(i)
	}
	wg.Wait()
	if codes[0] != 0 || codes[1] != 0 || ids[0] != ids[1] || ids[0] <= 0 {
		t.Fatalf("idempotence ids=%v codes=%v messages=%v", ids, codes, messages)
	}
	run, err := repo.FindRun(ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	snapshotRunActor(run)
	if run.OperationType != "restart" || len(run.Tasks) != 1 || run.TriggerUserID != 7 || run.TriggerUsername != "restart-tester" {
		t.Fatal("restart record missing operation, task or actor identity")
	}
	normal := &WorkflowRun{WorkflowID: wid, ProjectID: 19, EnvironmentID: env.ID, DefinitionVersion: 1, Status: RunPending, TriggerType: "manual", ContextSnapshot: mustJSON(map[string]interface{}{"target": env})}
	tasks := []TaskRun{{NodeKey: node.Key, NodeType: node.Type, NodeSnapshot: mustJSON(node), Status: RunPending}}
	if _, err := repo.CreateRun(ctx, normal, tasks); err == nil {
		t.Fatal("deployment did not block on active restart")
	}
	if _, code, _ := logic.SubmitRestart(ctx, 8, env.ID, req); code == 0 {
		t.Fatal("different actor replayed request")
	}
	changed := req
	changed.Name = "another-service"
	if _, code, _ := logic.SubmitRestart(ctx, 7, env.ID, changed); code == 0 {
		t.Fatal("idempotency key accepted different target")
	}
	if ok, err := repo.ClaimTask(ctx, run.ID, "restart", "test-worker", time.Minute); err != nil || !ok {
		t.Fatalf("claim %v %v", ok, err)
	}
	run, err = repo.FindRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	e := NewWorkloadRestartExecutor(db, factory, checker)
	// Simulate an accepted Kubernetes patch followed by a lost response.
	loseReply = true
	r := ExecutionRequest{Run: *run, Task: run.Tasks[0]}
	if _, err := e.Execute(ctx, r); err == nil || !strings.Contains(err.Error(), ErrSubmissionUncertain.Error()) {
		t.Fatalf("lost reply not uncertain: %v", err)
	}
	result, err := e.Execute(ctx, r)
	if err != nil || result.Status != RunRunning {
		t.Fatalf("recover %+v %v", result, err)
	}
	if writes != 1 {
		t.Fatalf("recovery repeated patch: %d", writes)
	}
	if err := repo.MarkTaskSubmitted(ctx, run.ID, "restart", "test-worker", result); err != nil {
		t.Fatal(err)
	}
	if retry, code, msg := logic.SubmitRestart(ctx, 7, env.ID, req); code != 0 || retry != run.ID {
		t.Fatalf("post-patch retry %d %d %s", retry, code, msg)
	}
	if result, err := e.Reconcile(ctx, r); err != nil || result.Status != RunRunning {
		t.Fatalf("premature completion %+v %v", result, err)
	}
	mu.Lock()
	_ = unstructured.SetNestedField(o.Object, o.GetGeneration(), "status", "observedGeneration")
	mu.Unlock()
	result, err = e.Reconcile(ctx, r)
	if err != nil || result.Status != RunSucceeded {
		t.Fatalf("ready result %+v %v", result, err)
	}
	if err := repo.CompleteSubmittedTask(ctx, run.ID, "restart", result.Status, ""); err != nil {
		t.Fatal(err)
	}
	// A deploy blocks a fresh restart in the opposite direction too.
	active, err := repo.CreateRun(ctx, normal, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if _, code, _ := logic.PreviewRestart(ctx, 7, env.ID, "deployment", "app"); code == 0 {
		t.Fatal("restart ignored active deployment")
	}
	exec("UPDATE workflow_run SET status='succeeded' WHERE id=?", active)
	// Queued permissions must be rechecked before any Kubernetes mutation.
	p, code, msg := logic.PreviewRestart(ctx, 7, env.ID, "deployment", "app")
	if code != 0 {
		t.Fatalf("new preview %d %s", code, msg)
	}
	newReq := RestartRequest{Kind: "Deployment", Name: "app", ExpectedToken: p.Token, RequestKey: "restart-mysql-revoked-2"}
	newID, code, msg := logic.SubmitRestart(ctx, 7, env.ID, newReq)
	if code != 0 {
		t.Fatalf("submit %d %s", code, msg)
	}
	_, _ = repo.ClaimTask(ctx, newID, "restart", "test-worker", time.Minute)
	newRun, _ := repo.FindRun(ctx, newID)
	revoked := NewWorkloadRestartExecutor(db, factory, authz.New(policies[:len(policies)-1]))
	if _, err := revoked.Execute(ctx, ExecutionRequest{Run: *newRun, Task: newRun.Tasks[0]}); err == nil {
		t.Fatal("queued revoked user executed")
	}
	if writes != 1 {
		t.Fatalf("denied restart wrote: %d", writes)
	}
	exec("UPDATE project_environment SET namespace='changed' WHERE id=?", env.ID)
	if _, err := e.Execute(ctx, ExecutionRequest{Run: *newRun, Task: newRun.Tasks[0]}); err == nil {
		t.Fatal("queued target rebinding not blocked")
	}
	if writes != 1 {
		t.Fatal("target drift caused a patch")
	}
}
