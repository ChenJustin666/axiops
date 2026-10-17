package authz

import (
	"context"
	"fmt"
	"strings"
)

const (
	ScopePlatform    = "platform"
	ScopeProject     = "project"
	ScopeService     = "service"
	ScopeBuild       = "build"
	ScopeWorkflow    = "workflow"
	ScopeCluster     = "cluster"
	ScopeEnvironment = "environment"
)

type Request struct {
	Resource      string
	Action        string
	ProjectID     int64
	ServiceID     int64
	BuildID       int64
	WorkflowID    int64
	ClusterID     int64
	EnvironmentID int64
	Environment   string
}

type Policy struct {
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Effect      string `json:"effect"`
	ScopeType   string `json:"scope_type"`
	ScopeID     int64  `json:"scope_id"`
	Environment string `json:"environment"`
}

type PolicyStore interface {
	ListUserPolicies(ctx context.Context, userID int64) ([]Policy, error)
}

type projectStateStore interface {
	ProjectDeleted(ctx context.Context, id int64) (bool, error)
}

type Checker interface {
	Authorize(ctx context.Context, userID int64, req Request) (bool, error)
	Scopes(ctx context.Context, userID int64, resource, action string) (ScopeSet, error)
	Policies(ctx context.Context, userID int64) ([]Policy, error)
}

func (a *Authorizer) Policies(ctx context.Context, userID int64) ([]Policy, error) {
	if userID <= 0 {
		return nil, nil
	}
	return a.store.ListUserPolicies(ctx, userID)
}

type Authorizer struct {
	store PolicyStore
}

func New(store PolicyStore) *Authorizer {
	return &Authorizer{store: store}
}

func (a *Authorizer) Authorize(ctx context.Context, userID int64, req Request) (bool, error) {
	if userID <= 0 || req.Resource == "" || req.Action == "" {
		return false, nil
	}
	// Project deletion is also the entry point for purging archived history.
	// Every other mutation of an archived project remains forbidden.
	if req.ProjectID > 0 && req.Action != "view" && req.Action != "read" && !(req.Resource == "project" && req.Action == "delete") {
		if store, ok := a.store.(projectStateStore); ok {
			deleted, err := store.ProjectDeleted(ctx, req.ProjectID)
			if err != nil || deleted {
				return false, err
			}
		}
	}
	policies, err := a.store.ListUserPolicies(ctx, userID)
	if err != nil {
		return false, err
	}

	allowed := false
	for _, policy := range policies {
		if !permissionMatches(policy, req.Resource, req.Action) || !scopeMatches(policy, req) {
			continue
		}
		if strings.EqualFold(policy.Effect, "deny") {
			return false, nil
		}
		if strings.EqualFold(policy.Effect, "allow") {
			allowed = true
		}
	}
	return allowed, nil
}

type ScopeSet struct {
	// Preserve complete policies for SQL compilation. The flattened fields below
	// remain for compatibility, but cannot represent scope AND environment.
	policies                  []Policy
	policyBacked              bool
	AllowAll                  bool
	DenyAll                   bool
	ProjectIDs                []int64
	ServiceIDs                []int64
	BuildIDs                  []int64
	WorkflowIDs               []int64
	ClusterIDs                []int64
	EnvironmentIDs            []int64
	DeniedProjects            []int64
	DeniedServices            []int64
	DeniedBuilds              []int64
	DeniedWorkflows           []int64
	DeniedClusters            []int64
	DeniedEnvironments        []int64
	ProjectEnvironments       []ProjectEnvironmentScope
	DeniedProjectEnvironments []ProjectEnvironmentScope
}

type ProjectEnvironmentScope struct {
	ProjectID   int64
	Environment string
}

func (s ScopeSet) HasAnyAllow() bool {
	if s.policyBacked {
		for _, p := range s.policies {
			if strings.EqualFold(p.Effect, "deny") && (p.ScopeType == "" || p.ScopeType == ScopePlatform) && (p.Environment == "" || p.Environment == "*") {
				return false
			}
		}
		for _, p := range s.policies {
			if strings.EqualFold(p.Effect, "allow") && validPolicyScope(p) {
				return true
			}
		}
		return false
	}
	return !s.DenyAll && (s.AllowAll || len(s.ProjectIDs) > 0 || len(s.ServiceIDs) > 0 || len(s.BuildIDs) > 0 || len(s.WorkflowIDs) > 0 || len(s.ClusterIDs) > 0 || len(s.EnvironmentIDs) > 0 || len(s.ProjectEnvironments) > 0)
}

func (a *Authorizer) Scopes(ctx context.Context, userID int64, resource, action string) (ScopeSet, error) {
	scopes := ScopeSet{policyBacked: true}
	if userID <= 0 || resource == "" || action == "" {
		return scopes, nil
	}
	policies, err := a.store.ListUserPolicies(ctx, userID)
	if err != nil {
		return scopes, err
	}

	for _, policy := range policies {
		if !permissionMatches(policy, resource, action) {
			continue
		}
		deny := strings.EqualFold(policy.Effect, "deny")
		if !deny && !strings.EqualFold(policy.Effect, "allow") {
			continue
		}
		scopes.policies = append(scopes.policies, policy)
		if policy.Environment != "" && policy.Environment != "*" && policy.ScopeType != ScopeEnvironment {
			if resource == "environment" && policy.ScopeType == ScopeProject {
				appendProjectEnvironmentScope(&scopes, policy.ScopeID, policy.Environment, deny)
			}
			continue
		}
		switch policy.ScopeType {
		case ScopePlatform, "":
			if deny {
				scopes.DenyAll = true
			} else {
				scopes.AllowAll = true
			}
		case ScopeProject:
			appendUnique(scopeSlice(&scopes, ScopeProject, deny), policy.ScopeID)
		case ScopeService:
			appendUnique(scopeSlice(&scopes, ScopeService, deny), policy.ScopeID)
		case ScopeBuild:
			appendUnique(scopeSlice(&scopes, ScopeBuild, deny), policy.ScopeID)
		case ScopeWorkflow:
			appendUnique(scopeSlice(&scopes, ScopeWorkflow, deny), policy.ScopeID)
		case ScopeCluster:
			appendUnique(scopeSlice(&scopes, ScopeCluster, deny), policy.ScopeID)
		case ScopeEnvironment:
			appendUnique(scopeSlice(&scopes, ScopeEnvironment, deny), policy.ScopeID)
		}
	}
	return scopes, nil
}

// EnvironmentSQLPredicate filters environment lists by platform, project, or
// individual environment bindings while preserving explicit deny precedence.
func (s ScopeSet) EnvironmentSQLPredicate(projectColumn, environmentColumn, codeColumn string, clusterColumn ...string) (string, []interface{}) {
	if s.policyBacked {
		columns := SQLScopeColumns{ProjectID: projectColumn, EnvironmentID: environmentColumn, Environment: codeColumn}
		if len(clusterColumn) > 0 {
			columns.ClusterID = clusterColumn[0]
		}
		return s.policySQLPredicate(columns)
	}
	if s.DenyAll {
		return "1 = 0", nil
	}
	var allow, deny []string
	var args []interface{}
	addIn := func(column string, ids []int64, target *[]string) {
		if column == "" || len(ids) == 0 {
			return
		}
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args = append(args, id)
		}
		*target = append(*target, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	if s.AllowAll {
		allow = append(allow, "1 = 1")
	}
	addIn(projectColumn, s.ProjectIDs, &allow)
	addIn(environmentColumn, s.EnvironmentIDs, &allow)
	addProjectEnvironments := func(scopes []ProjectEnvironmentScope, target *[]string) {
		if projectColumn == "" || codeColumn == "" {
			return
		}
		for _, scope := range scopes {
			*target = append(*target, fmt.Sprintf("(%s = ? AND %s = ?)", projectColumn, codeColumn))
			args = append(args, scope.ProjectID, scope.Environment)
		}
	}
	addProjectEnvironments(s.ProjectEnvironments, &allow)
	if len(allow) == 0 {
		return "1 = 0", nil
	}
	predicate := "(" + strings.Join(allow, " OR ") + ")"
	addIn(projectColumn, s.DeniedProjects, &deny)
	addIn(environmentColumn, s.DeniedEnvironments, &deny)
	addProjectEnvironments(s.DeniedProjectEnvironments, &deny)
	if len(deny) > 0 {
		predicate += " AND NOT (" + strings.Join(deny, " OR ") + ")"
	}
	return predicate, args
}

func appendProjectEnvironmentScope(scopes *ScopeSet, projectID int64, environment string, deny bool) {
	if projectID <= 0 || environment == "" || environment == "*" {
		return
	}
	target := &scopes.ProjectEnvironments
	if deny {
		target = &scopes.DeniedProjectEnvironments
	}
	for _, existing := range *target {
		if existing.ProjectID == projectID && existing.Environment == environment {
			return
		}
	}
	*target = append(*target, ProjectEnvironmentScope{ProjectID: projectID, Environment: environment})
}

// SQLPredicate returns a parameterized scope predicate for list queries.
func (s ScopeSet) SQLPredicate(projectColumn, serviceColumn, resourceColumn string) (string, []interface{}) {
	if s.policyBacked {
		return s.policySQLPredicate(SQLScopeColumns{ProjectID: projectColumn, ServiceID: serviceColumn, BuildID: resourceColumn})
	}
	if s.DenyAll {
		return "1 = 0", nil
	}
	var allow []string
	var args []interface{}
	if s.AllowAll {
		allow = append(allow, "1 = 1")
	}
	addIn := func(column string, ids []int64, target *[]string) {
		if column == "" || len(ids) == 0 {
			return
		}
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args = append(args, id)
		}
		*target = append(*target, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	addIn(projectColumn, s.ProjectIDs, &allow)
	addIn(serviceColumn, s.ServiceIDs, &allow)
	addIn(resourceColumn, s.BuildIDs, &allow)
	if len(allow) == 0 {
		return "1 = 0", nil
	}

	predicate := "(" + strings.Join(allow, " OR ") + ")"
	var deny []string
	addIn(projectColumn, s.DeniedProjects, &deny)
	addIn(serviceColumn, s.DeniedServices, &deny)
	addIn(resourceColumn, s.DeniedBuilds, &deny)
	if len(deny) > 0 {
		predicate += " AND NOT (" + strings.Join(deny, " OR ") + ")"
	}
	return predicate, args
}

// WorkflowSQLPredicate combines project-wide and individual workflow grants.
// Optional server-owned target expressions are environment ID, code, cluster ID.
func (s ScopeSet) WorkflowSQLPredicate(projectColumn, workflowColumn string, targetColumns ...string) (string, []interface{}) {
	if s.policyBacked {
		columns := SQLScopeColumns{ProjectID: projectColumn, WorkflowID: workflowColumn}
		if len(targetColumns) > 0 {
			columns.EnvironmentID = targetColumns[0]
		}
		if len(targetColumns) > 1 {
			columns.Environment = targetColumns[1]
		}
		if len(targetColumns) > 2 {
			columns.ClusterID = targetColumns[2]
		}
		return s.policySQLPredicate(columns)
	}
	if s.DenyAll {
		return "1 = 0", nil
	}
	var allow, deny []string
	var args []interface{}
	addIn := func(column string, ids []int64, target *[]string) {
		if column == "" || len(ids) == 0 {
			return
		}
		placeholders := make([]string, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args = append(args, id)
		}
		*target = append(*target, fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ",")))
	}
	if s.AllowAll {
		allow = append(allow, "1 = 1")
	}
	addIn(projectColumn, s.ProjectIDs, &allow)
	addIn(workflowColumn, s.WorkflowIDs, &allow)
	if len(allow) == 0 {
		return "1 = 0", nil
	}
	predicate := "(" + strings.Join(allow, " OR ") + ")"
	addIn(projectColumn, s.DeniedProjects, &deny)
	addIn(workflowColumn, s.DeniedWorkflows, &deny)
	if len(deny) > 0 {
		predicate += " AND NOT (" + strings.Join(deny, " OR ") + ")"
	}
	return predicate, args
}

func (s ScopeSet) SQLIDPredicate(scopeType, column string) (string, []interface{}) {
	if s.policyBacked {
		columns := SQLScopeColumns{}
		switch scopeType {
		case ScopeProject:
			columns.ProjectID = column
		case ScopeService:
			columns.ServiceID = column
		case ScopeBuild:
			columns.BuildID = column
		case ScopeWorkflow:
			columns.WorkflowID = column
		case ScopeCluster:
			columns.ClusterID = column
		case ScopeEnvironment:
			columns.EnvironmentID = column
		default:
			return "1 = 0", nil
		}
		return s.policySQLPredicate(columns)
	}
	translated := ScopeSet{AllowAll: s.AllowAll, DenyAll: s.DenyAll}
	switch scopeType {
	case ScopeProject:
		translated.ProjectIDs = s.ProjectIDs
		translated.DeniedProjects = s.DeniedProjects
	case ScopeCluster:
		translated.ProjectIDs = s.ClusterIDs
		translated.DeniedProjects = s.DeniedClusters
	case ScopeService:
		translated.ProjectIDs = s.ServiceIDs
		translated.DeniedProjects = s.DeniedServices
	case ScopeBuild:
		translated.ProjectIDs = s.BuildIDs
		translated.DeniedProjects = s.DeniedBuilds
	case ScopeWorkflow:
		translated.ProjectIDs = s.WorkflowIDs
		translated.DeniedProjects = s.DeniedWorkflows
	case ScopeEnvironment:
		translated.ProjectIDs = s.EnvironmentIDs
		translated.DeniedProjects = s.DeniedEnvironments
	default:
		return "1 = 0", nil
	}
	return translated.SQLPredicate(column, "", "")
}

func permissionMatches(policy Policy, resource, action string) bool {
	if policy.Resource != "*" && policy.Resource != resource {
		return false
	}
	if policy.Action == "*" || policy.Action == action {
		return true
	}
	return (action == "view" && policy.Action == "read") ||
		((action == "create" || action == "update" || action == "delete") && policy.Action == "write")
}

func scopeMatches(policy Policy, req Request) bool {
	if policy.Environment != "" && policy.Environment != "*" && policy.Environment != req.Environment {
		return false
	}
	switch policy.ScopeType {
	case ScopePlatform, "":
		return true
	case ScopeProject:
		return policy.ScopeID > 0 && policy.ScopeID == req.ProjectID
	case ScopeService:
		return policy.ScopeID > 0 && policy.ScopeID == req.ServiceID
	case ScopeBuild:
		return policy.ScopeID > 0 && policy.ScopeID == req.BuildID
	case ScopeWorkflow:
		return policy.ScopeID > 0 && policy.ScopeID == req.WorkflowID
	case ScopeCluster:
		return policy.ScopeID > 0 && policy.ScopeID == req.ClusterID
	case ScopeEnvironment:
		return policy.ScopeID > 0 && policy.ScopeID == req.EnvironmentID
	default:
		return false
	}
}

func scopeSlice(scopes *ScopeSet, scopeType string, deny bool) *[]int64 {
	switch scopeType {
	case ScopeProject:
		if deny {
			return &scopes.DeniedProjects
		}
		return &scopes.ProjectIDs
	case ScopeService:
		if deny {
			return &scopes.DeniedServices
		}
		return &scopes.ServiceIDs
	case ScopeBuild:
		if deny {
			return &scopes.DeniedBuilds
		}
		return &scopes.BuildIDs
	case ScopeWorkflow:
		if deny {
			return &scopes.DeniedWorkflows
		}
		return &scopes.WorkflowIDs
	case ScopeEnvironment:
		if deny {
			return &scopes.DeniedEnvironments
		}
		return &scopes.EnvironmentIDs
	default:
		if deny {
			return &scopes.DeniedClusters
		}
		return &scopes.ClusterIDs
	}
}

func appendUnique(values *[]int64, value int64) {
	if value <= 0 {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}
