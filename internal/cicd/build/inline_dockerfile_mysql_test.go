package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMySQLInlineDockerfileMigrationAndRoundTrip(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL not configured")
	}
	var c config.PlatformAPIConfig
	if conf.Load(path, &c) != nil {
		t.Fatal("cannot load config")
	}
	cfg, e := mysql.ParseDSN(c.MySQL.DSN())
	if e != nil {
		t.Fatal("invalid DSN")
	}
	admin, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_inline_test_%d", time.Now().UnixNano())
	if _, e = admin.Exec("CREATE DATABASE `" + schema + "`"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := admin.Exec("DROP DATABASE `" + schema + "`"); e != nil {
			t.Error(e)
		}
	}()
	cfg.DBName = schema
	cfg.ParseTime = true
	db, e := sql.Open("mysql", cfg.FormatDSN())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	exec := func(q string) {
		t.Helper()
		if _, e := db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	exec("CREATE TABLE build_config LIKE " + source + ".build_config")
	var has int
	if e = db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='build_config' AND column_name='dockerfile_source'`).Scan(&has); e != nil {
		t.Fatal(e)
	}
	if has > 0 {
		exec(`ALTER TABLE build_config DROP COLUMN dockerfile_source,DROP COLUMN dockerfile_content`)
	}
	migration, e := os.ReadFile("../../../migrations/045_inline_dockerfile.sql")
	if e != nil {
		t.Fatal(e)
	}
	exec(string(migration))
	repo := NewRepository(db)
	ctx := context.Background()
	content := "FROM eclipse-temurin:17-jre\n# 中文与 $JAVA_OPTS 保留\nCOPY target/app.jar app.jar\n"
	config := &BuildConfig{ServiceID: 1, TemplateType: "custom", BuildMode: "jar_image", Provider: KubeJobProviderName, BuildImage: "maven:3.9-eclipse-temurin-17", DockerfilePath: "Dockerfile", BuildContext: ".", DockerfileSource: "inline", DockerfileContent: content, ImageRepo: "registry.example/team/app", TimeoutSeconds: 1800}
	if e = repo.UpsertConfig(ctx, config); e != nil {
		t.Fatal(e)
	}
	saved, e := repo.FindConfigByServiceID(ctx, 1)
	if e != nil || saved.DockerfileSource != "inline" || saved.DockerfileContent != content {
		t.Fatal("saved Dockerfile did not round-trip")
	}
	frozen, e := json.Marshal(saved)
	if e != nil {
		t.Fatal(e)
	}
	config.DockerfileSource = "repository"
	config.DockerfileContent = ""
	if e = repo.UpsertConfig(ctx, config); e != nil {
		t.Fatal(e)
	}
	saved, e = repo.FindConfigByServiceID(ctx, 1)
	if e != nil || saved.DockerfileSource != "repository" || saved.DockerfileContent != "" {
		t.Fatal("switching back to repository retained inline content")
	}
	var queued BuildConfig
	if e = json.Unmarshal(frozen, &queued); e != nil || queued.DockerfileContent != content || queued.DockerfileSource != "inline" {
		t.Fatal("saved snapshot changed with configuration")
	}
}
