package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

func TestMySQLProductionClassificationRepair(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL test configuration not set")
	}
	var cfg config.PlatformAPIConfig
	if err := conf.Load(path, &cfg); err != nil {
		t.Fatal("could not load test configuration")
	}
	dsn, err := mysql.ParseDSN(cfg.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	schema := fmt.Sprintf("axiops_classification_test_%d", time.Now().UnixNano())
	if _, err = admin.ExecContext(ctx, "CREATE DATABASE `"+schema+"`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	dsn.DBName = schema
	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, `CREATE TABLE project_environment(id BIGINT PRIMARY KEY, project_id BIGINT, name VARCHAR(64), code VARCHAR(32), namespace VARCHAR(63), is_production BOOL, deploy_approval BOOL)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO project_environment VALUES
		(1,26,'生产环境','prod','rsc01-prod',0,0),
		(2,27,'Live','production','app-live',0,0),
		(3,26,'测试环境','qa','rsc01-qa',0,0),
		(4,26,'预发布','preprod','app-preprod',0,1),
		(5,28,'生产环境','custom','custom-prod',0,0),
		(6,29,'Production','prod','existing-live',1,0)`); err != nil {
		t.Fatal(err)
	}
	body, err := fs.ReadFile(migrationFiles, "050_production_environment_classification.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements, err := splitStatements(string(body))
	if err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		for _, statement := range statements {
			r, err := db.ExecContext(ctx, statement)
			if err != nil {
				t.Fatal(err)
			}
			if pass == 1 {
				n, _ := r.RowsAffected()
				if n != 0 {
					t.Fatal("repair was not idempotent")
				}
			}
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT id,is_production,deploy_approval,namespace FROM project_environment ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := []struct {
		production, approval bool
		namespace            string
	}{
		{true, true, "rsc01-prod"}, {true, true, "app-live"}, {false, false, "rsc01-qa"},
		{false, true, "app-preprod"}, {false, false, "custom-prod"}, {true, false, "existing-live"},
	}
	count := 0
	for rows.Next() {
		var id int
		var production, approval bool
		var namespace string
		if err := rows.Scan(&id, &production, &approval, &namespace); err != nil {
			t.Fatal(err)
		}
		w := want[id-1]
		if production != w.production || approval != w.approval || namespace != w.namespace {
			t.Fatalf("unexpected environment repair: id=%d", id)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != len(want) {
		t.Fatal("environment records were lost")
	}
}
