package credential

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/zeromicro/go-zero/core/conf"
)

type secretChecker struct{ deny string }

func (c *secretChecker) Authorize(_ context.Context, _ int64, r authz.Request) (bool, error) {
	return r.Action != c.deny && r.ProjectID == 19, nil
}
func (c *secretChecker) Scopes(context.Context, int64, string, string) (authz.ScopeSet, error) {
	return authz.ScopeSet{AllowAll: true}, nil
}
func (c *secretChecker) Policies(context.Context, int64) ([]authz.Policy, error) { return nil, nil }

func secretTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if conf.Load(path, &c) != nil {
			t.Fatal("test configuration unavailable")
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
		t.Fatal("database unavailable")
	}
	schema := fmt.Sprintf("axiops_secret_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg.DBName, cfg.ParseTime = schema, true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	_, err = db.Exec(`CREATE TABLE credential(id BIGINT PRIMARY KEY AUTO_INCREMENT,name VARCHAR(128),type VARCHAR(32),description TEXT,data MEDIUMTEXT,project_id BIGINT,created_by BIGINT,created_at DATETIME DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMySQLSecretVersionLifecycleDoesNotExposeOrModifySource(t *testing.T) {
	db := secretTestDB(t)
	ctx := context.Background()
	checker := &secretChecker{}
	logic := NewLogic(db, checker, "test-only-encryption-key")
	id, code := logic.Create(ctx, &CreateCredentialRequest{Name: "auth", Type: "secret", ProjectID: 19, Data: `{"SK":"old-test-sk","token":"old-test-token","remove":"old-test-remove","tls.key":"line1\nline2\n"}`}, 1)
	if code != 0 {
		t.Fatalf("creation failed code=%d", code)
	}
	before, _ := logic.repo.FindByID(ctx, id)
	if !strings.HasPrefix(before.Data, "v1:") || strings.Contains(before.Data, "old-test-sk") {
		t.Fatal("stored plaintext")
	}
	current, code := logic.GetByID(ctx, 1, id)
	if code != 0 || len(current.Keys) != 4 || current.Token == "" {
		t.Fatal("metadata missing")
	}
	rows, code := logic.List(ctx, 1, "secret", 19)
	encoded, _ := json.Marshal(rows)
	if code != 0 || len(rows) != 1 || strings.Contains(string(encoded), "old-test-sk") || strings.Contains(string(encoded), before.Data) {
		t.Fatal("list exposed sensitive content")
	}
	req := &DeriveSecretRequest{Name: "next", Token: current.Token, Retain: []string{"SK", "tls.key"}, Values: map[string]string{"token": "new-test-token", "access-key": "new-test-ak"}}
	nextID, code := logic.DeriveSecret(ctx, 1, id, req)
	if code != 0 || nextID == id {
		t.Fatalf("derive failed code=%d", code)
	}
	next, _ := logic.repo.FindByID(ctx, nextID)
	plain, err := logic.box.Decrypt(next.Data)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := secretFields(plain)
	if err != nil || fields["SK"] != "old-test-sk" || fields["tls.key"] != "line1\nline2\n" || fields["token"] != "new-test-token" || fields["access-key"] != "new-test-ak" {
		t.Fatal("version merge incorrect")
	}
	if _, exists := fields["remove"]; exists {
		t.Fatal("deleted key retained in new version")
	}
	original, _ := logic.repo.FindByID(ctx, id)
	if original.Data != before.Data {
		t.Fatal("old version changed")
	}
	assertCount := func(want int) {
		t.Helper()
		var count int
		if db.QueryRow("SELECT COUNT(*) FROM credential").Scan(&count) != nil || count != want {
			t.Fatal("unexpected credential writes")
		}
	}
	assertCount(2)
	for _, action := range []string{"view", "update", "create"} {
		checker.deny = action
		if _, code := logic.DeriveSecret(ctx, 1, id, req); code != errcode.CodeNoPermission {
			t.Fatal("unauthorized derive succeeded")
		}
		assertCount(2)
	}
	checker.deny = ""
	// A generic credential update after the editor opened invalidates its token.
	if code := logic.Update(ctx, 1, id, &UpdateCredentialRequest{Data: `{"SK":"changed-test-sk","token":"changed-test-token"}`}); code != 0 {
		t.Fatal("source update failed")
	}
	if _, code := logic.DeriveSecret(ctx, 1, id, req); code != errcode.CodeParamInvalid {
		t.Fatal("stale token accepted")
	}
	assertCount(2)
}
