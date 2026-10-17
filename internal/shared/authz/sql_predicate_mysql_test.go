package authz

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Execute actual MySQL predicates, not string snapshots: SQL NULL and collation
// behavior must agree with the concrete-resource evaluator.
func TestMySQLPolicyPredicateParityAndPagination(t *testing.T) {
	database := isolatedAuthorizationDB(t)
	ctx := context.Background()
	data, err := os.ReadFile("testdata/policy_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name     string
		Policies []Policy
		Request  Request
		Want     bool
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			scope, err := New(memoryStore(tc.Policies)).Scopes(ctx, 7, tc.Request.Resource, tc.Request.Action)
			if err != nil {
				t.Fatal(err)
			}
			predicate, args := scope.policySQLPredicate(SQLScopeColumns{ProjectID: "project_id", ServiceID: "service_id", BuildID: "build_id", WorkflowID: "workflow_id", ClusterID: "cluster_id", EnvironmentID: "environment_id", Environment: "environment"})
			r := tc.Request
			args = append(args, r.ProjectID, r.ServiceID, r.BuildID, r.WorkflowID, r.ClusterID, r.EnvironmentID, r.Environment)
			var got bool
			if err := database.QueryRowContext(ctx, "SELECT ("+predicate+") FROM (SELECT ? project_id,? service_id,? build_id,? workflow_id,? cluster_id,? environment_id,? environment) target", args...).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tc.Want {
				t.Fatalf("SQL=%v concrete=%v", got, tc.Want)
			}
		})
	}
	if _, err := database.Exec(`CREATE TABLE targets (id BIGINT PRIMARY KEY, project_id BIGINT NULL, workflow_id BIGINT NULL, environment_id BIGINT NULL, code VARCHAR(32) NULL, cluster_id BIGINT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO targets VALUES (1,10,101,7,'qa',2),(2,10,102,8,'prod',3),(3,20,103,9,'qa',2),(4,10,104,10,'QA',2),(5,NULL,NULL,NULL,NULL,NULL)`); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		policies memoryStore
		want     []int64
	}{
		{"global allow with project production deny", memoryStore{{Effect: "allow", ScopeType: ScopePlatform}, {Effect: "deny", ScopeType: ScopeProject, ScopeID: 10, Environment: "prod"}}, []int64{1, 3, 4, 5}},
		{"project AND environment", memoryStore{{Effect: "allow", ScopeType: ScopeProject, ScopeID: 10, Environment: "qa"}}, []int64{1}},
		{"environment ID AND code", memoryStore{{Effect: "allow", ScopeType: ScopeEnvironment, ScopeID: 8, Environment: "qa"}}, []int64{}},
		{"cluster with environment deny", memoryStore{{Effect: "allow", ScopeType: ScopeCluster, ScopeID: 2}, {Effect: "deny", ScopeType: ScopeEnvironment, ScopeID: 7, Environment: "qa"}}, []int64{3, 4}},
		{"NULL must not nullify scoped deny", memoryStore{{Effect: "allow", ScopeType: ScopePlatform}, {Effect: "deny", ScopeType: ScopeEnvironment, ScopeID: 7}}, []int64{2, 3, 4, 5}},
		{"case sensitive condition", memoryStore{{Effect: "allow", ScopeType: ScopePlatform, Environment: "QA"}}, []int64{4}},
		{"global deny", memoryStore{{Effect: "allow", ScopeType: ScopeProject, ScopeID: 10}, {Effect: "deny", ScopeType: ScopePlatform}}, []int64{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i := range tc.policies {
				tc.policies[i].Resource, tc.policies[i].Action = "workflow", "view"
			}
			a := New(tc.policies)
			scope, err := a.Scopes(ctx, 7, "workflow", "view")
			if err != nil {
				t.Fatal(err)
			}
			predicate, args := scope.WorkflowSQLPredicate("project_id", "workflow_id", "environment_id", "code", "cluster_id")
			var total int
			if err := database.QueryRow("SELECT COUNT(*) FROM targets WHERE "+predicate, args...).Scan(&total); err != nil {
				t.Fatal(err)
			}
			if total != len(tc.want) {
				t.Fatalf("count=%d want=%d", total, len(tc.want))
			}
			environmentPredicate, environmentArgs := scope.EnvironmentSQLPredicate("project_id", "environment_id", "code", "cluster_id")
			var environmentCount int
			if err := database.QueryRow("SELECT COUNT(*) FROM targets WHERE "+environmentPredicate, environmentArgs...).Scan(&environmentCount); err != nil {
				t.Fatal(err)
			}
			if environmentCount != total {
				t.Fatalf("environment list count=%d workflow count=%d", environmentCount, total)
			}
			ids := []int64{}
			for offset := 0; offset < total; offset++ {
				var id int64
				pageArgs := append(append([]interface{}{}, args...), 1, offset)
				if err := database.QueryRow("SELECT id FROM targets WHERE "+predicate+" ORDER BY id LIMIT ? OFFSET ?", pageArgs...).Scan(&id); err != nil {
					t.Fatal(err)
				}
				ids = append(ids, id)
			}
			if !reflect.DeepEqual(ids, tc.want) {
				t.Fatalf("rows=%v want=%v", ids, tc.want)
			}
			rows, err := database.Query("SELECT COALESCE(project_id,0),COALESCE(workflow_id,0),COALESCE(environment_id,0),COALESCE(code,''),COALESCE(cluster_id,0),id FROM targets ORDER BY id")
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			for rows.Next() {
				r := Request{Resource: "workflow", Action: "view"}
				var id int64
				if err := rows.Scan(&r.ProjectID, &r.WorkflowID, &r.EnvironmentID, &r.Environment, &r.ClusterID, &id); err != nil {
					t.Fatal(err)
				}
				got, err := a.Authorize(ctx, 7, r)
				if err != nil {
					t.Fatal(err)
				}
				want := false
				for _, visible := range ids {
					if visible == id {
						want = true
					}
				}
				if got != want {
					t.Fatalf("id=%d list=%v detail=%v", id, want, got)
				}
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
