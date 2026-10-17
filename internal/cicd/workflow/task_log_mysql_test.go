package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

// All test writes are confined to a unique disposable schema.
func TestMySQLTaskLogsSurviveRepositoryRestart(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("test config could not be loaded")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("MySQL integration test not configured")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("open test database failed")
	}
	defer admin.Close()
	schema := fmt.Sprintf("axiops_task_log_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName = schema
	database, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	_, err = database.Exec(`CREATE TABLE workflow_task_run (workflow_run_id BIGINT,node_key VARCHAR(64),status VARCHAR(16),log_text MEDIUMTEXT NULL, PRIMARY KEY(workflow_run_id,node_key))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = database.Exec(`INSERT INTO workflow_task_run VALUES (1,'deploy','running',NULL)`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(database)
	if err := repo.AppendTaskLog(ctx, 1, "deploy", "[apply] submitted"); err != nil {
		t.Fatal(err)
	}
	// New repository instance reads durable evidence and deduplicates the same observation.
	repo = NewRepository(database)
	if err := repo.AppendTaskLog(ctx, 1, "deploy", "[apply] submitted"); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendTaskLog(ctx, 1, "deploy", "[failed] CrashLoopBackOff exitCode=1"); err != nil {
		t.Fatal(err)
	}
	var text string
	if err := database.QueryRow(`SELECT log_text FROM workflow_task_run`).Scan(&text); err != nil {
		t.Fatal(err)
	}
	if strings.Count(text, "[apply]") != 1 || !strings.Contains(text, "CrashLoopBackOff") {
		t.Fatalf("unexpected log %q", text)
	}
	if _, err = database.Exec(`UPDATE workflow_task_run SET status='failed'`); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendTaskLog(ctx, 1, "deploy", "stale observation"); err != nil {
		t.Fatal(err)
	}
	var terminal string
	if err := database.QueryRow(`SELECT log_text FROM workflow_task_run`).Scan(&terminal); err != nil {
		t.Fatal(err)
	}
	if terminal != text {
		t.Fatal("terminal evidence was changed by stale poll")
	}
}
