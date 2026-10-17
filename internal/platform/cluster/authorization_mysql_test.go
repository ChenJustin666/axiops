package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/zeromicro/go-zero/core/conf"
)

type clusterPolicyStore []authz.Policy

func (p clusterPolicyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return p, nil
}

func TestMySQLProjectClusterDiscoveryPreservesDenials(t *testing.T) {
	path := os.Getenv("AXIOPS_TEST_CONFIG")
	if path == "" {
		t.Skip("MySQL test configuration is not set")
	}
	var c config.PlatformAPIConfig
	if conf.Load(path, &c) != nil {
		t.Fatal("cannot load test config")
	}
	cfg, err := mysql.ParseDSN(c.MySQL.DSN())
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("cannot open MySQL")
	}
	defer admin.Close()
	source := "`" + strings.ReplaceAll(cfg.DBName, "`", "``") + "`"
	schema := fmt.Sprintf("axiops_cluster_access_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + schema + "`"); err != nil {
			t.Error(err)
		}
	}()
	cfg.DBName, cfg.ParseTime = schema, true
	database, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("cannot open isolated MySQL")
	}
	defer database.Close()
	for _, table := range []string{"cluster", "project_environment"} {
		if _, err := database.Exec("CREATE TABLE `" + table + "` LIKE " + source + ".`" + table + "`"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.Exec(`INSERT INTO cluster(id,name,code) VALUES(51,'Test','authz-51'),(52,'Other','authz-52')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO project_environment(id,project_id,name,code,environment_type,cluster_id,namespace) VALUES(7,19,'Test','qa','k8s_namespace',51,'authz-qa')`); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		policies clusterPolicyStore
		want     []int64
	}{
		{"attached cluster discovery", clusterPolicyStore{{Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}}, []int64{51}},
		{"global deny cannot be bypassed", clusterPolicyStore{{Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}, {Effect: "deny", ScopeType: authz.ScopePlatform}}, []int64{}},
		{"project deny cannot be bypassed", clusterPolicyStore{{Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}, {Effect: "deny", ScopeType: authz.ScopeProject, ScopeID: 19}}, []int64{}},
		{"cluster deny cannot be bypassed", clusterPolicyStore{{Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 19}, {Effect: "deny", ScopeType: authz.ScopeCluster, ScopeID: 51}}, []int64{}},
		{"global allow preserves cluster deny", clusterPolicyStore{{Effect: "allow", ScopeType: authz.ScopePlatform}, {Effect: "deny", ScopeType: authz.ScopeCluster, ScopeID: 51}}, []int64{52}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i := range tc.policies {
				tc.policies[i].Resource, tc.policies[i].Action = "cluster", "view"
			}
			scope, err := authz.New(tc.policies).Scopes(context.Background(), 7, "cluster", "view")
			if err != nil {
				t.Fatal(err)
			}
			items, err := NewRepository(database).List(context.Background(), scope, 19)
			if err != nil {
				t.Fatal(err)
			}
			ids := []int64{}
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			if !reflect.DeepEqual(ids, tc.want) {
				t.Fatalf("visible clusters=%v want=%v", ids, tc.want)
			}
		})
	}
}
