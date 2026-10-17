package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

func testMySQLServiceRollback(t *testing.T, db *sql.DB) {
	ctx := actorContext(7, "rollback-tester")
	repo := NewRepository(db)
	store := NewSQLDeploymentSnapshotStore(db)
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO app_service(id,project_id,name,code,status) VALUES(901,19,'server','rollback-server','active'),(902,19,'gateway','rollback-gateway','active')`)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status) VALUES(901,19,'QA','rollback-qa','k8s_namespace',2,'rollback-qa','validated','active'),(902,19,'Prod','rollback-prod','k8s_namespace',3,'rollback-prod','validated','active')`)
	// Legacy deleted projects may retain an active environment row. Recreating a
	// project with the same namespace must not be mistaken for two live targets.
	exec("INSERT INTO project(id,status) VALUES(20,'deleted')")
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status) VALUES(903,20,'Deleted QA','deleted-qa','k8s_namespace',2,'rollback-qa','validated','active')`)
	env, err := repo.FindEnvironmentTarget(ctx, 901)
	if err != nil {
		t.Fatal(err)
	}
	nodes := []Node{}
	for _, sid := range []int64{901, 902} {
		nodes = append(nodes, Node{Key: fmt.Sprintf("deploy-%d", sid), Type: NodeYAMLDeploy, Config: mustJSON(YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: 1})})
	}
	wf := &Workflow{ProjectID: 19, TargetEnvironmentID: 901, Name: "Rollback QA", Code: "rollback-qa", DeliveryType: DeliveryTypeK8sYAML, Nodes: nodes}
	wid, err := repo.Create(ctx, wf)
	if err != nil {
		t.Fatal(err)
	}
	create := func(status string, ids ...int64) *WorkflowRun {
		t.Helper()
		run := &WorkflowRun{WorkflowID: wid, ProjectID: 19, EnvironmentID: 901, DefinitionVersion: 1, TriggerUserID: 7, TriggerType: "manual", Status: status, ContextSnapshot: mustJSON(map[string]interface{}{"target": env})}
		tasks := []TaskRun{}
		for _, sid := range ids {
			node := nodes[0]
			if sid == 902 {
				node = nodes[1]
			}
			tasks = append(tasks, TaskRun{NodeKey: node.Key, NodeType: node.Type, Status: RunSucceeded, NodeSnapshot: mustJSON(node)})
		}
		id, err := repo.CreateRun(ctx, run, tasks)
		if err != nil {
			t.Fatal(err)
		}
		run, err = repo.FindRun(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	manifest := func(sid int64, letter string) []byte {
		return []byte(fmt.Sprintf("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app-%d\n  namespace: rollback-qa\nspec:\n  selector:\n    matchLabels: {app: app-%d}\n  template:\n    metadata:\n      labels: {app: app-%d}\n    spec:\n      containers:\n      - name: app\n        image: registry.example/app@sha256:%s\n", sid, sid, sid, strings.Repeat(letter, 64)))
	}
	save := func(run *WorkflowRun, sid int64, letter string, applied bool) *DeploymentSnapshot {
		t.Helper()
		var task TaskRun
		for _, item := range run.Tasks {
			if item.NodeKey == fmt.Sprintf("deploy-%d", sid) {
				task = item
			}
		}
		s, err := NewYAMLDeploymentSnapshot(*run, task, YAMLDeployConfig{ServiceID: sid, ServiceRevisionID: 1}, YAMLTarget{ClusterID: 2, Namespace: "rollback-qa", Manifest: manifest(sid, letter)})
		if err != nil {
			t.Fatal(err)
		}
		if err = store.Prepare(ctx, s); err != nil {
			t.Fatal(err)
		}
		if applied {
			if err = store.MarkApplied(ctx, run.ID, task.NodeKey, NodeYAMLDeploy); err != nil {
				t.Fatal(err)
			}
		}
		s, err = scanDeployment(db, ctx, "workflow_run_id=? AND node_key=?", run.ID, task.NodeKey)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	// The first run failed because of another service; server's own success must
	// remain an eligible rollback target.
	old := create(RunFailed, 901, 902)
	target := save(old, 901, "a", true)
	other := save(old, 902, "c", true)
	currentRun := create(RunSucceeded, 901, 902)
	current := save(currentRun, 901, "b", true)
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}, {Resource: "deployment", Action: "rollback", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}})}
	options, code := logic.RollbackOptions(ctx, 7, currentRun.ID)
	if code != 0 || len(options) != 2 || len(options[0].Versions) != 2 {
		t.Fatalf("options=%+v code=%d", options, code)
	}
	preview, code, msg := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || preview.BlockedReason != "" || preview.Current.SnapshotID != current.ID {
		t.Fatalf("preview=%+v code=%d msg=%s", preview, code, msg)
	}
	if _, code, _ := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, other.ID); code == 0 {
		t.Fatal("another service snapshot accepted")
	}
	exec("UPDATE project_environment SET namespace='wrong-prod' WHERE id=901")
	if _, code, _ := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID); code == 0 {
		t.Fatal("moved namespace accepted")
	}
	exec("UPDATE project_environment SET namespace='rollback-qa' WHERE id=901")
	// Sharing a namespace defeats environment isolation, even across projects.
	exec("UPDATE project_environment SET cluster_id=2,namespace='rollback-qa' WHERE id=902")
	if _, code, _ := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID); code == 0 {
		t.Fatal("shared environment namespace accepted")
	}
	exec("UPDATE project_environment SET cluster_id=3,namespace='rollback-prod' WHERE id=902")
	// Replaying another service's resource must be rejected before submission.
	exec("UPDATE deployment_snapshot SET rendered_manifest=? WHERE id=?", target.RenderedManifest, other.ID)
	blocked, code, _ := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || blocked == nil || !strings.Contains(blocked.BlockedReason, "其他服务") {
		t.Fatalf("shared resource accepted: %+v code=%d", blocked, code)
	}
	exec("UPDATE deployment_snapshot SET rendered_manifest=? WHERE id=?", other.RenderedManifest, other.ID)
	// Helm must also refuse to operate a Release claimed by a different service.
	exec("UPDATE deployment_snapshot SET provider=?,helm_release_name='shared-release' WHERE id=?", NodeHelmDeploy, other.ID)
	helmTarget := *target
	helmTarget.Provider, helmTarget.HelmReleaseName = NodeHelmDeploy, "shared-release"
	if err := checkRollbackResourceOwnership(ctx, db, &helmTarget); err == nil {
		t.Fatal("shared Helm release accepted")
	}
	exec("UPDATE deployment_snapshot SET provider=?,helm_release_name='' WHERE id=?", NodeYAMLDeploy, other.ID)
	updateManifest := func(id int64, raw []byte) {
		t.Helper()
		digest := sha256.Sum256(raw)
		exec("UPDATE deployment_snapshot SET rendered_manifest=?,manifest_digest=? WHERE id=?", raw, hex.EncodeToString(digest[:]), id)
	}
	mutable := []byte(strings.Replace(string(target.RenderedManifest), "@sha256:"+strings.Repeat("a", 64), ":latest", 1))
	updateManifest(target.ID, mutable)
	blocked, code, _ = logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || blocked == nil || !strings.Contains(blocked.BlockedReason, "镜像摘要") {
		t.Fatalf("mutable image accepted: %+v code=%d", blocked, code)
	}
	updateManifest(target.ID, target.RenderedManifest)
	extra := append(append([]byte(nil), current.RenderedManifest...), []byte("\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: extra-config\n  namespace: rollback-qa\ndata:\n  mode: test\n")...)
	updateManifest(current.ID, extra)
	blocked, code, _ = logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || blocked == nil || !strings.Contains(blocked.BlockedReason, "ConfigMap/extra-config") {
		t.Fatalf("removed resource silently accepted: %+v code=%d", blocked, code)
	}
	updateManifest(current.ID, current.RenderedManifest)
	// Recheck after opening the drawer: another complete deployment invalidates
	// the old confirmation, without creating any rollback run.
	stale := RollbackRequest{OriginRunID: currentRun.ID, ServiceID: 901, ExpectedToken: preview.Token}
	nextRun := create(RunSucceeded, 901)
	newCurrent := save(nextRun, 901, "d", true)
	if _, code, _ := logic.SubmitRollback(ctx, 7, target.ID, "rollback-stale-key", stale); code == 0 {
		t.Fatal("stale confirmation accepted")
	}
	preview, code, msg = logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || preview.BlockedReason != "" {
		t.Fatalf("new preview %d %s", code, msg)
	}
	req := RollbackRequest{OriginRunID: currentRun.ID, ServiceID: 901, ExpectedToken: preview.Token}
	type result struct {
		id   int64
		code int
		key  string
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("rollback-race-%d", i)
			id, code, _ := logic.SubmitRollback(ctx, 7, target.ID, key, req)
			results <- result{id, code, key}
		}(i)
	}
	wg.Wait()
	close(results)
	var winning result
	success := 0
	for r := range results {
		if r.code == 0 {
			success++
			winning = r
		}
	}
	if success != 1 {
		t.Fatalf("concurrent rollbacks accepted=%d", success)
	}
	saved, err := repo.FindRun(ctx, winning.id)
	if err != nil || saved.WorkflowID != wid || saved.RunNumber != 4 || len(saved.Tasks) != 1 {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
	// Same service is reserved while this rollback is pending; an unrelated
	// service in the same namespace may still be published.
	normal := &WorkflowRun{WorkflowID: wid, ProjectID: 19, EnvironmentID: 901, DefinitionVersion: 1, Status: RunPending, ContextSnapshot: mustJSON(map[string]interface{}{"target": env})}
	if _, err := repo.CreateRun(ctx, normal, []TaskRun{{NodeKey: nodes[0].Key, NodeType: NodeYAMLDeploy, NodeSnapshot: mustJSON(nodes[0]), Status: RunPending}}); err == nil {
		t.Fatal("normal deployment overlapped active rollback")
	}
	independent := create(RunSucceeded, 902)
	if independent.ID <= winning.id {
		t.Fatal("independent service did not create a run")
	}
	provider := &fakeYAMLProvider{}
	executor := NewYAMLRollbackExecutor(store, provider)
	// Ownership/binding may change while queued; workers must recheck both.
	exec("UPDATE deployment_snapshot SET rendered_manifest=? WHERE id=?", target.RenderedManifest, other.ID)
	if _, err := executor.Execute(ctx, ExecutionRequest{Run: *saved, Task: saved.Tasks[0]}); err == nil {
		t.Fatal("worker ignored changed ownership")
	}
	exec("UPDATE deployment_snapshot SET rendered_manifest=? WHERE id=?", other.RenderedManifest, other.ID)
	exec("UPDATE project_environment SET cluster_id=2,namespace='rollback-qa' WHERE id=902")
	if _, err := executor.Execute(ctx, ExecutionRequest{Run: *saved, Task: saved.Tasks[0]}); err == nil {
		t.Fatal("worker ignored shared environment namespace")
	}
	exec("UPDATE project_environment SET cluster_id=3,namespace='rollback-prod' WHERE id=902")
	result1, err := executor.Execute(ctx, ExecutionRequest{Run: *saved, Task: saved.Tasks[0]})
	if err != nil {
		t.Fatal(err)
	}
	if string(provider.request.Manifest) != string(target.RenderedManifest) {
		t.Fatal("rollback did not use exact historical YAML")
	}
	// A restarted worker reuses its own prepared output and the provider key.
	recovered, err := NewYAMLRollbackExecutor(store, provider).Execute(ctx, ExecutionRequest{Run: *saved, Task: saved.Tasks[0]})
	if err != nil || recovered.ProviderRunID != result1.ProviderRunID {
		t.Fatalf("recovery failed: %v", err)
	}
	task := saved.Tasks[0]
	task.ProviderRunID = result1.ProviderRunID
	if _, err = executor.Reconcile(ctx, ExecutionRequest{Run: *saved, Task: task}); err != nil {
		t.Fatal(err)
	}
	if _, err = executor.Reconcile(ctx, ExecutionRequest{Run: *saved, Task: task}); err != nil {
		t.Fatal("completed snapshot not idempotent", err)
	}
	exec("UPDATE workflow_run SET status='succeeded' WHERE id=?", saved.ID)
	deployed, err := LoadCurrentDeployment(ctx, db, 19, 901, 901)
	if err != nil || deployed == nil || deployed.RunID != saved.ID || deployed.SnapshotID == target.ID || deployed.SnapshotID == newCurrent.ID {
		t.Fatalf("rollback missing current snapshot: %+v %v", deployed, err)
	}
	if id, code, _ := logic.SubmitRollback(ctx, 7, target.ID, winning.key, req); code != 0 || id != saved.ID {
		t.Fatal("retry did not return original run")
	}
	var gatewayLatest int64
	if err := db.QueryRow("SELECT id FROM deployment_snapshot WHERE service_id=902 ORDER BY id DESC LIMIT 1").Scan(&gatewayLatest); err != nil || gatewayLatest != other.ID {
		t.Fatal("other service snapshot changed")
	}
	same, code, _ := logic.PreviewRollback(ctx, 7, currentRun.ID, 901, target.ID)
	if code != 0 || same.BlockedReason == "" {
		t.Fatal("same-version rollback not blocked")
	}
	// A worker that wakes after a newer deployment may not overwrite it.
	newer := create(RunSucceeded, 901)
	save(newer, 901, "e", true)
	if _, err := executor.Execute(context.Background(), ExecutionRequest{Run: *saved, Task: saved.Tasks[0]}); err == nil {
		t.Fatal("stale worker overwrote newer version")
	}
}
