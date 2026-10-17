package project

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
	"github.com/zeromicro/go-zero/core/conf"
)

// All mutations are confined to a disposable schema, never application records.
func TestMySQLInitializationResumesAndRejectsStaleCompletionOverwrite(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("test configuration could not be loaded")
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
	schema := fmt.Sprintf("axiops_init_test_%d", time.Now().UnixNano())
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
	_, err = db.Exec(`CREATE TABLE project (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(128), code VARCHAR(64), description TEXT,
		status VARCHAR(20), delivery_type VARCHAR(32), delivery_status VARCHAR(32), owner_user_id BIGINT,
		created_by BIGINT, initialization_status VARCHAR(20), initialization_step INT,
		initialized_at DATETIME(6) NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(db)
	p := &Project{Name: "resume", Code: "resume", Status: "active", DeliveryType: DeliveryTypeK8sYAML,
		DeliveryStatus: DeliveryStatusActive, InitializationStatus: InitializationStatusPending}
	p.ID, err = repo.Create(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	p.InitializationStatus, p.InitializationStep = InitializationStatusInProgress, 2
	if err := repo.Update(ctx, p); err != nil {
		t.Fatal(err)
	}
	resumed, err := NewRepository(db).FindByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.InitializationStatus != InitializationStatusInProgress || resumed.InitializationStep != 2 || resumed.InitializedAt != nil {
		t.Fatalf("saved initialization did not survive a fresh repository read: %+v", resumed)
	}
	stale := *resumed
	now := time.Now().Truncate(time.Microsecond)
	resumed.InitializationStatus, resumed.InitializationStep, resumed.InitializedAt = InitializationStatusCompleted, 4, &now
	if err := repo.Update(ctx, resumed); err != nil {
		t.Fatal(err)
	}
	// Simulate a request which read the row before another tab completed it.
	stale.InitializationStep = 1
	if err := repo.Update(ctx, &stale); err != nil {
		t.Fatal(err)
	}
	final, err := repo.FindByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.InitializationStatus != InitializationStatusCompleted || final.InitializationStep != 4 || final.InitializedAt == nil || !final.InitializedAt.Equal(now) {
		t.Fatalf("a stale write regressed completion: %+v", final)
	}
	if _, err := db.Exec("UPDATE project SET status='deleted' WHERE id=?", p.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, &stale); err != projectstate.ErrDeleted {
		t.Fatalf("stale update resurrected project: %v", err)
	}
	for _, status := range []string{"", "active", "deleted"} {
		list, total, err := repo.List(ctx, authz.ScopeSet{AllowAll: true}, "", status, 1, 20)
		want := int64(0)
		if status == "deleted" {
			want = 1
		}
		if err != nil || total != want || int64(len(list)) != want {
			t.Fatalf("filter=%s total=%d err=%v", status, total, err)
		}
	}
	archived, err := repo.FindByCode(ctx, p.Code)
	if err != nil || archived != nil {
		t.Fatalf("archived code still reserved: %v", err)
	}
}
