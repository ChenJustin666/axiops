package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMySQLWorkflowEnvironmentBinding(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL integration test not configured")
	}
	var c config.PlatformAPIConfig
	if conf.Load(path, &c) != nil {
		t.Fatal("cannot load test configuration")
	}
	cfg, err := mysql.ParseDSN(c.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("cannot open test database")
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_workflow_binding_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName = schema
	cfg.ParseTime = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"workflow", "workflow_node", "workflow_run", "workflow_task_run", "project_environment", "app_service", "build_run", "notification_subscription", "notification_outbox"} {
		exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`")
	}
	exec("CREATE TABLE project(id BIGINT PRIMARY KEY,status VARCHAR(16) DEFAULT 'active')")
	exec("INSERT INTO project(id) VALUES(19)")
	repo := NewRepository(db)
	ctx := context.Background()
	build := &Workflow{ProjectID: 19, Name: "Build", Code: "build", DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "notify", Name: "Notice", Type: NodeNotify}}}
	id, err := repo.Create(ctx, build)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := repo.FindByID(ctx, id)
	if err != nil || saved.TargetEnvironmentID != 0 {
		t.Fatalf("read workflow err=%v", err)
	}
	scope, _ := authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}}).Scopes(ctx, 7, "workflow", "view")
	list, err := repo.List(ctx, scope, 19)
	if err != nil || len(list) != 1 || list[0].RequiresEnvironment {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	task := TaskRun{NodeKey: "notify", NodeType: NodeNotify, NodeSnapshot: mustJSON(build.Nodes[0]), Status: RunPending}
	run := &WorkflowRun{WorkflowID: id, ProjectID: 19, DefinitionVersion: 1, Status: RunPending, TriggerType: "manual", ContextSnapshot: mustJSON(map[string]interface{}{"target": nil})}
	if _, err := repo.CreateRun(ctx, run, []TaskRun{task}); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE workflow SET definition_version=2 WHERE id=?`, id)
	if _, err := repo.CreateRun(ctx, run, []TaskRun{task}); err == nil {
		t.Fatal("accepted stale workflow version")
	}
	exec(`INSERT INTO project_environment (id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status) VALUES (7,19,'QA','qa','k8s_namespace',2,'risk-qa','validated','active')`)
	exec(`UPDATE workflow SET target_environment_id=7 WHERE id=?`, id)
	target, err := repo.FindEnvironmentTarget(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	run.EnvironmentID = 7
	run.DefinitionVersion = 2
	run.ContextSnapshot = mustJSON(map[string]interface{}{"target": target})
	if _, err := repo.CreateRun(ctx, run, []TaskRun{task}); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE project_environment SET namespace='risk-prod' WHERE id=7`)
	if _, err := repo.CreateRun(ctx, run, []TaskRun{task}); err == nil {
		t.Fatal("accepted changed target")
	}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM workflow_run`).Scan(&total); err != nil || total != 2 {
		t.Fatalf("run count=%d err=%v", total, err)
	}
	t.Run("cancel retry and recovery", func(t *testing.T) { testMySQLCancellationAndRetry(t, db, id) })
	t.Run("lease clock and concurrent claims", func(t *testing.T) { testMySQLLeaseRecoveryClock(t, db, id) })
	t.Run("ordered multi-service results", func(t *testing.T) { testMySQLOrderedResults(t, db, id) })
	t.Run("recorded and legacy executor accounts", func(t *testing.T) { testMySQLRunActors(t, db) })
	t.Run("guarded single-service rollback", func(t *testing.T) {
		exec("CREATE TABLE deployment_snapshot LIKE " + source + ".deployment_snapshot")
		testMySQLServiceRollback(t, db)
	})
	t.Run("guarded workload restart", func(t *testing.T) {
		exec("CREATE TABLE IF NOT EXISTS deployment_snapshot LIKE " + source + ".deployment_snapshot")
		testMySQLWorkloadRestart(t, db)
	})
	t.Run("environment permissions list detail and pagination", func(t *testing.T) {
		testMySQLWorkflowAuthorization(t, db)
	})
}

func testMySQLCancellationAndRetry(t *testing.T, db *sql.DB, workflowID int64) {
	ctx := context.Background()
	repo := NewRepository(db)
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE workflow SET target_environment_id=0 WHERE id=?`, workflowID)
	exec(`INSERT INTO app_service(id,project_id,name,code,status) VALUES(9,19,'Retry fixture','retry-fixture','active')`)
	create := func() int64 {
		t.Helper()
		run := &WorkflowRun{WorkflowID: workflowID, ProjectID: 19, DefinitionVersion: 2, Status: RunPending, TriggerType: "manual", ContextSnapshot: mustJSON(map[string]interface{}{})}
		tasks := []TaskRun{{NodeKey: "build", NodeType: NodeBuild, NodeSnapshot: mustJSON(Node{Key: "build", Type: NodeBuild, RetryLimit: 1, Config: mustJSON(BuildNodeConfig{ServiceID: 9})}), Status: RunPending}, {NodeKey: "notify", NodeType: NodeNotify, DependsOn: []string{"build"}, NodeSnapshot: mustJSON(Node{Key: "notify", Type: NodeNotify}), Status: RunPending}}
		id, err := repo.CreateRun(ctx, run, tasks)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	id := create()
	if ok, err := repo.ClaimTask(ctx, id, "build", "first", time.Minute); err != nil || !ok {
		t.Fatalf("claim %v %v", ok, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := repo.RenewTaskLease(ctx, id, "build", "first", time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	if ok, err := repo.ClaimTask(ctx, id, "build", "second", time.Minute); err != nil || ok {
		t.Fatal("live lease stolen")
	}
	if err := repo.MarkTaskSubmitted(ctx, id, "build", "first", ExecutionResult{ProviderRunID: "build:31"}); err != nil {
		t.Fatal(err)
	}
	run, err := repo.FindRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.ScheduleTaskRetry(ctx, *run, run.Tasks[0], "temporary failure"); err != nil || !ok {
		t.Fatalf("retry %v %v", ok, err)
	}
	if ok, err := repo.ClaimTask(ctx, id, "build", "second", time.Minute); err != nil || ok {
		t.Fatal("retry backoff ignored")
	}
	exec(`UPDATE workflow_task_run SET next_retry_at=DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 1 SECOND) WHERE workflow_run_id=?`, id)
	repo = NewRepository(db)
	run, err = repo.FindRun(ctx, id)
	if err != nil || run.Tasks[0].RetryCount != 1 || len(run.Tasks[0].RetryHistory) != 1 {
		t.Fatalf("history lost: %v", err)
	}
	if ok, err := repo.ClaimTask(ctx, id, "build", "second", time.Minute); err != nil || !ok {
		t.Fatal("retry not runnable")
	}
	if _, err := repo.RequestCancel(ctx, id); err != nil {
		t.Fatal(err)
	}
	if state, err := repo.RenewTaskLease(ctx, id, "build", "second", time.Minute); err != nil || state != RunCancelling {
		t.Fatalf("cancel lease %s %v", state, err)
	}
	if err := repo.FinishCancellation(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err = repo.FindRun(ctx, id)
	if err != nil || run.Status != RunCancelling || run.Tasks[1].Status != RunCancelled {
		t.Fatal("premature completion")
	}
	if err := repo.ReleaseTaskLease(ctx, id, "build", "second"); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteCancellingTask(ctx, id, "build"); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishCancellation(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err = repo.FindRun(ctx, id)
	if err != nil || run.Status != RunCancelled {
		t.Fatal("cancel did not converge")
	}
	if ok, err := repo.RequestCancel(ctx, id); err != nil || !ok {
		t.Fatal("cancel not idempotent")
	}

	id = create()
	if ok, err := repo.ClaimTask(ctx, id, "build", "worker", time.Minute); err != nil || !ok {
		t.Fatal(err)
	}
	if err := repo.CompleteTask(ctx, id, "build", "worker", RunFailed, "failed"); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishCancellation(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err = repo.FindRun(ctx, id)
	if err != nil || run.Status != RunFailed || run.Tasks[1].Status != RunCancelled {
		t.Fatal("failure did not stop pending nodes")
	}
}
