package build

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
	"github.com/zeromicro/go-zero/core/conf"
)

// Opt-in integration test: copies schemas only into a uniquely named database.
// Never changes the source database's tables or build records.
func TestMySQLNullableBuildAndCommandRetry(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("test configuration could not be loaded")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("AXIOPS_TEST_MYSQL_DSN is not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid integration test DSN")
	}
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("axiops_build_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName, cfg.ParseTime = schema, true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, table := range []string{"build_run", "build_log", "build_command", "build_number_sequence", "build_concurrency_policy", "build_platform_settings", "build_config", "credential", "image_registry", "app_service", "code_source", "notification_subscription", "notification_outbox", "workflow_run", "workflow_task_run"} {
		if _, err := db.Exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO build_concurrency_policy(scope_type,scope_id,max_running) VALUES('platform',0,10)"); err != nil {
		t.Fatal(err)
	}
	ctx, repo := context.Background(), NewRepository(db)
	if _, err := db.Exec("CREATE TABLE project(id BIGINT PRIMARY KEY,status VARCHAR(16) DEFAULT 'active')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO project(id) VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO app_service(id,project_id,name,code,registry_id) VALUES(1,1,'test','test',1)"); err != nil {
		t.Fatal(err)
	}
	id, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 1, ProjectID: 1, BuildNumber: 1, Status: StatusQueued, Provider: KubeJobProviderName})
	if err != nil {
		t.Fatal(err)
	}
	run, err := repo.FindRunByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if run.GitChangedFiles != "[]" || run.Stages != "[]" || run.ErrorMessage != "" || run.StartedAt != nil {
		t.Fatalf("nullable fields not normalized: %+v", run)
	}
	logs, err := repo.ListLogsByRunID(ctx, id)
	if err != nil || logs == nil || len(logs) != 0 {
		t.Fatalf("empty logs must be an array: %+v %v", logs, err)
	}
	logic := NewLogic(db, allowRegistryAuth{})
	for _, values := range [][2]string{{"1", "2Gi"}, {"", ""}, {"500m", ""}, {"", "1Gi"}} {
		settings, code := logic.SavePlatformBuildSettings(ctx, 1, SavePlatformBuildSettingsRequest{
			TimeoutSeconds: 1800, ResourceCPU: values[0], ResourceMemory: values[1],
		})
		if code != 0 || settings == nil || settings.ResourceCPU != values[0] || settings.ResourceMemory != values[1] {
			t.Fatalf("optional platform resources did not round-trip: %+v code=%d", settings, code)
		}
	}
	for _, status := range []string{StatusQueued, StatusRunning, StatusTimeout} {
		if _, err := db.Exec("UPDATE build_run SET status=?,error_message='Insufficient cpu' WHERE id=?", status, id); err != nil {
			t.Fatal(err)
		}
		logs, code := logic.GetLogs(ctx, 1, id)
		if code != 0 || len(logs) != 1 || !strings.Contains(logs[0].LogText, status) || !strings.Contains(logs[0].LogText, "Insufficient cpu") {
			t.Fatalf("historical/queued diagnostic missing: %+v code=%d", logs, code)
		}
	}
	if _, err := db.Exec("UPDATE build_run SET status='queued',error_message=NULL WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []BuildLog{
		{BuildRunID: id, Stage: "system", LogText: "waiting for cpu", LineCount: 1},
		{BuildRunID: id, Stage: "build", LogText: "clone started", LineCount: 1},
		{BuildRunID: id, Stage: "system", LogText: "pulling image", LineCount: 1},
	} {
		if _, err := repo.CreateLog(ctx, &entry); err != nil {
			t.Fatal(err)
		}
	}
	logs, err = repo.ListLogsByRunID(ctx, id)
	if err != nil || len(logs) != 2 || logs[0].LogText != "pulling image" || logs[1].LogText != "clone started" {
		t.Fatalf("stage upserts corrupted logs: %+v %v", logs, err)
	}
	if _, err := db.Exec("UPDATE build_log SET log_text=NULL WHERE build_run_id=? AND stage='build'", id); err != nil {
		t.Fatal(err)
	}
	logs, err = repo.ListLogsByRunID(ctx, id)
	if err != nil || len(logs) != 2 || logs[1].LogText != "" {
		t.Fatalf("NULL log text not readable: %+v %v", logs, err)
	}
	runs, total, err := repo.ListRuns(ctx, authz.ScopeSet{AllowAll: true}, 1, 1, "", 1, 10)
	if err != nil || total != 1 || len(runs) != 1 {
		t.Fatalf("list runs: total=%d err=%v", total, err)
	}
	if _, err := db.Exec(`UPDATE build_platform_settings SET enabled=1`); err != nil {
		t.Fatal(err)
	}
	command, err := repo.ClaimCommand(ctx, "test-worker")
	if err != nil || command == nil {
		t.Fatalf("claim NULL last_error: %v", err)
	}
	if command.LastError != "" {
		t.Fatal("expected empty initial error")
	}
	if err := repo.FailCommand(ctx, command.ID, "test-worker", "temporary provider outage", 3, time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE build_command SET next_attempt_at=CURRENT_TIMESTAMP"); err != nil {
		t.Fatal(err)
	}
	retry, err := repo.ClaimCommand(ctx, "test-worker")
	if err != nil || retry == nil || retry.ID != command.ID || retry.Attempts != 2 {
		t.Fatalf("retry stranded after failure: command=%+v err=%v", retry, err)
	}
	if _, err := db.Exec("UPDATE build_command SET locked_at=?", time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RequeueStaleCommands(ctx, time.Now().Add(-time.Minute), 3); err != nil {
		t.Fatal(err)
	}
	recovered, err := repo.ClaimCommand(ctx, "replacement-worker")
	if err != nil || recovered == nil || recovered.Attempts != 2 {
		t.Fatalf("stale submission stranded: %v", err)
	}

	if _, err := db.Exec("INSERT INTO credential(id,name,type,data,project_id) VALUES(1,'test','registry_auth','opaque',0)"); err != nil {
		t.Fatal(err)
	}
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer registry.Close()
	handler := NewHandler(NewLogic(db, allowRegistryAuth{}).WithCredentialDecryptor(registryTestDecryptor{}))
	for _, body := range []string{
		`{"name":"ACR","type":"acr","url":"registry.example/team","credential_id":1}`,
		`{"id":1,"name":"ACR updated","type":"acr","url":"registry.example/team","credential_id":1,"namespaces":["abc","dbc"]}`,
	} {
		body = strings.ReplaceAll(body, "registry.example/team", registry.URL+"/team")
		recorder := httptest.NewRecorder()
		handler.SaveImageRegistry(recorder, httptest.NewRequest("POST", "/api/v1/build-registries", strings.NewReader(body)))
		if recorder.Code != 200 {
			t.Fatalf("save registry: %s", recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	handler.ListImageRegistries(recorder, httptest.NewRequest("GET", "/api/v1/build-registries?project_id=1", nil))
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "ACR updated") || !strings.Contains(recorder.Body.String(), `"namespaces":["abc","dbc"]`) {
		t.Fatalf("list registry: %s", recorder.Body.String())
	}
	t.Run("atomic config and binding", func(t *testing.T) { testAtomicConfigBinding(t, db) })
	store := NewSQLRegistryAuthStore(db, registryTestDecryptor{})
	auth, err := store.Load(ctx, 1, 1)
	if err != nil || auth == nil || auth.Username != "robot" || strings.Join(auth.Namespaces, ",") != "abc,dbc" {
		t.Fatalf("load registry credential: %v", err)
	}
	registryHost := strings.TrimPrefix(registry.URL, "http://")
	for _, imageName := range []string{"risk_server", registryHost + "/team/custom", registryHost + "/abc/risk_server"} {
		if _, err := db.Exec("UPDATE app_service SET image_name=? WHERE id=1", imageName); err != nil {
			t.Fatal(err)
		}
		resolved, err := appsvc.NewRepository(db).ResolveImageRepository(ctx, 1)
		want := imageName
		if imageName == "risk_server" {
			want = registryHost + "/team/risk_server"
		}
		if err != nil || resolved != want {
			t.Fatalf("namespace resolution: got=%q want=%q err=%v", resolved, want, err)
		}
	}
	if _, err := db.Exec("UPDATE credential SET project_id=2 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx, 1, 1); err == nil {
		t.Fatal("cross-project registry credential was loaded")
	}
	testDurableBuildLifecycle(t, db)
	t.Run("workflow idempotency and cancellation fence", func(t *testing.T) { testWorkflowSubmissionFence(t, db) })
	t.Run("deleted project rejects submission without adding work", func(t *testing.T) {
		var beforeRuns, beforeCommands, afterRuns, afterCommands int
		if err := db.QueryRow("SELECT COUNT(*) FROM build_run").Scan(&beforeRuns); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM build_command").Scan(&beforeCommands); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("UPDATE project SET status='deleted' WHERE id=1"); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 1, ProjectID: 1, BuildNumber: 999, Status: StatusQueued}); err != projectstate.ErrDeleted {
			t.Fatalf("deleted project accepted: %v", err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM build_run").Scan(&afterRuns); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM build_command").Scan(&afterCommands); err != nil {
			t.Fatal(err)
		}
		if beforeRuns != afterRuns || beforeCommands != afterCommands {
			t.Fatal("rejected submission added work")
		}
	})
}

func testWorkflowSubmissionFence(t *testing.T, db *sql.DB) {
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO workflow_run(id,workflow_id,project_id,environment_id,definition_version,trigger_type,trigger_user_id,status,context_snapshot) VALUES(99,1,1,0,1,'manual',1,'running','{}')`)
	exec(`INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,node_snapshot,depends_on,status,locked_by,lease_until) VALUES(99,'build','build','{}','[]','running','worker',DATE_ADD(CURRENT_TIMESTAMP,INTERVAL 1 MINUTE))`)
	r := NewRepository(db)
	ctx := context.Background()
	input := BuildRun{ServiceID: 1, ProjectID: 1, BuildNumber: 100, Status: StatusQueued, Provider: KubeJobProviderName, WorkflowRunID: 99, WorkflowNodeKey: "build", TriggerType: "workflow", TriggerKey: "workflow:99:build"}
	id, err := r.CreateRunWithSubmitCommand(ctx, &input)
	if err != nil {
		t.Fatal(err)
	}
	input.BuildNumber++
	duplicate, err := NewRepository(db).CreateRunWithSubmitCommand(ctx, &input)
	if err != nil || id != duplicate {
		t.Fatalf("duplicate build %d %d %v", id, duplicate, err)
	}
	found, err := r.FindRunByTriggerKey(ctx, 1, input.TriggerKey)
	if err != nil || found == nil || found.ID != id {
		t.Fatal("cannot recover submission")
	}
	exec(`UPDATE workflow_run SET status='cancelling' WHERE id=99`)
	input.TriggerKey = "workflow:99:build:retry:1"
	if _, err := r.CreateRunWithSubmitCommand(ctx, &input); err == nil {
		t.Fatal("submitted work after cancellation")
	}
}

func testDurableBuildLifecycle(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	repo := NewRepository(db)
	// Build submission now locks an active service. Create all referenced
	// services in this disposable schema instead of relying on absent fixtures.
	for _, id := range []int64{11, 12, 21, 22} {
		if _, err := db.Exec(`INSERT INTO app_service(id,project_id,name,code,status) VALUES(?,1,?,?,'active')`, id, fmt.Sprintf("Fixture %d", id), fmt.Sprintf("fixture-%d", id)); err != nil {
			t.Fatal(err)
		}
	}
	// This is the isolated schema created above; close earlier fixture runs.
	if _, err := db.Exec(`UPDATE build_run SET status='failed'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE build_command SET status='succeeded'`); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveConcurrencyPolicy(ctx, 0, 1, 7200, 1); err != nil {
		t.Fatal(err)
	}
	first, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 11, ProjectID: 1, BuildNumber: 1, Status: StatusQueued, Provider: KubeJobProviderName})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 12, ProjectID: 1, BuildNumber: 1, Status: StatusQueued, Provider: KubeJobProviderName})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimCommand(ctx, "first-worker")
	if err != nil || claimed == nil || claimed.BuildRunID != first {
		t.Fatalf("first claim %+v %v", claimed, err)
	}
	if next, err := repo.ClaimCommand(ctx, "other-worker"); err != nil || next != nil {
		t.Fatalf("concurrency limit bypassed %+v %v", next, err)
	}
	if _, err := db.Exec(`UPDATE build_command SET attempts=3,locked_at=? WHERE id=?`, time.Now().Add(-10*time.Minute), claimed.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RequeueStaleCommands(ctx, time.Now().Add(-time.Minute), 3); err != nil {
		t.Fatal(err)
	}
	resumed, err := repo.ClaimCommand(ctx, "restart-worker")
	if err != nil || resumed == nil || resumed.BuildRunID != first || resumed.Attempts != 3 {
		t.Fatalf("last-attempt crash not recovered %+v %v", resumed, err)
	}
	// Cancellation must wait for an in-flight submit before discovering its Job.
	if err := repo.RequestCancel(ctx, first); err != nil {
		t.Fatal(err)
	}
	if next, err := repo.ClaimCommand(ctx, "cancel-worker"); err != nil || next != nil {
		t.Fatalf("cancel raced unacknowledged submit %+v %v", next, err)
	}
	resolver := NewSQLExecutionResolver(db, "ci")
	if err := resolver.MarkSubmitted(ctx, first, ProviderExecution{JobName: "axiops-build-fixture"}); err != nil {
		t.Fatal(err)
	}
	run, _ := repo.FindRunByID(ctx, first)
	if run.Status != StatusCancelling || run.JobName == "" {
		t.Fatal("submission erased concurrent cancellation")
	}
	if err := repo.CompleteCommand(ctx, resumed.ID, "restart-worker"); err != nil {
		t.Fatal(err)
	}
	cancel, err := repo.ClaimCommand(ctx, "cancel-worker")
	if err != nil || cancel == nil || cancel.CommandType != CommandCancel {
		t.Fatalf("cancel not claimed %+v %v", cancel, err)
	}
	if err := repo.RetryUncertainCommand(ctx, cancel.ID, "cancel-worker", "cluster offline", time.Second); err != nil {
		t.Fatal(err)
	}
	run, _ = repo.FindRunByID(ctx, first)
	if run.Status != StatusCancelling {
		t.Fatal("uncertain cancellation freed capacity")
	}
	if err := repo.TransitionRunStatus(ctx, first, StatusCancelled, ""); err != nil {
		t.Fatal(err)
	}
	cleanup, err := repo.FindCleanupRun(ctx)
	if err != nil || cleanup == nil || cleanup.ID != first {
		t.Fatalf("terminal cleanup was not durable %+v %v", cleanup, err)
	}
	settings, err := repo.FindPlatformBuildSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	policy := SavePlatformBuildSettingsRequest{Enabled: true, ClusterID: 2, Namespace: "ci", TimeoutSeconds: 1800, ExecutionPolicy: ExecutionPolicy{MaxBuildsPerNode: 2, LogRetentionDays: 30}}
	if err := repo.SavePlatformBuildSettings(ctx, policy, 1); err != ErrBuildPoolBusy {
		t.Fatalf("pool moved before old cleanup: %v (previous %+v)", err, settings)
	}
	if err := repo.FinishCleanup(ctx, first, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.SavePlatformBuildSettings(ctx, policy, 1); err != nil {
		t.Fatal(err)
	}
	settings, err = repo.FindPlatformBuildSettings(ctx)
	if err != nil || settings.ExecutionPolicy.MaxBuildsPerNode != 2 || settings.ExecutionPolicy.LogRetentionDays != 30 {
		t.Fatal("execution policy failed to round-trip")
	}
	if _, err := repo.CreateLog(ctx, &BuildLog{BuildRunID: first, Stage: "build", LogURL: "s3://fixture/build.log", LogText: "saved tail"}); err != nil {
		t.Fatal(err)
	}
	logs, err := NewLogic(db, allowRegistryAuth{}).WithLogArchive(unavailableArchive{}).listLogs(ctx, first)
	if err != nil || len(logs) != 1 || !strings.Contains(logs[0].LogText, "saved tail") || !logs[0].Truncated {
		t.Fatalf("archive outage hid SQL log: %+v %v", logs, err)
	}
	if _, err := db.Exec(`UPDATE build_run SET finished_at=? WHERE id=?`, time.Now().AddDate(0, 0, -31), first); err != nil {
		t.Fatal(err)
	}
	expired, err := repo.FindExpiredLog(ctx)
	if err != nil || expired == nil || expired.BuildRunID != first {
		t.Fatalf("retention did not select old log %+v %v", expired, err)
	}
	if err := repo.ExpireLog(ctx, expired.ID); err != nil {
		t.Fatal(err)
	}
	logs, err = repo.ListLogsByRunID(ctx, first)
	if err != nil || len(logs) != 1 || logs[0].LogURL != "" || !strings.Contains(logs[0].LogText, "清理") {
		t.Fatalf("expired log metadata missing %+v %v", logs, err)
	}
	fenced := NewRepository(db).WithWorkerSettings(settings)
	policy.ResourceCPU = "500m"
	if err := repo.SavePlatformBuildSettings(ctx, policy, 1); err != nil {
		t.Fatal(err)
	}
	if next, err := fenced.ClaimCommand(ctx, "old-settings-worker"); err != nil || next != nil {
		t.Fatalf("old executor settings still dispatch %+v %v", next, err)
	}
	next, err := repo.ClaimCommand(ctx, "next-worker")
	if err != nil || next == nil || next.BuildRunID != second {
		t.Fatalf("capacity not released %+v %v", next, err)
	}
	if err := repo.SaveConcurrencyPolicy(ctx, 0, 3, 7200, 1); err != nil {
		t.Fatal(err)
	}
	snapshot := `{"config":{"cache_enabled":true,"cache_strategy":"pvc"}}`
	cached1, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 21, ProjectID: 1, BuildNumber: 1, Status: StatusQueued, Provider: KubeJobProviderName, ConfigSnapshot: &snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 21, ProjectID: 1, BuildNumber: 2, Status: StatusQueued, Provider: KubeJobProviderName, ConfigSnapshot: &snapshot}); err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateRunWithSubmitCommand(ctx, &BuildRun{ServiceID: 22, ProjectID: 1, BuildNumber: 1, Status: StatusQueued, Provider: KubeJobProviderName, ConfigSnapshot: &snapshot})
	if err != nil {
		t.Fatal(err)
	}
	if next, err := repo.ClaimCommand(ctx, "cache-worker"); err != nil || next == nil || next.BuildRunID != cached1 {
		t.Fatalf("first cached build not claimed %+v %v", next, err)
	}
	if next, err := repo.ClaimCommand(ctx, "parallel-worker"); err != nil || next == nil || next.BuildRunID != other {
		t.Fatalf("shared cache ran concurrently or blocked unrelated service %+v %v", next, err)
	}
}

type allowRegistryAuth struct{ denyRegistryAuth }

type denyConfigBindingAuth struct{ allowRegistryAuth }

func (denyConfigBindingAuth) Authorize(_ context.Context, _ int64, request authz.Request) (bool, error) {
	return request.Resource != "service", nil
}

func testAtomicConfigBinding(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	repo := NewRepository(db)
	logic := NewLogic(db, allowRegistryAuth{})
	registryID := int64(1)
	url, branch := "https://git.example/team/api.git", "release"
	req := &CreateBuildConfigRequest{ServiceID: 1, BuildMode: "dockerfile", ImageRepo: "registry.example/abc/api",
		ServiceBinding: &BuildServiceBinding{RegistryID: &registryID, RepoURL: &url, RepoBranch: &branch}}
	if code := logic.SaveBuildConfig(ctx, req, 1); code != errcode.CodeSuccess {
		t.Fatalf("combined save: code=%d", code)
	}
	assertValues := func(wantURL, wantBranch, wantImage string) {
		t.Helper()
		svc, err := appsvc.NewRepository(db).FindByID(ctx, 1)
		if err != nil || svc == nil || svc.RepoURL != wantURL || svc.RepoBranch != wantBranch || svc.ImageName != wantImage {
			t.Fatalf("unexpected binding: %+v err=%v", svc, err)
		}
		config, err := repo.FindConfigByServiceID(ctx, 1)
		if err != nil || config == nil || config.ImageRepo != wantImage {
			t.Fatalf("unexpected config: %+v err=%v", config, err)
		}
	}
	assertValues(url, branch, req.ImageRepo)
	config, _ := repo.FindConfigByServiceID(ctx, 1)
	badJSON, changedBranch := "{invalid", "must-rollback"
	config.EnvVars = &badJSON
	config.ImageRepo = "registry.example/abc/must-rollback"
	if err := repo.SaveConfigWithBinding(ctx, config, &BuildServiceBinding{RepoBranch: &changedBranch, RegistryID: &registryID}); err == nil {
		t.Fatal("invalid config unexpectedly committed")
	}
	assertValues(url, branch, req.ImageRepo)
	if code := NewLogic(db, denyConfigBindingAuth{}).SaveBuildConfig(ctx, req, 1); code != errcode.CodeNoPermission {
		t.Fatalf("combined save bypassed service:update: %d", code)
	}
	foreignSource := int64(991)
	if _, err := db.Exec(`INSERT INTO code_source(id,project_id,name,type,url) VALUES(991,2,'foreign','gitea','https://git.example')`); err != nil {
		t.Fatal(err)
	}
	req.ServiceBinding.CodeSourceID = &foreignSource
	if code := logic.SaveBuildConfig(ctx, req, 1); code != errcode.CodeNoPermission {
		t.Fatalf("cross-project code source accepted: %d", code)
	}
	req.ServiceBinding.CodeSourceID = nil
	if _, err := db.Exec(`INSERT INTO image_registry(id,project_id,name,type,url) VALUES(992,2,'foreign','harbor','https://foreign.example')`); err != nil {
		t.Fatal(err)
	}
	foreignRegistry := int64(992)
	req.ServiceBinding.RegistryID = &foreignRegistry
	if code := logic.SaveBuildConfig(ctx, req, 1); code != errcode.CodeNoPermission {
		t.Fatalf("cross-project registry accepted: %d", code)
	}
	assertValues(url, branch, req.ImageRepo)
	// Legacy clients keep all service bindings and require only build_config:update.
	req.ServiceBinding = nil
	if code := NewLogic(db, denyConfigBindingAuth{}).SaveBuildConfig(ctx, req, 1); code != errcode.CodeSuccess {
		t.Fatalf("legacy save lost compatibility: %d", code)
	}
	assertValues(url, branch, req.ImageRepo)
}

func (allowRegistryAuth) Authorize(context.Context, int64, authz.Request) (bool, error) {
	return true, nil
}

type registryTestDecryptor struct{}

func (registryTestDecryptor) Decrypt(string) (string, error) {
	return `{"username":"robot","password":"test-password"}`, nil
}
