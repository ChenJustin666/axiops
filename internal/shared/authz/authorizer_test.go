package authz

import (
	"context"
	"reflect"
	"testing"
)

type memoryStore []Policy

func (m memoryStore) ListUserPolicies(context.Context, int64) ([]Policy, error) { return m, nil }

type archivedStore struct{ memoryStore }

func (archivedStore) ProjectDeleted(context.Context, int64) (bool, error) { return true, nil }

func TestDeletedProjectIsReadOnlyEvenForAdministrator(t *testing.T) {
	a := New(archivedStore{memoryStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: ScopePlatform}}})
	for _, action := range []string{"view", "read", "create", "update", "delete", "run", "approve", "cancel"} {
		allowed, err := a.Authorize(context.Background(), 1, Request{Resource: "workflow", Action: action, ProjectID: 1})
		if err != nil || allowed != (action == "view" || action == "read") {
			t.Fatalf("action=%s allowed=%v err=%v", action, allowed, err)
		}
	}
}

func TestArchivedProjectPurgeStillRequiresDeletePermission(t *testing.T) {
	for _, policies := range []memoryStore{
		{{Resource: "project", Action: "delete", Effect: "allow", ScopeType: ScopeProject, ScopeID: 1}},
		{{Resource: "project", Action: "view", Effect: "allow", ScopeType: ScopeProject, ScopeID: 1}},
		{{Resource: "project", Action: "delete", Effect: "allow", ScopeType: ScopeProject, ScopeID: 2}},
		{{Resource: "*", Action: "*", Effect: "allow", ScopeType: ScopePlatform}, {Resource: "project", Action: "delete", Effect: "deny", ScopeType: ScopeProject, ScopeID: 1}},
	} {
		a := New(archivedStore{policies})
		allowed, err := a.Authorize(context.Background(), 1, Request{Resource: "project", Action: "delete", ProjectID: 1})
		want := len(policies) == 1 && policies[0].Action == "delete" && policies[0].ScopeID == 1
		if err != nil || allowed != want {
			t.Fatalf("policies=%+v allowed=%v err=%v", policies, allowed, err)
		}
	}
}

func TestAuthorizeScopeAndDenyPrecedence(t *testing.T) {
	a := New(memoryStore{
		{Resource: "build", Action: "view", Effect: "allow", ScopeType: ScopeProject, ScopeID: 10, Environment: "*"},
		{Resource: "build", Action: "view", Effect: "deny", ScopeType: ScopeBuild, ScopeID: 99, Environment: "*"},
	})

	allowed, err := a.Authorize(context.Background(), 7, Request{Resource: "build", Action: "view", ProjectID: 10, BuildID: 98})
	if err != nil || !allowed {
		t.Fatalf("expected project build to be allowed: allowed=%v err=%v", allowed, err)
	}
	allowed, err = a.Authorize(context.Background(), 7, Request{Resource: "build", Action: "view", ProjectID: 10, BuildID: 99})
	if err != nil || allowed {
		t.Fatalf("expected explicit build deny to win: allowed=%v err=%v", allowed, err)
	}
	allowed, err = a.Authorize(context.Background(), 7, Request{Resource: "build", Action: "view", ProjectID: 11, BuildID: 98})
	if err != nil || allowed {
		t.Fatalf("expected unrelated project to be denied: allowed=%v err=%v", allowed, err)
	}
}

func TestLegacyActionAliasesAndEnvironment(t *testing.T) {
	a := New(memoryStore{{Resource: "*", Action: "read", Effect: "allow", ScopeType: ScopeProject, ScopeID: 10, Environment: "test"}})
	allowed, _ := a.Authorize(context.Background(), 7, Request{Resource: "service", Action: "view", ProjectID: 10, Environment: "test"})
	if !allowed {
		t.Fatal("read must remain compatible with view")
	}
	allowed, _ = a.Authorize(context.Background(), 7, Request{Resource: "service", Action: "view", ProjectID: 10, Environment: "prod"})
	if allowed {
		t.Fatal("environment-scoped permission leaked into prod")
	}
}

func TestScopesSQLPredicate(t *testing.T) {
	s := ScopeSet{ProjectIDs: []int64{10}, ServiceIDs: []int64{20}, DeniedBuilds: []int64{99}}
	predicate, args := s.SQLPredicate("project_id", "service_id", "id")
	want := "(project_id IN (?) OR service_id IN (?)) AND NOT (id IN (?))"
	if predicate != want {
		t.Fatalf("predicate=%q want=%q", predicate, want)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(10), int64(20), int64(99)}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestClusterSQLIDPredicate(t *testing.T) {
	s := ScopeSet{ClusterIDs: []int64{2, 3}, DeniedClusters: []int64{3}}
	predicate, args := s.SQLIDPredicate(ScopeCluster, "id")
	want := "(id IN (?,?)) AND NOT (id IN (?))"
	if predicate != want {
		t.Fatalf("predicate=%q want=%q", predicate, want)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(2), int64(3), int64(3)}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestPlatformScopeSQLIDPredicateAllowsAll(t *testing.T) {
	s := ScopeSet{AllowAll: true, DeniedClusters: []int64{9}}
	predicate, args := s.SQLIDPredicate(ScopeCluster, "id")
	if predicate != "(1 = 1) AND NOT (id IN (?))" {
		t.Fatalf("predicate=%q", predicate)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(9)}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestWorkflowScopeAndSQLPredicate(t *testing.T) {
	a := New(memoryStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: ScopeWorkflow, ScopeID: 42},
		{Resource: "workflow", Action: "view", Effect: "deny", ScopeType: ScopeWorkflow, ScopeID: 43},
	})
	allowed, err := a.Authorize(context.Background(), 7, Request{Resource: "workflow", Action: "view", ProjectID: 10, WorkflowID: 42})
	if err != nil || !allowed {
		t.Fatalf("expected selected workflow to be allowed: allowed=%v err=%v", allowed, err)
	}
	allowed, err = a.Authorize(context.Background(), 7, Request{Resource: "workflow", Action: "view", ProjectID: 10, WorkflowID: 43})
	if err != nil || allowed {
		t.Fatalf("expected denied workflow to remain denied: allowed=%v err=%v", allowed, err)
	}

	s := ScopeSet{ProjectIDs: []int64{10}, WorkflowIDs: []int64{42}, DeniedWorkflows: []int64{43}}
	predicate, args := s.WorkflowSQLPredicate("project_id", "id")
	want := "(project_id IN (?) OR id IN (?)) AND NOT (id IN (?))"
	if predicate != want {
		t.Fatalf("predicate=%q want=%q", predicate, want)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(10), int64(42), int64(43)}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestPoliciesRejectsAnonymousUser(t *testing.T) {
	a := New(memoryStore{{Resource: "*", Action: "*", Effect: "allow", ScopeType: ScopePlatform}})
	policies, err := a.Policies(context.Background(), 0)
	if err != nil || len(policies) != 0 {
		t.Fatalf("anonymous policies=%v err=%v", policies, err)
	}
}

func TestEnvironmentScopeMatchesEnvironmentID(t *testing.T) {
	a := New(memoryStore{{Resource: "release", Action: "deploy", Effect: "allow", ScopeType: ScopeEnvironment, ScopeID: 12, Environment: "prod"}})
	allowed, err := a.Authorize(context.Background(), 7, Request{Resource: "release", Action: "deploy", EnvironmentID: 12, Environment: "prod"})
	if err != nil || !allowed {
		t.Fatalf("expected matching environment to be allowed: allowed=%v err=%v", allowed, err)
	}
	allowed, err = a.Authorize(context.Background(), 7, Request{Resource: "release", Action: "deploy", EnvironmentID: 13, Environment: "prod"})
	if err != nil || allowed {
		t.Fatalf("expected other environment id to be denied: allowed=%v err=%v", allowed, err)
	}
}

func TestEnvironmentSQLPredicate(t *testing.T) {
	s := ScopeSet{ProjectIDs: []int64{10}, EnvironmentIDs: []int64{20}, DeniedEnvironments: []int64{21}}
	predicate, args := s.EnvironmentSQLPredicate("project_id", "id", "code")
	want := "(project_id IN (?) OR id IN (?)) AND NOT (id IN (?))"
	if predicate != want {
		t.Fatalf("predicate=%q want=%q", predicate, want)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(10), int64(20), int64(21)}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestEnvironmentScopesTranslateProjectEnvironmentLabel(t *testing.T) {
	a := New(memoryStore{{
		Resource: "environment", Action: "view", Effect: "allow",
		ScopeType: ScopeProject, ScopeID: 10, Environment: "test",
	}})
	scopes, err := a.Scopes(context.Background(), 7, "environment", "view")
	if err != nil {
		t.Fatal(err)
	}
	predicate, args := scopes.EnvironmentSQLPredicate("project_id", "id", "code")
	if predicate != "((project_id <=> ? AND BINARY code <=> BINARY ?))" {
		t.Fatalf("unexpected predicate: %s", predicate)
	}
	if !reflect.DeepEqual(args, []interface{}{int64(10), "test"}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestEnvironmentScopesKeepEnvironmentIDWithLabel(t *testing.T) {
	a := New(memoryStore{{
		Resource: "environment", Action: "view", Effect: "allow",
		ScopeType: ScopeEnvironment, ScopeID: 12, Environment: "prod",
	}})
	scopes, err := a.Scopes(context.Background(), 7, "environment", "view")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scopes.EnvironmentIDs, []int64{12}) {
		t.Fatalf("unexpected environment scopes: %#v", scopes.EnvironmentIDs)
	}
}
