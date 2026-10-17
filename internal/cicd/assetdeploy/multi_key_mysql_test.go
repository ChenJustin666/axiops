package assetdeploy

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/zeromicro/go-zero/core/conf"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMySQLPrepareGroupedSecretPreservesAllKeysAndValidatesBeforeWriting(t *testing.T) {
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
	defer admin.Close()
	schema := fmt.Sprintf("axiops_asset_test_%d", time.Now().UnixNano())
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
	if _, err := db.Exec(`CREATE TABLE credential(id BIGINT PRIMARY KEY, project_id BIGINT, type VARCHAR(32), data TEXT)`); err != nil {
		t.Fatal(err)
	}
	box, _ := secretbox.New("test-only-encryption-key")
	encrypted, _ := box.Encrypt(`{"SK":"test-sk","tls.key":"line1\nline2\n"}`)
	if _, err := db.Exec("INSERT INTO credential VALUES(42,19,'secret',?)", encrypted); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	manager := &Manager{db: db, decrypt: box, client: func(context.Context, int64) (kubernetes.Interface, error) { return client, nil }}
	run := workflow.WorkflowRun{ProjectID: 19}
	deploy := workflow.YAMLDeployConfig{ServiceID: 17}
	target := workflow.YAMLTarget{Namespace: "test", ClusterID: 1, AssetSecrets: []serviceassets.SecretRef{
		{Name: "grouped", Key: "SK", CredentialID: 42, CredentialKey: "SK"},
		{Name: "grouped", Key: "tls.key", CredentialID: 42, CredentialKey: "tls.key"},
	}}
	for i := 0; i < 2; i++ {
		if _, err := manager.PrepareYAMLDeployment(ctx, run, deploy, target); err != nil {
			t.Fatal(err)
		}
		got, err := client.CoreV1().Secrets("test").Get(ctx, "grouped", metav1.GetOptions{})
		if err != nil || len(got.Data) != 2 || string(got.Data["SK"]) != "test-sk" || string(got.Data["tls.key"]) != "line1\nline2\n" {
			t.Fatal("one key overwrote another")
		}
	}
	for _, bad := range []string{"missing", "SK"} {
		invalid := target
		invalid.AssetSecrets = append(append([]serviceassets.SecretRef{}, target.AssetSecrets...), serviceassets.SecretRef{Name: "grouped", Key: bad, CredentialID: 42, CredentialKey: bad})
		actions := len(client.Actions())
		if _, err := manager.PrepareYAMLDeployment(ctx, run, deploy, invalid); err == nil {
			t.Fatal("invalid key accepted")
		}
		if len(client.Actions()) != actions {
			t.Fatal("partial writes before validation")
		}
	}
	run.ProjectID = 20
	actions := len(client.Actions())
	if _, err := manager.PrepareYAMLDeployment(ctx, run, deploy, target); err == nil || len(client.Actions()) != actions {
		t.Fatal("cross-project credential allowed")
	}
}
