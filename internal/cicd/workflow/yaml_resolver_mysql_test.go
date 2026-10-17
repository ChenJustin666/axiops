package workflow

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

// All writes go to a unique disposable schema, never to application records.
func TestMySQLYAMLVariablesAreScopedAndLayered(t *testing.T) {
	dsn := os.Getenv("AXIOPS_TEST_MYSQL_DSN")
	if path := os.Getenv("AXIOPS_TEST_CONFIG"); dsn == "" && path != "" {
		var c config.PlatformAPIConfig
		if err := conf.Load(path, &c); err != nil {
			t.Fatal("test configuration could not be loaded")
		}
		dsn = c.MySQL.DSN()
	}
	if dsn == "" {
		t.Skip("AXIOPS_TEST_MYSQL_DSN is not set")
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
	schema := fmt.Sprintf("axiops_yaml_test_%d", time.Now().UnixNano())
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
	statements := []string{
		`CREATE TABLE project (id BIGINT, code TEXT)`,
		`CREATE TABLE cluster (id BIGINT, name TEXT)`,
		`CREATE TABLE project_environment (id BIGINT, project_id BIGINT, environment_type TEXT, binding_status TEXT, status TEXT, cluster_id BIGINT, namespace TEXT, code TEXT)`,
		`CREATE TABLE app_service (id BIGINT, project_id BIGINT, delivery_type TEXT, source_type TEXT, status TEXT, image_name TEXT, code TEXT)`,
		`CREATE TABLE service_revision (id BIGINT, service_id BIGINT, definition_format TEXT, definition TEXT)`,
		`CREATE TABLE environment_service_override (project_environment_id BIGINT, service_id BIGINT, image_ref TEXT, replicas INT, environment_yaml_revision_id BIGINT DEFAULT 0)`,
		`CREATE TABLE environment_service_yaml_revision(id BIGINT,environment_id BIGINT,service_id BIGINT,base_revision_id BIGINT,definition TEXT)`,
		`CREATE TABLE environment_resource (id BIGINT, environment_id BIGINT, manifest TEXT, kind TEXT, name TEXT)`,
		`CREATE TABLE variable_set (id BIGINT, project_id BIGINT, environment_id BIGINT, service_id BIGINT, scope TEXT, name TEXT)`,
		`CREATE TABLE variable_item (id BIGINT, variable_set_id BIGINT, variable_key TEXT, value_text TEXT, is_secret INT)`,
		`INSERT INTO project VALUES (19,'risk')`,
		`INSERT INTO cluster VALUES (2,'ci')`,
		`INSERT INTO project_environment VALUES (17,19,'k8s_namespace','validated','active',2,'risk-qa','qa')`,
		`INSERT INTO app_service VALUES (17,19,'k8s_yaml','template','active','risk_server','risk-server')`,
		`INSERT INTO service_revision VALUES (13,17,'k8s_yaml','manifest')`,
		`INSERT INTO variable_set VALUES (1,19,17,0,'environment','default'), (2,19,17,17,'environment','default'), (3,19,17,18,'environment','default'), (4,19,18,17,'environment','default'), (5,20,17,17,'environment','default')`,
		`INSERT INTO variable_item VALUES (1,1,'port','8080',0), (2,1,'replicas','1',0), (3,2,'port','8001',0), (4,3,'port','9000',0), (5,4,'port','9001',0), (6,5,'port','9002',0), (7,2,'TOKEN','private',1)`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	source, err := NewSQLYAMLSourceStore(database).LoadYAMLSource(context.Background(), 17, 17, 13)
	if err != nil {
		t.Fatal(err)
	}
	if source == nil || source.Variables["port"] != "8001" || source.Variables["replicas"] != "1" || len(source.Variables) != 2 {
		t.Fatalf("incorrect variable resolution: %+v", source)
	}
}
