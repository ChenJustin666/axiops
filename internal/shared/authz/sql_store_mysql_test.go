package authz

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

// This test creates and drops only its own schema. Application users and roles
// are never edited, and the DSN/password must never appear in assertion output.
func isolatedAuthorizationDB(t *testing.T) *sql.DB {
	t.Helper()
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
		t.Fatal("could not open test MySQL")
	}
	t.Cleanup(func() { admin.Close() })
	schema := fmt.Sprintf("axiops_authz_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName, cfg.ParseTime = schema, true
	database, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("could not open isolated MySQL")
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestMySQLCurrentPolicyRevocationAndScopes(t *testing.T) {
	database := isolatedAuthorizationDB(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		"CREATE TABLE user (id BIGINT PRIMARY KEY, status INT, deleted_at DATETIME NULL, department_id BIGINT)",
		"CREATE TABLE project (id BIGINT PRIMARY KEY, status VARCHAR(32))",
		"CREATE TABLE user_role (user_id BIGINT, role_id BIGINT)",
		"CREATE TABLE project_member (user_id BIGINT, role_id BIGINT, project_id BIGINT)",
		"CREATE TABLE role_permission (role_id BIGINT, resource VARCHAR(64), action VARCHAR(64), effect VARCHAR(16))",
		"CREATE TABLE role_binding (id BIGINT PRIMARY KEY, subject_type VARCHAR(32), subject_id BIGINT, role_id BIGINT, scope_type VARCHAR(32), scope_id BIGINT, environment VARCHAR(64), status INT, expires_at DATETIME NULL)",
		"INSERT INTO user VALUES (7,1,NULL,10)",
		"INSERT INTO project VALUES (19,'active'),(20,'active')",
		"INSERT INTO role_permission VALUES (1,'*','*','allow'),(2,'service','update','allow'),(3,'workflow','run','allow'),(4,'service','update','deny')",
	} {
		exec(statement)
	}
	a := NewSQLAuthorizer(database)
	ctx := context.Background()
	check := func(name string, request Request, want bool) {
		t.Helper()
		got, err := a.Authorize(ctx, 7, request)
		if err != nil || got != want {
			t.Fatalf("%s: allowed=%v want=%v error=%v", name, got, want, err)
		}
	}
	service := Request{Resource: "service", Action: "update", ProjectID: 19, ServiceID: 50}
	exec("INSERT INTO user_role VALUES (7,1)")
	check("platform role", Request{Resource: "cluster", Action: "delete", ClusterID: 5}, true)
	exec("DELETE FROM user_role WHERE user_id=7")
	check("revoked role without token replacement", Request{Resource: "cluster", Action: "delete", ClusterID: 5}, false)
	exec("INSERT INTO project_member VALUES (7,2,19)")
	check("own project service", service, true)
	check("other project service", Request{Resource: "service", Action: "update", ProjectID: 20, ServiceID: 51}, false)
	exec("INSERT INTO role_binding VALUES (1,'department',10,3,'environment',22,'qa',1,NULL)")
	workflow := Request{Resource: "workflow", Action: "run", ProjectID: 19, WorkflowID: 11, EnvironmentID: 22, Environment: "qa"}
	check("department environment grant", workflow, true)
	check("same code another environment", Request{Resource: "workflow", Action: "run", ProjectID: 20, WorkflowID: 12, EnvironmentID: 23, Environment: "qa"}, false)
	exec("UPDATE user SET department_id=11 WHERE id=7")
	check("department reassignment revokes", workflow, false)
	exec("INSERT INTO role_binding VALUES (2,'user',7,3,'environment',22,'qa',1,DATE_SUB(NOW(),INTERVAL 1 SECOND))")
	check("expired direct grant", workflow, false)
	exec("UPDATE role_binding SET expires_at=DATE_ADD(NOW(),INTERVAL 1 HOUR) WHERE id=2")
	check("renewed direct grant", workflow, true)
	exec("UPDATE role_binding SET status=0 WHERE id=2")
	check("disabled direct grant", workflow, false)
	exec("INSERT INTO role_binding VALUES (3,'user',7,4,'service',50,'*',1,NULL)")
	check("service deny overrides project allow", service, false)
	exec("DELETE FROM role_binding WHERE id=3")
	check("deny removal restores project allow", service, true)
	exec("UPDATE project SET status='deleted' WHERE id=19")
	check("deleted project rejects mutation", service, false)
	exec("UPDATE project SET status='active' WHERE id=19")
	exec("UPDATE user SET status=0 WHERE id=7")
	check("disabled user", service, false)
	exec("UPDATE user SET status=1,deleted_at=NOW() WHERE id=7")
	check("deleted user", service, false)
}
