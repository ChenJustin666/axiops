package sysauthservicelogic

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/authz"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/portal-rpc/pb"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/pkg/jwt"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestMySQLLegacyPrincipalRevocation(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("could not load test config")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("MySQL test configuration is not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid MySQL test configuration")
	}
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("could not open test MySQL")
	}
	t.Cleanup(func() { admin.Close() })
	schema := fmt.Sprintf("axiops_legacy_auth_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName, cfg.ParseTime = schema, true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("could not open isolated MySQL")
	}
	t.Cleanup(func() { db.Close() })
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, stmt := range []string{
		"CREATE TABLE sys_user (id BIGINT PRIMARY KEY, username VARCHAR(64), nickname VARCHAR(64), status INT, is_deleted INT, is_need_reset_pwd INT)",
		"CREATE TABLE sys_role (id BIGINT PRIMARY KEY, code VARCHAR(64), is_deleted INT)",
		"CREATE TABLE sys_user_role (user_id BIGINT, role_id BIGINT, is_deleted INT)",
		"CREATE TABLE sys_token (id BIGINT PRIMARY KEY, token VARCHAR(255), owner_type INT, owner_id BIGINT, name VARCHAR(64), expire_time DATETIME NULL, status INT, is_deleted INT)",
		"INSERT INTO sys_user VALUES (7,'operator','Original',1,0,0)",
		"INSERT INTO sys_role VALUES (1,'SUPER_ADMIN',0),(2,'viewer',0),(3,'Viewer',0),(4,'  ',0)",
		"INSERT INTO sys_user_role VALUES (7,1,0),(7,2,0),(7,3,0),(7,4,0),(7,99,0)",
		"INSERT INTO sys_token VALUES (10,'CaseSensitiveToken',1,7,'test-user',NULL,1,0),(11,'role-token',2,2,'test-role',NULL,1,0)",
	} {
		exec(stmt)
	}
	reader := authz.NewSQLPrincipalReader(db)
	l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: reader})
	claims := &jwt.JWTClaims{UserName: &jwt.AccountInfo{UserId: 7, UserName: "operator", Roles: []string{"SUPER_ADMIN"}, Uuid: "unchanged-session"}}
	check := func(name string, valid bool, roles []string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			res, err := l.currentJWTPrincipal(claims)
			if err != nil || res.IsValid != valid {
				t.Fatalf("valid=%v response=%+v err=%v", valid, res, err)
			}
			if valid && !reflect.DeepEqual(res.Roles, roles) {
				t.Fatalf("roles=%v want=%v", res.Roles, roles)
			}
			account, refreshErr := NewRefreshTokenLogic(context.Background(), l.svcCtx).currentRefreshAccount(claims)
			if valid && (refreshErr != nil || account == nil || !reflect.DeepEqual(account.Roles, roles)) {
				t.Fatalf("refresh did not use current roles: %+v %v", account, refreshErr)
			}
			if !valid && (refreshErr == nil || account != nil) {
				t.Fatal("inactive account accepted by refresh")
			}
		})
	}
	check("initial roles deduplicate and ignore missing roles", true, []string{"SUPER_ADMIN", "viewer"})
	exec("UPDATE sys_user_role SET is_deleted=1 WHERE role_id=1")
	check("same JWT loses revoked admin", true, []string{"viewer"})
	exec("UPDATE sys_role SET is_deleted=1 WHERE id IN (2,3)")
	check("same JWT loses deleted roles", true, []string{})
	exec("UPDATE sys_user SET status=0")
	check("disabled user", false, nil)
	exec("UPDATE sys_user SET status=1,is_deleted=1")
	check("deleted user", false, nil)
	exec("UPDATE sys_user SET is_deleted=0,is_need_reset_pwd=1")
	check("password reset required", false, nil)
	exec("UPDATE sys_user SET is_need_reset_pwd=0,username='renamed'")
	check("renamed identity invalidates old JWT", false, nil)
	exec("UPDATE sys_user SET username='operator',nickname='Current'")
	exec("UPDATE sys_role SET is_deleted=0 WHERE id=2")
	check("current role restored", true, []string{"viewer"})
	res, err := l.currentJWTPrincipal(claims)
	if err != nil || res.NickName != "Current" {
		t.Fatal("stale nickname")
	}
	checkToken := func(value string, valid bool, roles []string) {
		t.Helper()
		res, err := l.VerifyToken(&pb.VerifyTokenRequest{Token: "Bearer " + value})
		if err != nil || res.IsValid != valid {
			t.Fatalf("system token valid=%v err=%v", valid, err)
		}
		if valid && !reflect.DeepEqual(res.Roles, roles) {
			t.Fatalf("stale system token roles=%v", res.Roles)
		}
	}
	checkToken("CaseSensitiveToken", true, []string{"viewer"})
	exec("UPDATE sys_user SET status=0 WHERE id=7")
	checkToken("CaseSensitiveToken", false, nil)
	exec("UPDATE sys_user SET status=1 WHERE id=7")
	checkToken("casesensitivetoken", false, nil)
	if token, err := reader.Token(context.Background(), "CaseSensitiveToken"); err != nil || token == nil || token.Token != "" {
		t.Fatal("token reader must not return plaintext")
	}
	exec("UPDATE sys_token SET status=0 WHERE id=10")
	checkToken("CaseSensitiveToken", false, nil)
	exec("UPDATE sys_token SET status=1,expire_time=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE id=10")
	checkToken("CaseSensitiveToken", false, nil)
	exec("UPDATE sys_token SET expire_time=NULL,is_deleted=1 WHERE id=10")
	checkToken("CaseSensitiveToken", false, nil)
	checkToken("role-token", true, []string{"viewer"})
	exec("UPDATE sys_role SET is_deleted=1 WHERE id=2")
	checkToken("role-token", false, nil)
	exec("DROP TABLE sys_user_role")
	if res, err := l.currentJWTPrincipal(claims); err == nil || res != nil {
		t.Fatal("DB failure must not use JWT roles")
	}
}
