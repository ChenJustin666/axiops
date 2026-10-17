package project

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
	"github.com/zeromicro/go-zero/core/conf"
)

type deletionPolicies []authz.Policy

func (p deletionPolicies) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return p, nil
}

func TestDeleteRequiresConfirmationBeforeDatabaseAccess(t *testing.T) {
	l := NewLogic(nil, nil)
	for _, req := range []*DeleteProjectRequest{nil, {ConfirmName: ""}} {
		if _, code := l.Delete(context.Background(), 1, 1, req); code != errcode.CodeParamInvalid {
			t.Fatalf("code=%d", code)
		}
	}
	for _, body := range []string{"", `{}`, `{"confirm_name":""}`, `invalid`} {
		r := httptest.NewRequest("DELETE", "/api/v1/projects/1", strings.NewReader(body))
		r.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		NewHandler(nil).Delete(w, r)
		var result struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Code != errcode.CodeParamInvalid {
			t.Fatalf("unconfirmed delete code=%d", result.Code)
		}
	}
}

func TestDeleteAuthorizationBeforeDatabaseAccess(t *testing.T) {
	l := NewLogic(nil, authz.New(deletionPolicies{}))
	if _, code := l.Delete(context.Background(), 1, 1, &DeleteProjectRequest{ConfirmName: "sample"}); code != errcode.CodeNoPermission {
		t.Fatalf("code=%d", code)
	}
	if _, code := l.CheckDeletion(context.Background(), 1, 1); code != errcode.CodeNoPermission {
		t.Fatalf("check code=%d", code)
	}
}

func TestMySQLProjectDeletionProtectionAndIsolation(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("could not load test configuration")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("MySQL test configuration is not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("axiops_project_delete_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName = schema
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(query string) {
		t.Helper()
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{
		`CREATE TABLE project(id BIGINT PRIMARY KEY,name VARCHAR(128),status VARCHAR(16) NOT NULL DEFAULT 'active',updated_at DATETIME DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB`,
		`CREATE TABLE app_service(id BIGINT PRIMARY KEY,project_id BIGINT,KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE project_environment(id BIGINT PRIMARY KEY,project_id BIGINT,KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE workflow(id BIGINT PRIMARY KEY,project_id BIGINT,KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE variable_set(id BIGINT PRIMARY KEY,project_id BIGINT,environment_id BIGINT,service_id BIGINT,KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE variable_item(id BIGINT PRIMARY KEY,variable_set_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE environment_resource(id BIGINT PRIMARY KEY,environment_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE environment_service_override(id BIGINT PRIMARY KEY,project_environment_id BIGINT,service_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE environment_service_yaml_revision(id BIGINT PRIMARY KEY,environment_id BIGINT,service_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE workflow_node(id BIGINT PRIMARY KEY,workflow_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE service_revision(id BIGINT PRIMARY KEY,service_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE build_config(id BIGINT PRIMARY KEY,service_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE role_binding(id BIGINT PRIMARY KEY,scope_type VARCHAR(32),scope_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE project_member(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE build_run(id BIGINT PRIMARY KEY,project_id BIGINT,service_id BIGINT,status VARCHAR(32),KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE workflow_run(id BIGINT PRIMARY KEY,project_id BIGINT,status VARCHAR(32),KEY(project_id)) ENGINE=InnoDB`,
		`CREATE TABLE future_owned_data(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE audit_log(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE build_log(id BIGINT PRIMARY KEY,build_run_id BIGINT,FOREIGN KEY(build_run_id) REFERENCES build_run(id)) ENGINE=InnoDB`,
		`CREATE TABLE build_command(id BIGINT PRIMARY KEY,build_run_id BIGINT,status VARCHAR(32)) ENGINE=InnoDB`,
		`CREATE TABLE workflow_task_run(id BIGINT PRIMARY KEY,workflow_run_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE helm_execution(id BIGINT PRIMARY KEY,workflow_run_id BIGINT,status VARCHAR(32)) ENGINE=InnoDB`,
		`CREATE TABLE notification_subscription(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE notification_outbox(id BIGINT PRIMARY KEY,subscription_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE code_source(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE webhook_delivery(id BIGINT PRIMARY KEY,source_id BIGINT,build_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE credential(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE image_registry(id BIGINT PRIMARY KEY,project_id BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE build_number_sequence(service_id BIGINT PRIMARY KEY,next_number BIGINT) ENGINE=InnoDB`,
		`CREATE TABLE service_revision_sequence(service_id BIGINT PRIMARY KEY,next_number BIGINT) ENGINE=InnoDB`,
	} {
		exec(query)
	}
	exec(`INSERT INTO project(id,name) VALUES(1,'删除测试'),(2,'保留项目')`)
	for _, table := range []string{"app_service", "project_environment", "workflow", "project_member"} {
		exec("INSERT INTO " + table + " VALUES(1,1),(2,2)")
	}
	exec(`INSERT INTO variable_set VALUES(1,1,1,1),(2,2,2,2),(3,0,0,0)`)
	exec(`INSERT INTO variable_item VALUES(1,1),(2,2),(3,3)`)
	for _, table := range []string{"environment_resource", "workflow_node", "service_revision", "build_config"} {
		exec("INSERT INTO " + table + " VALUES(1,1),(2,2)")
	}
	for _, table := range []string{"environment_service_override", "environment_service_yaml_revision"} {
		exec("INSERT INTO " + table + " VALUES(1,1,1),(2,2,2)")
	}
	exec(`INSERT INTO role_binding VALUES(1,'project',1),(2,'service',1),(3,'environment',1),(4,'workflow',1),(5,'project',2),(6,'platform',0)`)
	exec(`INSERT INTO audit_log VALUES(1,1)`)
	l := NewLogic(db, authz.New(deletionPolicies{{Resource: "project", Action: "delete", Effect: "allow", ScopeType: "project", ScopeID: 1}}))
	ctx := context.Background()
	req := &DeleteProjectRequest{ConfirmName: "删除测试"}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("exact name and scope", func(t *testing.T) {
		for _, name := range []string{"wrong", " 删除测试", "删除测试 "} {
			if _, code := l.Delete(ctx, 1, 1, &DeleteProjectRequest{ConfirmName: name}); code != errcode.CodeParamInvalid {
				t.Fatalf("code=%d", code)
			}
		}
		if _, code := l.Delete(ctx, 1, 2, &DeleteProjectRequest{ConfirmName: "保留项目"}); code != errcode.CodeNoPermission {
			t.Fatalf("code=%d", code)
		}
		if count("project") != 2 {
			t.Fatal("unconfirmed delete changed data")
		}
	})
	t.Run("preview followed by a new run is rejected", func(t *testing.T) {
		check, code := l.CheckDeletion(ctx, 1, 1)
		if code != 0 || !check.CanDelete {
			t.Fatalf("check=%+v code=%d", check, code)
		}
		for _, status := range []string{"queued", "dispatching", "running", "cancelling", "unknown"} {
			exec("INSERT INTO build_run VALUES(1,1,1,'" + status + "')")
			check, code = l.Delete(ctx, 1, 1, req)
			if code != errcode.CodeProjectInUse || check.CanDelete {
				t.Fatalf("status=%s check=%+v code=%d", status, check, code)
			}
			if count("variable_item") != 3 || count("project") != 2 {
				t.Fatal("blocked delete changed data")
			}
			exec("DELETE FROM build_run")
		}
	})
	t.Run("completed history and unknown owned data are retained", func(t *testing.T) {
		exec("INSERT INTO future_owned_data VALUES(1,1)")
		for _, status := range []string{"success", "failed", "timeout", "cancelled"} {
			exec("INSERT INTO build_run VALUES(1,1,1,'" + status + "')")
			check, code := l.CheckDeletion(ctx, 1, 1)
			if code != 0 || !check.CanDelete {
				t.Fatalf("status=%s check=%+v code=%d", status, check, code)
			}
			exec("DELETE FROM build_run")
		}
		for _, status := range []string{"pending", "pending_approval", "running", "cancelling"} {
			exec("INSERT INTO workflow_run VALUES(1,1,'" + status + "')")
			if _, code := l.Delete(ctx, 1, 1, req); code != errcode.CodeProjectInUse {
				t.Fatalf("status=%s code=%d", status, code)
			}
			exec("DELETE FROM workflow_run")
		}
		exec("INSERT INTO build_run VALUES(1,1,1,'failed')")
		exec("INSERT INTO workflow_run VALUES(1,1,'succeeded'),(2,1,'failed'),(3,1,'cancelled')")
		exec("INSERT INTO build_run VALUES(2,2,2,'success')")
		exec("INSERT INTO workflow_run VALUES(4,2,'succeeded')")
		for _, table := range []string{"build_log", "workflow_task_run", "notification_outbox"} {
			exec("INSERT INTO " + table + " VALUES(1,1),(2,2)")
		}
		exec("INSERT INTO workflow_task_run VALUES(3,4)")
		exec("INSERT INTO build_command VALUES(1,1,'dead'),(2,2,'succeeded')")
		exec("INSERT INTO helm_execution VALUES(1,1,'succeeded'),(2,4,'succeeded')")
		for _, table := range []string{"notification_subscription", "code_source"} {
			exec("INSERT INTO " + table + " VALUES(1,1),(2,2)")
		}
		for _, table := range []string{"credential", "image_registry"} {
			exec("INSERT INTO " + table + " VALUES(1,1),(2,2),(3,0)")
		}
		exec("INSERT INTO webhook_delivery VALUES(1,1,1),(2,2,2)")
		for _, table := range []string{"build_number_sequence", "service_revision_sequence"} {
			exec("INSERT INTO " + table + " VALUES(1,42),(2,13)")
		}
	})
	t.Run("failed status update rolls back", func(t *testing.T) {
		exec(`CREATE TRIGGER fail_delete BEFORE UPDATE ON project FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='test failure'`)
		if _, code := l.Delete(ctx, 1, 1, req); code != errcode.CodeDBError {
			t.Fatalf("code=%d", code)
		}
		if count("variable_item") != 3 || count("role_binding") != 6 || count("project") != 2 {
			t.Fatal("failed cleanup did not roll back")
		}
		exec("DROP TRIGGER fail_delete")
	})
	t.Run("submission holding project lock prevents deletion", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if err := projectstate.LockWritable(ctx, tx, 1); err != nil {
			t.Fatal(err)
		}
		result := make(chan int, 1)
		deleteCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go func() {
			_, code := l.Delete(deleteCtx, 1, 1, req)
			result <- code
		}()
		if _, err := tx.ExecContext(ctx, "INSERT INTO build_run VALUES(99,1,1,'queued')"); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if code := <-result; code != errcode.CodeProjectInUse {
			t.Fatalf("concurrent submission was not protected: code=%d", code)
		}
		exec("DELETE FROM build_run WHERE id=99")
	})
	t.Run("soft deletion preserves data and rejects subsequent submissions", func(t *testing.T) {
		if _, code := l.Delete(ctx, 1, 1, req); code != 0 {
			t.Fatalf("code=%d", code)
		}
		for _, table := range []string{"project", "app_service", "project_environment", "workflow", "project_member", "environment_resource", "workflow_node", "service_revision", "build_config", "environment_service_override", "environment_service_yaml_revision"} {
			if count(table) != 2 {
				t.Fatalf("%s not isolated", table)
			}
		}
		if count("variable_set") != 3 || count("variable_item") != 3 || count("role_binding") != 6 || count("build_run") != 2 || count("workflow_run") != 4 || count("future_owned_data") != 1 || count("audit_log") != 1 {
			t.Fatal("shared data or other project's data changed")
		}
		if _, code := l.Delete(ctx, 1, 1, req); code != errcode.CodeProjectDeleted {
			t.Fatalf("second delete code=%d", code)
		}
		var status string
		if err := db.QueryRow("SELECT status FROM project WHERE id=1").Scan(&status); err != nil || status != "deleted" {
			t.Fatalf("status=%s err=%v", status, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if err := projectstate.LockWritable(ctx, tx, 1); err != projectstate.ErrDeleted {
			t.Fatalf("deleted project accepted: %v", err)
		}
		if err := projectstate.LockWritable(ctx, tx, 2); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("archived purge blocks ongoing executor work", func(t *testing.T) {
		permanent := &DeleteProjectRequest{ConfirmName: "删除测试", Permanent: true}
		exec("UPDATE build_command SET status='processing' WHERE id=1")
		if _, code := l.Delete(ctx, 1, 1, permanent); code != errcode.CodeProjectInUse {
			t.Fatalf("processing command was purged: %d", code)
		}
		exec("UPDATE build_command SET status='dead' WHERE id=1")
		exec("UPDATE helm_execution SET status='running' WHERE id=1")
		if _, code := l.Delete(ctx, 1, 1, permanent); code != errcode.CodeProjectInUse {
			t.Fatalf("running helm execution was purged: %d", code)
		}
		exec("UPDATE helm_execution SET status='succeeded' WHERE id=1")
	})
	t.Run("failed permanent purge rolls all child deletes back", func(t *testing.T) {
		exec(`CREATE TRIGGER fail_purge BEFORE DELETE ON project FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='test failure'`)
		if _, code := l.Delete(ctx, 1, 1, &DeleteProjectRequest{ConfirmName: "删除测试", Permanent: true}); code != errcode.CodeDBError {
			t.Fatalf("code=%d", code)
		}
		if count("project") != 2 || count("build_log") != 2 || count("workflow_task_run") != 3 || count("future_owned_data") != 1 || count("role_binding") != 6 {
			t.Fatal("permanent purge did not roll back")
		}
		exec("DROP TRIGGER fail_purge")
	})
	t.Run("permanent purge removes descendants and isolates shared data", func(t *testing.T) {
		check, code := l.CheckDeletion(ctx, 1, 1)
		if code != 0 || !check.CanDelete {
			t.Fatalf("archived preview=%+v code=%d", check, code)
		}
		for _, key := range []string{"build_log", "workflow_task_run", "notification_outbox", "helm_execution", "webhook_delivery"} {
			found := false
			for _, item := range check.Items {
				if item.Kind == key {
					found = true
				}
			}
			if !found {
				t.Fatalf("descendant missing from preview: %s", key)
			}
		}
		if _, code := l.Delete(ctx, 1, 1, &DeleteProjectRequest{ConfirmName: "删除测试", Permanent: true}); code != 0 {
			t.Fatalf("code=%d", code)
		}
		for _, table := range []string{"project", "app_service", "project_environment", "workflow", "project_member", "environment_resource", "workflow_node", "service_revision", "build_config", "environment_service_override", "environment_service_yaml_revision", "build_log", "build_command", "build_run", "workflow_run", "workflow_task_run", "notification_outbox", "notification_subscription", "helm_execution", "webhook_delivery", "code_source", "build_number_sequence", "service_revision_sequence"} {
			if count(table) != 1 {
				t.Fatalf("%s not completely cleaned or other project affected", table)
			}
		}
		for _, table := range []string{"variable_set", "variable_item", "role_binding", "credential", "image_registry"} {
			if count(table) != 2 {
				t.Fatalf("%s shared isolation failed", table)
			}
		}
		if count("future_owned_data") != 0 || count("audit_log") != 1 {
			t.Fatal("optional ownership or audit retention failed")
		}
		if _, code := l.Delete(ctx, 1, 1, &DeleteProjectRequest{ConfirmName: "删除测试", Permanent: true}); code != errcode.CodeProjectNotFound {
			t.Fatalf("repeat purge=%d", code)
		}
	})
}
