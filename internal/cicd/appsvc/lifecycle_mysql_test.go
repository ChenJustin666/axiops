package appsvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/zeromicro/go-zero/core/conf"
)

type lifecycleAccess struct{}

func (lifecycleAccess) Authorize(_ context.Context, user int64, r authz.Request) (bool, error) {
	return user == 1 && r.ProjectID == 19, nil
}
func (lifecycleAccess) Scopes(_ context.Context, user int64, _, _ string) (authz.ScopeSet, error) {
	return authz.ScopeSet{AllowAll: user == 1}, nil
}
func (lifecycleAccess) Policies(context.Context, int64) ([]authz.Policy, error) { return nil, nil }

// Only schema definitions are copied; every write is in a disposable database.
func TestMySQLServiceLifecycle(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("could not load test config")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("MySQL integration test not configured")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid DSN")
	}
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("open test connection failed")
	}
	defer admin.Close()
	schema := fmt.Sprintf("axiops_service_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
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
	for _, table := range []string{"app_service", "service_revision", "service_revision_sequence", "project_environment", "environment_service_override", "workflow", "workflow_node", "workflow_run", "workflow_task_run", "build_run", "deployment_snapshot", "delivery_template", "build_command", "host_deployment_command", "host_deployment_rollout", "helm_execution", "variable_set", "variable_item", "environment_resource", "environment_service_yaml_revision", "build_config", "build_number_sequence", "role_binding"} {
		if _, err = db.Exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`"); err != nil {
			t.Fatal(err)
		}
	}
	exec := func(query string, args ...interface{}) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref) VALUES(1,19,'risk_server','risk-server','k8s_yaml','template','{}'),(2,20,'other','other','k8s_yaml','manual','{}')`)
	ctx := context.Background()
	logic := NewLogic(db, lifecycleAccess{})
	assertCode := func(got, want int) {
		t.Helper()
		if got != want {
			t.Fatalf("code=%d want=%d", got, want)
		}
	}
	t.Run("new manual Helm service is rejected before persistence", func(t *testing.T) {
		exec("CREATE TABLE project LIKE " + source + ".project")
		exec(`INSERT INTO project(id,name,code,delivery_type,delivery_status) VALUES(19,'Helm','helm','helm','active')`)
		id, result := logic.Create(ctx, &CreateAppServiceRequest{
			ProjectID: 19, Name: "manual-chart", Code: "manual-chart", SourceType: "manual",
		}, 1)
		if id != 0 || result != errcode.CodeParamInvalid {
			t.Fatalf("manual Helm creation returned id=%d code=%d", id, result)
		}
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM app_service WHERE code='manual-chart'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rejected creation wrote a service: count=%d err=%v", count, err)
		}
	})
	_, code := logic.DeleteChecked(ctx, 2, 1)
	assertCode(code, errcode.CodeNoPermission)
	_, code = logic.DeleteChecked(ctx, 1, 2)
	assertCode(code, errcode.CodeNoPermission)
	assertCode(logic.Rename(ctx, 2, 1, "unauthorized"), errcode.CodeNoPermission)
	assertCode(logic.Rename(ctx, 1, 1, "风控网关"), 0)
	s, _ := logic.repo.FindByID(ctx, 1)
	if s.Name != "风控网关" || s.Code != "risk-server" || s.ImageName != "risk_server" {
		t.Fatalf("rename changed stable identity: %+v", s)
	}
	revision, code := logic.CreateRevision(ctx, 1, 1, &CreateServiceRevisionRequest{Definition: "apiVersion: v1\nkind: Service\nmetadata:\n  name: risk-server\n"})
	assertCode(code, 0)
	exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace) VALUES(1,19,'QA','qa','k8s',1,'qa')`)
	exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(1,1,?)`, revision.ID)
	check, code := logic.CheckDeletion(ctx, 1, 1)
	assertCode(code, 0)
	if check.CanDelete || len(check.Blockers) != 1 || check.Blockers[0].Kind != "environment" {
		t.Fatalf("environment check: %+v", check)
	}
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, errcode.CodeServiceInUse)
	assertCode(logic.DetachEnvironment(ctx, 1, 1, 1, 99), errcode.CodeParamInvalid)
	assertCode(logic.DetachEnvironment(ctx, 1, 1, 1, 1), 0)
	exec(`INSERT INTO workflow(id,project_id,name,code,delivery_type) VALUES(1,19,'delivery','delivery','k8s_yaml')`)
	exec(`INSERT INTO workflow_node(workflow_id,node_key,name,node_type,config_json,depends_on,ordinal) VALUES(1,'build','build','build','{"service_id":1}','[]',0)`)
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, errcode.CodeServiceInUse)
	exec(`DELETE FROM workflow_node WHERE workflow_id=1`)
	exec(`INSERT INTO workflow_run(id,workflow_id,project_id,environment_id,definition_version,context_snapshot,status) VALUES(1,1,19,1,1,'{}','pending_approval')`)
	exec(`INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status) VALUES(1,'build','build','[]','{"config":{"service_id":1}}','pending')`)
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, errcode.CodeServiceInUse)
	exec(`UPDATE workflow_run SET status='failed'`)
	check, code = logic.CheckDeletion(ctx, 1, 1)
	assertCode(code, 0)
	if !check.CanDelete {
		t.Fatal("unstarted task in a terminal run blocked deletion")
	}
	exec(`UPDATE workflow_task_run SET status='running'`)
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, errcode.CodeServiceInUse) // incomplete task still protects service
	exec(`UPDATE workflow_task_run SET status='cancelled'`)

	// Concurrent submit wins the lock: delete must recheck after it commits.
	writer, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = servicestate.LockActive(ctx, writer, 19, 1); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { _, c := logic.DeleteChecked(ctx, 1, 1); done <- c }()
	select {
	case <-done:
		writer.Rollback()
		t.Fatal("delete bypassed service transaction lock")
	case <-time.After(50 * time.Millisecond):
	}
	_, err = writer.ExecContext(ctx, `INSERT INTO build_run(service_id,project_id,build_number,config_snapshot,status,provider) VALUES(1,19,1,'{}','queued','kube_job')`)
	if err != nil {
		writer.Rollback()
		t.Fatal(err)
	}
	if err = writer.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case code = <-done:
		assertCode(code, errcode.CodeServiceInUse)
	case <-time.After(10 * time.Second):
		t.Fatal("delete did not release")
	}
	exec(`UPDATE build_run SET status='success'`)
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, 0)
	_, code = logic.DeleteChecked(ctx, 1, 1)
	assertCode(code, 0) // idempotent
	visible, total, code := logic.List(ctx, 1, 19, "", "", 1, 20)
	assertCode(code, 0)
	if total != 0 || len(visible) != 0 {
		t.Fatal("deleted service visible in normal list")
	}
	trash, total, code := logic.List(ctx, 1, 19, "", "deleted", 1, 20)
	assertCode(code, 0)
	if total != 1 || trash[0].ID != 1 {
		t.Fatal("deleted service missing from trash")
	}
	assertCode(logic.Update(ctx, 1, 1, &UpdateAppServiceRequest{Status: "active"}), errcode.CodeParamInvalid)
	assertCode(logic.Rename(ctx, 1, 1, "deleted rename"), errcode.CodeParamInvalid)
	writer, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if servicestate.LockActive(ctx, writer, 19, 1) == nil {
		t.Error("deleted service accepted new references")
	}
	writer.Rollback()
	assertCode(logic.Restore(ctx, 2, 1), errcode.CodeNoPermission)
	assertCode(logic.Restore(ctx, 1, 1), 0)
	s, _ = logic.repo.FindByID(ctx, 1)
	if s.Status != "active" || s.CurrentRevisionID != revision.ID || s.ImageName != "risk_server" || s.Name != "风控网关" {
		t.Fatalf("restore lost identity: %+v", s)
	}
	revisions, code := logic.ListRevisions(ctx, 1, 1)
	assertCode(code, 0)
	if len(revisions) != 1 {
		t.Fatal("revision history lost")
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM build_run WHERE service_id=1`).Scan(&count); err != nil || count != 1 {
		t.Fatal("build history lost")
	}

	// Replacing a template appends to the same service and leaves environment
	// bindings and build history untouched. A stale chooser cannot win silently.
	exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(1,1,?)`, revision.ID)
	expected := revision.ID
	request := &CreateServiceRevisionRequest{SourceCommit: "template:7@1", Definition: "apiVersion: v1\nkind: Service\nmetadata:\n  name: changed-template\n", ExpectedRevisionID: &expected}
	_, code = logic.CreateRevision(ctx, 1, 1, &CreateServiceRevisionRequest{Definition: "apiVersion: [", ExpectedRevisionID: &expected})
	assertCode(code, errcode.CodeServiceDefinitionInvalid)
	unchanged, _ := logic.repo.FindByID(ctx, 1)
	if unchanged.CurrentRevisionID != expected {
		t.Fatal("invalid YAML changed current service revision")
	}
	_, code = logic.CreateRevision(ctx, 2, 1, request)
	assertCode(code, errcode.CodeNoPermission)
	next, code := logic.CreateRevision(ctx, 1, 1, request)
	assertCode(code, 0)
	_, code = logic.CreateRevision(ctx, 1, 1, request)
	assertCode(code, errcode.CodeServiceRevisionChanged)
	var bound int64
	if err = db.QueryRow(`SELECT service_revision_id FROM environment_service_override WHERE service_id=1`).Scan(&bound); err != nil || bound != revision.ID {
		t.Fatal("template replacement changed environment binding")
	}
	s, _ = logic.repo.FindByID(ctx, 1)
	if s.CurrentRevisionID != next.ID || s.Name != "风控网关" || s.ImageName != "risk_server" {
		t.Fatal("template replacement changed service identity")
	}
	revisions, code = logic.ListRevisions(ctx, 1, 1)
	assertCode(code, 0)
	if len(revisions) != 2 || revisions[0].SourceCommit != "template:7@1" || revisions[1].ID != revision.ID {
		t.Fatal("template replacement lost history or wrote stale revision")
	}
	// Two services imported from the same template are independent snapshots.
	exec(`INSERT INTO delivery_template(id,kind,name,files,variables,created_by) VALUES(7,'k8s_yaml','shared-origin','[]','[]',1)`)
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref) VALUES(4,19,'service-b','service-b','k8s_yaml','template','{"template_id":7}')`)
	otherRevision, code := logic.CreateRevision(ctx, 1, 4, &CreateServiceRevisionRequest{SourceCommit: "template:7@1", Definition: request.Definition})
	assertCode(code, 0)
	exec(`INSERT INTO environment_service_override(project_environment_id,service_id,service_revision_id) VALUES(1,4,?)`, otherRevision.ID)
	beforeB, _ := logic.repo.FindByID(ctx, 4)
	expected = next.ID
	_, code = logic.CreateRevision(ctx, 1, 1, &CreateServiceRevisionRequest{SourceCommit: "manual-edit", Definition: "# AxiOps-Variables: [{\"key\":\"port\",\"value\":\"9100\"}]\napiVersion: v1\nkind: Service\nmetadata:\n  name: a-only\n", ExpectedRevisionID: &expected})
	assertCode(code, 0)
	afterB, _ := logic.repo.FindByID(ctx, 4)
	otherRevisions, code := logic.ListRevisions(ctx, 1, 4)
	assertCode(code, 0)
	if beforeB.CurrentRevisionID != afterB.CurrentRevisionID || len(otherRevisions) != 1 || otherRevisions[0].Definition != otherRevision.Definition {
		t.Fatal("saving service A changed service B's configuration")
	}
	var templateFiles, templateVariables string
	var templateRevision int64
	if err = db.QueryRow(`SELECT files,variables,revision FROM delivery_template WHERE id=7`).Scan(&templateFiles, &templateVariables, &templateRevision); err != nil || templateFiles != "[]" || templateVariables != "[]" || templateRevision != 1 {
		t.Fatal("saving service A changed the template library")
	}
	for sid, want := range map[int64]int64{1: revision.ID, 4: otherRevision.ID} {
		if err = db.QueryRow(`SELECT service_revision_id FROM environment_service_override WHERE service_id=?`, sid).Scan(&bound); err != nil || bound != want {
			t.Fatal("saving service A changed an existing environment configuration")
		}
	}
	exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref) VALUES(3,19,'new','new','k8s_yaml','manual','{}')`)
	zero := int64(0)
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, c := logic.CreateRevision(ctx, 1, 3, &CreateServiceRevisionRequest{Definition: "apiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\n", ExpectedRevisionID: &zero})
			results <- c
		}()
	}
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		select {
		case c := <-results:
			if c == 0 {
				wins++
			} else if c == errcode.CodeServiceRevisionChanged {
				conflicts++
			} else {
				t.Fatalf("unexpected concurrent result %d", c)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("concurrent template save did not finish")
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
	for i, kind := range []string{"k8s_yaml", "helm"} {
		t.Run("purge "+kind, func(t *testing.T) {
			id := int64(10 + i*10)
			name := fmt.Sprintf("purge-%d", id)
			exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref) VALUES(?,19,?,?,?,'template','{}')`, id, name, name, kind)
			assertCode(logic.Purge(ctx, 1, id, name), errcode.CodeParamInvalid)
			exec(`INSERT INTO service_revision(service_id,revision_number,definition,definition_format,content_hash,created_by) VALUES(?,1,'saved configuration','k8s_yaml','hash',1)`, id)
			exec(`INSERT INTO variable_set(project_id,service_id,name,scope) VALUES(19,?,'default','service')`, id)
			exec(`INSERT INTO variable_item(variable_set_id,variable_key,value_text) SELECT id,'key','value' FROM variable_set WHERE service_id=?`, id)
			exec(`INSERT INTO build_config(service_id) VALUES(?)`, id)
			exec(`INSERT INTO environment_resource(environment_id,name,kind,manifest) VALUES(1,?,'AxiOpsHelmValues','{}'),(1,?,'AxiOpsServiceSettings','{}')`, fmt.Sprintf("axiops-s%d-helm-values", id), fmt.Sprintf("axiops-service-settings-%d", id))
			exec(`INSERT IGNORE INTO environment_resource(environment_id,name,kind,manifest) VALUES(1,'shared-config','ConfigMap','shared')`)
			_, result := logic.DeleteChecked(ctx, 1, id)
			assertCode(result, 0)
			assertCode(logic.Purge(ctx, 2, id, name), errcode.CodeNoPermission)
			assertCode(logic.Purge(ctx, 1, id, "wrong"), errcode.CodeParamInvalid)
			// A terminal build still owns its pending cleanup; never erase it early.
			exec(`INSERT INTO build_run(service_id,project_id,build_number,config_snapshot,status,provider,cleanup_pending) VALUES(?,19,1,'{}','success','kube_job',1)`, id)
			assertCode(logic.Purge(ctx, 1, id, name), errcode.CodeServiceInUse)
			exec(`UPDATE build_run SET cleanup_pending=0 WHERE service_id=?`, id)
			exec(`INSERT INTO build_command(build_run_id,command_type,provider,payload,status,idempotency_key) SELECT id,'cancel','kube_job','{}','pending',? FROM build_run WHERE service_id=?`, name, id)
			assertCode(logic.Purge(ctx, 1, id, name), errcode.CodeServiceInUse)
			exec(`UPDATE build_command SET status='succeeded' WHERE idempotency_key=?`, name)
			if kind == "helm" {
				exec(`INSERT INTO workflow_task_run(workflow_run_id,node_key,node_type,depends_on,node_snapshot,status) VALUES(1,'purge-helm','helm_deploy','[]',?,'cancelled')`, fmt.Sprintf(`{"config":{"service_id":%d}}`, id))
				exec(`INSERT INTO helm_execution(idempotency_key,request_hash,workflow_run_id,node_key,operation,cluster_id,namespace,release_name,status) VALUES('purge-helm','hash',1,'purge-helm','upgrade',1,'qa','purge-helm','running')`)
				assertCode(logic.Purge(ctx, 1, id, name), errcode.CodeServiceInUse)
				exec(`UPDATE helm_execution SET status='failed' WHERE idempotency_key='purge-helm'`)
			}
			c, result := logic.CheckDeletion(ctx, 1, id)
			assertCode(result, 0)
			if !c.CanPurge || c.CanDelete {
				t.Fatal("trash capabilities incorrect")
			}
			// Every configuration delete must roll back if final identity deletion fails.
			exec(`CREATE TRIGGER fail_service_purge BEFORE DELETE ON app_service FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='test failure'`)
			assertCode(logic.Purge(ctx, 1, id, name), errcode.CodeDBError)
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM variable_set WHERE service_id=?`, id).Scan(&n); err != nil || n != 1 {
				t.Fatal("failed purge lost configuration")
			}
			exec(`DROP TRIGGER fail_service_purge`)
			assertCode(logic.Purge(ctx, 1, id, name), 0)
			for _, table := range []string{"app_service", "service_revision", "variable_set", "build_config"} {
				column := "service_id"
				if table == "app_service" {
					column = "id"
				}
				if err := db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+column+"=?", id).Scan(&n); err != nil || n != 0 {
					t.Fatalf("purge left %s: %d %v", table, n, err)
				}
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM environment_resource WHERE name IN (?,?)`, fmt.Sprintf("axiops-s%d-helm-values", id), fmt.Sprintf("axiops-service-settings-%d", id)).Scan(&n); err != nil || n != 0 {
				t.Fatal("service settings remain")
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM environment_resource WHERE name='shared-config'`).Scan(&n); err != nil || n != 1 {
				t.Fatal("shared resource deleted")
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM build_run WHERE service_id=?`, id).Scan(&n); err != nil || n != 1 {
				t.Fatal("historical build evidence deleted")
			}
			assertCode(logic.Restore(ctx, 1, id), errcode.CodeServiceNotFound)
			// Reuse the exact code with a new identity; old builds remain on the old ID.
			exec(`INSERT INTO app_service(project_id,name,code,delivery_type,source_type,source_ref) VALUES(19,?,?,?,'template','{}')`, name, name, kind)
			created, err := logic.repo.FindByProjectAndCode(ctx, 19, name)
			if err != nil || created == nil || created.ID == id {
				t.Fatal("code cannot be recreated independently")
			}
			assertCode(logic.Purge(ctx, 1, created.ID, name), errcode.CodeParamInvalid)
		})
	}
	t.Run("Git Chart source binding commits atomically with revision", func(t *testing.T) {
		exec(`INSERT INTO app_service(id,project_id,name,code,delivery_type,source_type,source_ref) VALUES(40,19,'git-chart','git-chart','helm','git','{"retain":true}')`)
		zero := int64(0)
		rev := &ServiceRevision{ServiceID: 40, SourceCommit: "frozen-commit", Definition: "native-chart-snapshot", DefinitionFormat: "helm_chart", ContentHash: "hash", CreatedBy: 1, ExpectedRevisionID: &zero, RepositoryBinding: &HelmRepositoryBinding{CodeSourceID: 9, RepoPath: "team/chart", Ref: "main", ChartPath: "deploy/chart", CommitID: "frozen-commit"}}
		id, _, err := logic.repo.CreateRevision(ctx, rev)
		if err != nil {
			t.Fatal(err)
		}
		s, err := logic.repo.FindByID(ctx, 40)
		if err != nil {
			t.Fatal(err)
		}
		var source struct {
			Retain     bool                  `json:"retain"`
			Repository HelmRepositoryBinding `json:"helm_repository"`
		}
		if json.Unmarshal(s.SourceRef, &source) != nil || !source.Retain || source.Repository.RevisionID != id || s.CurrentRevisionID != id {
			t.Fatalf("source not saved atomically: %s", s.SourceRef)
		}
		rev.RepositoryBinding.CommitID = "new-upstream"
		if _, _, err = logic.repo.CreateRevision(ctx, rev); !errors.Is(err, errRevisionChanged) {
			t.Fatalf("stale import=%v", err)
		}
		s, _ = logic.repo.FindByID(ctx, 40)
		if json.Unmarshal(s.SourceRef, &source) != nil || source.Repository.CommitID != "frozen-commit" {
			t.Fatal("stale import rewrote source")
		}
		_, _, err = logic.repo.CreateRevision(ctx, &ServiceRevision{ServiceID: 40, SourceCommit: "manual-chart-edit", Definition: "custom-chart-snapshot", DefinitionFormat: "helm_chart", ContentHash: "other", CreatedBy: 1, ExpectedRevisionID: &id})
		if err != nil {
			t.Fatal(err)
		}
		s, _ = logic.repo.FindByID(ctx, 40)
		if json.Unmarshal(s.SourceRef, &source) != nil || source.Repository.RevisionID != id || s.CurrentRevisionID == id {
			t.Fatal("platform edit lost Git origin")
		}
	})
}
