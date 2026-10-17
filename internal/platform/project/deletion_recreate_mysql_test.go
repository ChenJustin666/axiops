package project

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/migrations"
	"github.com/zeromicro/go-zero/core/conf"
)

// Exercise the real migration, SQL permissions and API logic in a disposable DB.
func TestMySQLProjectDeleteRecreateWithManagedSchema(t *testing.T) {
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
	schema := fmt.Sprintf("axiops_project_recreate_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName, cfg.MultiStatements, cfg.ParseTime = schema, true, true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	exec := func(query string, args ...interface{}) int64 {
		t.Helper()
		r, err := db.Exec(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := r.LastInsertId()
		return id
	}
	for _, name := range []string{"001_platform_base.sql", "002_asset_management.sql", "003_cicd_core.sql"} {
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		exec(string(body))
	}
	// An existing archived project must be released by migration, not erased.
	archivedID := exec("INSERT INTO project(name,code,status) VALUES('反复测试','repeat-test','deleted')")
	ctx := context.Background()
	if _, err := migrations.Ensure(ctx, db, true, 10); err != nil {
		t.Fatal(err)
	}
	if result, err := migrations.Ensure(ctx, db, true, 10); err != nil || len(result.Applied) != 0 {
		t.Fatalf("migration repeat: %+v %v", result, err)
	}
	l := NewLogic(db, authz.NewSQLAuthorizer(db))
	create := func() int64 {
		t.Helper()
		id, code := l.Create(ctx, &CreateProjectRequest{Name: "反复测试", Code: "repeat-test", DeliveryType: DeliveryTypeK8sYAML}, 1)
		if code != 0 {
			t.Fatalf("create code=%d", code)
		}
		return id
	}
	for round := 0; round < 3; round++ {
		id := create()
		if id == archivedID {
			t.Fatal("creation resurrected archived project")
		}
		if _, code := l.Create(ctx, &CreateProjectRequest{Name: "反复测试", Code: "repeat-test", DeliveryType: DeliveryTypeK8sYAML}, 1); code != errcode.CodeProjectNameExists {
			t.Fatalf("live duplicate allowed: %d", code)
		}
		svcID := exec("INSERT INTO app_service(project_id,name,code) VALUES(?,'test-service','test-service')", id)
		exec("INSERT INTO build_number_sequence(service_id,current_number) VALUES(?,1)", svcID)
		runID := exec("INSERT INTO build_run(project_id,service_id,build_number,provider,status) VALUES(?,?,1,'kubernetes','failed')", id, svcID)
		exec("INSERT INTO production_release_policy(environment_id,project_id,source_environment_id,updated_by) VALUES(?,?,?,1)", id, id, id+1000)
		verificationID := exec("INSERT INTO deployment_verification(project_id,environment_id,service_id,snapshot_id,build_run_id,image_ref,manifest_digest,verified_by) VALUES(?,?,?,?,?,'registry/app@sha256:test',REPEAT('a',64),1)", id, id+1000, svcID, id, runID)
		exec("INSERT INTO deployment_verification_event(project_id,verification_id,action,actor_id) VALUES(?,?,'pass',1)", id, verificationID)
		exec("INSERT INTO build_log(build_run_id,log_text) VALUES(?,'saved build output')", runID)
		exec("UPDATE build_run SET cleanup_pending=1 WHERE id=?", runID)
		if _, code := l.Delete(ctx, 1, id, &DeleteProjectRequest{ConfirmName: "反复测试", Permanent: true}); code != errcode.CodeProjectInUse {
			t.Fatalf("pending cleanup lost: %d", code)
		}
		exec("UPDATE build_run SET cleanup_pending=0 WHERE id=?", runID)
		if round == 0 {
			if _, code := l.Delete(ctx, 1, id, &DeleteProjectRequest{ConfirmName: "反复测试"}); code != 0 {
				t.Fatalf("archive=%d", code)
			}
			// Archived duplicates coexist while a new live project holds the code.
			newID := create()
			if newID == id {
				t.Fatal("new project reused old identity")
			}
			if _, code := l.Delete(ctx, 1, newID, &DeleteProjectRequest{ConfirmName: "反复测试", Permanent: true}); code != 0 {
				t.Fatalf("purge new=%d", code)
			}
		}
		if _, code := l.Delete(ctx, 1, id, &DeleteProjectRequest{ConfirmName: "反复测试", Permanent: true}); code != 0 {
			t.Fatalf("round=%d purge=%d", round, code)
		}
		for _, query := range []string{
			"SELECT COUNT(*) FROM project WHERE id=?",
			"SELECT COUNT(*) FROM app_service WHERE project_id=?",
			"SELECT COUNT(*) FROM project_member WHERE project_id=?",
			"SELECT COUNT(*) FROM build_run WHERE project_id=?",
			"SELECT COUNT(*) FROM production_release_policy WHERE project_id=?",
			"SELECT COUNT(*) FROM deployment_verification WHERE project_id=?",
			"SELECT COUNT(*) FROM deployment_verification_event WHERE project_id=?",
		} {
			var n int
			if err := db.QueryRow(query, id).Scan(&n); err != nil || n != 0 {
				t.Fatalf("leftover: %s count=%d err=%v", query, n, err)
			}
		}
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM build_log WHERE build_run_id=?", runID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("log remains: %d %v", n, err)
		}
		if err := db.QueryRow("SELECT COUNT(*) FROM build_number_sequence WHERE service_id=?", svcID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("number sequence remains: %d %v", n, err)
		}
	}
	// Concurrent requests must produce one project and a meaningful collision.
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan int, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, code := l.Create(ctx, &CreateProjectRequest{Name: "反复测试", Code: "repeat-test", DeliveryType: DeliveryTypeK8sYAML}, 1)
			results <- code
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for code := range results {
		if code == 0 {
			successes++
		} else if code != errcode.CodeProjectNameExists {
			t.Fatalf("concurrent creation code=%d", code)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent creations succeeded=%d", successes)
	}
	// A disabled project still reserves its code.
	exec("UPDATE project SET status='disabled' WHERE code='repeat-test' AND status='active'")
	if _, code := l.Create(ctx, &CreateProjectRequest{Name: "反复测试", Code: "repeat-test", DeliveryType: DeliveryTypeK8sYAML}, 1); code != errcode.CodeProjectNameExists {
		t.Fatalf("disabled duplicate allowed=%d", code)
	}
	if _, code := l.Delete(ctx, 1, archivedID, &DeleteProjectRequest{ConfirmName: "反复测试", Permanent: true}); code != 0 {
		t.Fatalf("legacy archive purge=%d", code)
	}
}
