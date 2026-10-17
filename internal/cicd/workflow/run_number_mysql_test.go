package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/testschema"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
	"github.com/zeromicro/go-zero/core/conf"
)

// Replays the migration only in an isolated database, including when the live
// schema is already migrated. Never rewrites existing user history.
func TestMySQLWorkflowRunNumberMigrationAndConcurrency(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL not configured")
	}
	var c config.PlatformAPIConfig
	if conf.Load(path, &c) != nil {
		t.Fatal("cannot load config")
	}
	cfg, err := mysql.ParseDSN(c.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid DSN")
	}
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_number_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := admin.Exec("DROP DATABASE `" + schema + "`"); e != nil {
			t.Error(e)
		}
	}()
	cfg.DBName = schema
	cfg.ParseTime = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, e := db.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, table := range []string{"workflow", "workflow_run", "workflow_task_run"} {
		exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`")
	}
	exec("CREATE TABLE project LIKE " + source + ".project")
	exec("INSERT INTO project(id,name,code) VALUES(19,'Promotion','promotion'),(20,'Other','other')")
	var has int
	if err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='workflow_run' AND column_name='run_number'`).Scan(&has); err != nil {
		t.Fatal(err)
	}
	if has > 0 {
		exec(`ALTER TABLE workflow_run DROP INDEX uk_workflow_run_number, DROP COLUMN run_number`)
		exec(`ALTER TABLE workflow DROP COLUMN run_sequence`)
	}
	exec(`INSERT INTO workflow(id,project_id,name,code,delivery_type,status,definition_version) VALUES(1,19,'Old QA','qa','k8s_yaml','active',1),(2,19,'New Prod','prod','k8s_yaml','active',1),(3,20,'Other','other','k8s_yaml','active',1)`)
	exec(`INSERT INTO workflow_run(id,workflow_id,project_id,environment_id,definition_version,status,context_snapshot) VALUES(10,1,19,0,1,'failed','{}'),(20,3,20,0,1,'succeeded','{}'),(40,1,19,0,1,'succeeded','{}')`)
	migration, err := os.ReadFile("../../../migrations/044_workflow_run_numbers.sql")
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{}
	for _, line := range strings.Split(string(migration), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			lines = append(lines, line)
		}
	}
	for _, statement := range strings.Split(strings.Join(lines, "\n"), ";") {
		if strings.TrimSpace(statement) != "" {
			exec(statement)
		}
	}
	for _, want := range [][2]int64{{10, 1}, {20, 1}, {40, 2}} {
		var number int64
		if e := db.QueryRow(`SELECT run_number FROM workflow_run WHERE id=?`, want[0]).Scan(&number); e != nil || number != want[1] {
			t.Fatalf("backfill id=%d number=%d err=%v", want[0], number, e)
		}
	}
	repo := NewRepository(db)
	ctx := context.Background()
	run := func(wid int64) *WorkflowRun {
		return &WorkflowRun{WorkflowID: wid, ProjectID: 19, DefinitionVersion: 1, Status: RunPending, TriggerType: "manual", ContextSnapshot: mustJSON(map[string]interface{}{})}
	}
	first := run(2)
	if _, err = repo.CreateRun(ctx, first, nil); err != nil || first.RunNumber != 1 {
		t.Fatalf("new Job must start at 1: %d %v", first.RunNumber, err)
	}
	failed := run(2)
	duplicate := TaskRun{NodeKey: "same", NodeType: NodeNotify, Status: RunPending, NodeSnapshot: mustJSON(Node{Key: "same", Type: NodeNotify})}
	if _, err = repo.CreateRun(ctx, failed, []TaskRun{duplicate, duplicate}); err == nil {
		t.Fatal("invalid tasks committed")
	}
	var group sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() { defer group.Done(); _, e := repo.CreateRun(ctx, run(2), nil); results <- e }()
	}
	group.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatal(e)
		}
	}
	var count, distinct, maximum, sequence int
	if err = db.QueryRow(`SELECT COUNT(*),COUNT(DISTINCT run_number),MAX(run_number) FROM workflow_run WHERE workflow_id=2`).Scan(&count, &distinct, &maximum); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT run_sequence FROM workflow WHERE id=2`).Scan(&sequence); err != nil {
		t.Fatal(err)
	}
	if count != 13 || distinct != 13 || maximum != 13 || sequence != 13 {
		t.Fatalf("numbering count=%d distinct=%d max=%d seq=%d", count, distinct, maximum, sequence)
	}
	rollback := run(2)
	rollback.TriggerUserID = 7
	id, err := repo.createRollbackRun(ctx, rollback, []TaskRun{duplicate}, 1, "rollback-one", false)
	if err != nil || rollback.RunNumber != 14 {
		t.Fatalf("rollback number: %v", err)
	}
	repeated, err := repo.createRollbackRun(ctx, run(2), []TaskRun{duplicate}, 1, "rollback-one", false)
	if err == nil {
		t.Fatal("another user reused rollback key")
	}
	_ = repeated
	repeated, err = repo.createRollbackRun(ctx, rollback, []TaskRun{duplicate}, 1, "rollback-one", false)
	if err != nil || repeated != id {
		t.Fatal("rollback not idempotent")
	}
	for i := 0; i < 2; i++ {
		standalone := run(0)
		if _, err = repo.createRollbackRun(ctx, standalone, []TaskRun{duplicate}, 1, fmt.Sprintf("standalone-%d", i), false); err != nil {
			t.Fatal(err)
		}
	}
	old := run(1)
	if _, err = repo.CreateRun(ctx, old, nil); err != nil || old.RunNumber != 3 {
		t.Fatal("old Job counter did not continue")
	}
	t.Run("deleted project rejects normal and rollback runs", func(t *testing.T) {
		var before, after int
		if err := db.QueryRow("SELECT COUNT(*) FROM workflow_run").Scan(&before); err != nil {
			t.Fatal(err)
		}
		exec("UPDATE project SET status='deleted' WHERE id=19")
		defer exec("UPDATE project SET status='active' WHERE id=19")
		if _, err := repo.CreateRun(ctx, run(2), nil); err != projectstate.ErrDeleted {
			t.Fatalf("deleted project accepted normal run: %v", err)
		}
		for _, wid := range []int64{0, 2} {
			if _, err := repo.CreateRollbackRun(ctx, run(wid), nil, 1, "deleted-project"); err != projectstate.ErrDeleted {
				t.Fatalf("deleted project accepted rollback: %v", err)
			}
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM workflow_run").Scan(&after); err != nil {
			t.Fatal(err)
		}
		if before != after {
			t.Fatal("rejected submission changed history")
		}
	})
	t.Run("promote real persisted artifact with target configuration", func(t *testing.T) { testMySQLPromotion(t, db, source) })
}

func testMySQLPromotion(t *testing.T, db *sql.DB, source string) {
	exec := func(q string, args ...interface{}) {
		t.Helper()
		if _, e := db.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for _, table := range []string{"app_service", "service_revision", "cluster", "project_environment", "environment_service_override", "environment_resource", "variable_set", "variable_item", "build_run", "workflow_node", "deployment_snapshot"} {
		exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`")
	}
	testschema.EnvironmentYAML(t, db)
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref,image_name,current_revision_id) VALUES(1,19,'api','api','k8s_yaml','manual','{}','registry.example/team/api',11)`)
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 1
  selector:
    matchLabels: {app: api}
  template:
    metadata:
      labels: {app: api}
    spec:
      containers:
      - name: api
        image: {{.Image}}
        env:
        - name: DB_HOST
          value: {{.DB_HOST}}
`
	exec(`INSERT INTO service_revision(id,service_id,revision_number,definition_format,definition,content_hash) VALUES(11,1,1,'k8s_yaml',?,'promotion')`, manifest)
	// Generic reuse is tested on an approval-gated non-production target. The
	// production-specific admission tests below require a real verification.
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace,binding_status,status,is_production,deploy_approval) VALUES(1,19,'Test','qa','k8s_namespace',2,'api-qa','validated','active',0,0),(2,19,'Stage','stage','k8s_namespace',3,'api-prod','validated','active',0,1)`)
	exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(1,1,11),(2,1,11)`)
	exec(`INSERT INTO variable_set(id,project_id,environment_id,service_id,name,scope,revision) VALUES(1,19,1,1,'default','environment',1),(2,19,2,1,'default','environment',1)`)
	exec(`INSERT INTO variable_item(variable_set_id,variable_key,value_text) VALUES(1,'DB_HOST','test-db'),(2,'DB_HOST','prod-db')`)
	exec(`INSERT INTO environment_resource(environment_id,name,kind,manifest,revision) VALUES(1,'axiops-service-settings-1','AxiOpsServiceSettings','{"resources":{"memory_limit":"1Gi"}}',1),(2,'axiops-service-settings-1','AxiOpsServiceSettings','{"resources":{"memory_limit":"4Gi"}}',1)`)
	digest := "sha256:" + strings.Repeat("a", 64)
	exec(`INSERT INTO build_run(id,service_id,project_id,build_number,status,provider,image_url,image_digest,git_commit,git_branch) VALUES(42,1,19,1,'success','kube_job','registry.example/team/api:qa-1',?,'commit','qa')`, digest)
	ctx := context.Background()
	repo := NewRepository(db)
	wf := &Workflow{ProjectID: 19, TargetEnvironmentID: 2, Name: "Production", Code: "production", DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{
		{Key: "approval", Type: NodeApproval},
		{Key: "build", Type: NodeBuild, DependsOn: []string{"approval"}, Config: mustJSON(map[string]interface{}{"service_id": 1})},
		{Key: "deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}, Config: mustJSON(map[string]interface{}{"service_id": 1, "service_revision_id": 11})},
	}}
	wid, e := repo.Create(ctx, wf)
	if e != nil {
		t.Fatal(e)
	}
	resolver := NewDefaultYAMLResolver(NewSQLYAMLSourceStore(db))
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19, Environment: "*"}}), yamlResolver: resolver}
	req := &RunRequest{EnvironmentID: 2, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "existing_image", SourceBuildID: 42}
	id, code, message := logic.RunWithValidation(ctx, 7, wid, req)
	if code != 0 {
		t.Fatalf("promotion: code=%d %s", code, message)
	}
	run, e := repo.FindRun(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if run.RunNumber != 1 || run.Status != RunPendingApproval || len(run.Tasks) != 2 {
		t.Fatal("wrong promotion run or approval bypass")
	}
	for _, task := range run.Tasks {
		if task.NodeType == NodeBuild {
			t.Fatal("promotion scheduled a build")
		}
	}
	c, e := yamlDeployConfig(run.Tasks[1].NodeSnapshot)
	if e != nil {
		t.Fatal(e)
	}
	exec(`UPDATE variable_item SET value_text='new-prod-db' WHERE variable_set_id=2`)
	exec(`UPDATE environment_resource SET manifest='{"resources":{"memory_limit":"6Gi"}}' WHERE environment_id=2`)
	target, e := resolver.RenderYAML(ctx, *run, c)
	if e != nil || target.ClusterID != 3 || target.Namespace != "api-prod" || !strings.Contains(string(target.Manifest), "memory: 4Gi") || !strings.Contains(string(target.Manifest), "value: prod-db") || !strings.Contains(string(target.Manifest), "registry.example/team/api:qa-1@"+digest) {
		t.Fatalf("wrong frozen deployment: err=%v manifest=%s", e, target.Manifest)
	}
	var provenance map[string]interface{}
	if json.Unmarshal(run.ContextSnapshot, &provenance) != nil || provenance["execution_mode"] != "existing_image" {
		t.Fatal("missing source evidence")
	}
	var builds int
	if e = db.QueryRow(`SELECT COUNT(*) FROM build_run`).Scan(&builds); e != nil || builds != 1 {
		t.Fatal("promotion created another build")
	}
	// Configuration-only release must retain the deployed A image, even after B
	// was built, and use the newly saved target config under the same run queue.
	snapshot, e := NewYAMLDeploymentSnapshot(*run, run.Tasks[1], c, target)
	if e != nil {
		t.Fatal(e)
	}
	store := NewSQLDeploymentSnapshotStore(db)
	if e = store.Prepare(ctx, snapshot); e != nil {
		t.Fatal(e)
	}
	if e = store.MarkApplied(ctx, run.ID, run.Tasks[1].NodeKey, NodeYAMLDeploy); e != nil {
		t.Fatal(e)
	}
	exec(`UPDATE workflow_run SET status='succeeded' WHERE id=?`, run.ID)
	exec(`INSERT INTO build_run(id,service_id,project_id,build_number,status,provider,image_url,image_digest) VALUES(43,1,19,2,'success','kube_job','registry.example/team/api:qa-2',?)`, "sha256:"+strings.Repeat("b", 64))
	current, e := repo.LoadConfigurationDeployment(ctx, 19, 2, 1)
	if e != nil || current == nil {
		t.Fatalf("current deployment: %v", e)
	}
	sourceConfig, e := NewSQLYAMLSourceStore(db).LoadYAMLSource(ctx, 2, 1, 11)
	if e != nil {
		t.Fatal(e)
	}
	configReq := &RunRequest{EnvironmentID: 2, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "config_only", SourceSnapshotID: current.SnapshotID, ExpectedConfigDigest: DeploymentConfigDigest(sourceConfig)}
	configID, code, message := logic.RunWithValidation(ctx, 7, wid, configReq)
	if code != 0 {
		t.Fatalf("config release: %d %s", code, message)
	}
	configRun, e := repo.FindRun(ctx, configID)
	if e != nil || len(configRun.Tasks) != 2 {
		t.Fatal("configuration run created a build or lost approval")
	}
	configNode, e := yamlDeployConfig(configRun.Tasks[1].NodeSnapshot)
	if e != nil {
		t.Fatal(e)
	}
	configTarget, e := resolver.RenderYAML(ctx, *configRun, configNode)
	if e != nil || !strings.Contains(string(configTarget.Manifest), "qa-1@"+digest) || strings.Contains(string(configTarget.Manifest), "qa-2") || !strings.Contains(string(configTarget.Manifest), "memory: 6Gi") || !strings.Contains(string(configTarget.Manifest), "value: new-prod-db") {
		t.Fatalf("configuration release changed artifact or used stale config: %v", e)
	}
	if _, duplicateCode, _ := logic.RunWithValidation(ctx, 7, wid, configReq); duplicateCode == 0 {
		t.Fatal("concurrent configuration release accepted")
	}
	exec(`UPDATE workflow_run SET status='failed' WHERE id=?`, configID)
	configReq.ExpectedConfigDigest = strings.Repeat("0", 64)
	if _, staleCode, _ := logic.RunWithValidation(ctx, 7, wid, configReq); staleCode == 0 {
		t.Fatal("stale configuration preview accepted")
	}
	exec(`UPDATE build_run SET image_digest='' WHERE id=42`)
	if _, code, _ = logic.RunWithValidation(ctx, 7, wid, req); code == 0 {
		t.Fatal("legacy artifact without digest accepted")
	}
	exec(`UPDATE project_environment SET namespace='other-prod' WHERE id=2`)
	if _, e = resolver.RenderYAML(ctx, *run, c); e == nil {
		t.Fatal("queued deployment silently changed target")
	}
	exec(`UPDATE project_environment SET namespace='api-prod',is_production=1 WHERE id=2`)
	exec(`UPDATE build_run SET image_digest=? WHERE id=42`, digest)
	t.Run("verified production admission and recovery", func(t *testing.T) { testMySQLVerifiedProduction(t, db, wid) })
	t.Run("Helm production image selection and frozen environment", func(t *testing.T) { testMySQLHelmProduction(t, db, source) })
}
