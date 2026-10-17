package authz

import (
	"context"
	"strings"
)

type Action string

const (
	ActionUnknown  Action = ""
	ActionView     Action = "view"
	ActionRead     Action = "read"
	ActionCreate   Action = "create"
	ActionUpdate   Action = "update"
	ActionDelete   Action = "delete"
	ActionDeploy   Action = "deploy"
	ActionApprove  Action = "approve"
	ActionRollback Action = "rollback"
	ActionExecute  Action = "execute"
	ActionTerminal Action = "terminal"
	ActionSSH      Action = "ssh"
	ActionSQLQuery Action = "sql_query"
	ActionSQLExec  Action = "sql_exec"
	ActionManage   Action = "manage"
)

type ResourceType string

const (
	ResourceTypeAPI         ResourceType = "api"
	ResourceTypeProject     ResourceType = "project"
	ResourceTypeEnvironment ResourceType = "environment"
	ResourceTypeCluster     ResourceType = "cluster"
	ResourceTypeNamespace   ResourceType = "namespace"
	ResourceTypeHost        ResourceType = "host"
	ResourceTypeDatabase    ResourceType = "database"
	ResourceTypePipeline    ResourceType = "pipeline"
	ResourceTypeRelease     ResourceType = "release"
	ResourceTypeArtifact    ResourceType = "artifact"
	ResourceTypeAlert       ResourceType = "alert"
	ResourceTypeLog         ResourceType = "log"
	ResourceTypeTicket      ResourceType = "ticket"
	ResourceTypeAI          ResourceType = "ai"
)

type ScopeKind string

const (
	ScopeKindGlobal      ScopeKind = "global"
	ScopeKindOrg         ScopeKind = "org"
	ScopeKindDept        ScopeKind = "dept"
	ScopeKindProject     ScopeKind = "project"
	ScopeKindEnvironment ScopeKind = "environment"
	ScopeKindCluster     ScopeKind = "cluster"
	ScopeKindNamespace   ScopeKind = "namespace"
	ScopeKindHost        ScopeKind = "host"
	ScopeKindDatabase    ScopeKind = "database"
)

type ResourceRef struct {
	Type          ResourceType      `json:"type,omitempty"`
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name,omitempty"`
	Path          string            `json:"path,omitempty"`
	Method        string            `json:"method,omitempty"`
	OrgID         uint64            `json:"orgId,omitempty"`
	DeptID        uint64            `json:"deptId,omitempty"`
	ProjectID     uint64            `json:"projectId,omitempty"`
	EnvironmentID uint64            `json:"environmentId,omitempty"`
	ClusterID     uint64            `json:"clusterId,omitempty"`
	Namespace     string            `json:"namespace,omitempty"`
	HostID        uint64            `json:"hostId,omitempty"`
	DatabaseID    uint64            `json:"databaseId,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

type Scope struct {
	Kind           ScopeKind `json:"kind,omitempty"`
	OrgIDs         []uint64  `json:"orgIds,omitempty"`
	DeptIDs        []uint64  `json:"deptIds,omitempty"`
	ProjectIDs     []uint64  `json:"projectIds,omitempty"`
	EnvironmentIDs []uint64  `json:"environmentIds,omitempty"`
	ClusterIDs     []uint64  `json:"clusterIds,omitempty"`
	Namespaces     []string  `json:"namespaces,omitempty"`
	HostIDs        []uint64  `json:"hostIds,omitempty"`
	DatabaseIDs    []uint64  `json:"databaseIds,omitempty"`
}

func (s Scope) HasAnyRestriction() bool {
	return len(s.OrgIDs) > 0 ||
		len(s.DeptIDs) > 0 ||
		len(s.ProjectIDs) > 0 ||
		len(s.EnvironmentIDs) > 0 ||
		len(s.ClusterIDs) > 0 ||
		len(s.Namespaces) > 0 ||
		len(s.HostIDs) > 0 ||
		len(s.DatabaseIDs) > 0
}

type AuthorizeRequest struct {
	SubjectUserID uint64      `json:"subjectUserId,omitempty"`
	SubjectRoles  []string    `json:"subjectRoles,omitempty"`
	Action        Action      `json:"action,omitempty"`
	Resource      ResourceRef `json:"resource,omitempty"`
	Scope         Scope       `json:"scope,omitempty"`
	Environment   string      `json:"environment,omitempty"`
}

type Authorizer interface {
	Authorize(ctx context.Context, req AuthorizeRequest) (bool, error)
}

func (a Action) IsReadOnly() bool {
	switch a {
	case ActionView, ActionRead, ActionSQLQuery:
		return true
	default:
		return false
	}
}

func (a Action) IsHighRisk() bool {
	switch a {
	case ActionDeploy, ActionApprove, ActionRollback, ActionExecute, ActionTerminal, ActionSSH, ActionSQLExec, ActionManage:
		return true
	default:
		return false
	}
}

func NewAPIAuthorizeRequest(roles []string, path, method string) AuthorizeRequest {
	return AuthorizeRequest{
		SubjectRoles: roles,
		Action:       actionFromMethod(method),
		Resource: ResourceRef{
			Type:   ResourceTypeAPI,
			Path:   path,
			Method: method,
		},
	}
}

func NewResourceAuthorizeRequest(roles []string, userID uint64, action Action, resource ResourceRef, scope Scope, environment string) AuthorizeRequest {
	return AuthorizeRequest{
		SubjectUserID: userID,
		SubjectRoles:  roles,
		Action:        action,
		Resource:      resource,
		Scope:         scope,
		Environment:   environment,
	}
}

func NewProjectAuthorizeRequest(roles []string, userID uint64, action Action, projectID uint64) AuthorizeRequest {
	return NewResourceAuthorizeRequest(roles, userID, action, ResourceRef{
		Type:      ResourceTypeProject,
		ProjectID: projectID,
	}, Scope{
		Kind:       ScopeKindProject,
		ProjectIDs: singleUint64(projectID),
	}, "")
}

func NewEnvironmentAuthorizeRequest(roles []string, userID uint64, action Action, projectID, environmentID uint64, environment string) AuthorizeRequest {
	return NewResourceAuthorizeRequest(roles, userID, action, ResourceRef{
		Type:          ResourceTypeEnvironment,
		ProjectID:     projectID,
		EnvironmentID: environmentID,
	}, Scope{
		Kind:           ScopeKindEnvironment,
		ProjectIDs:     singleUint64(projectID),
		EnvironmentIDs: singleUint64(environmentID),
	}, environment)
}

func NewClusterAuthorizeRequest(roles []string, userID uint64, action Action, projectID, clusterID uint64, namespace string) AuthorizeRequest {
	return NewResourceAuthorizeRequest(roles, userID, action, ResourceRef{
		Type:      ResourceTypeCluster,
		ProjectID: projectID,
		ClusterID: clusterID,
		Namespace: namespace,
	}, Scope{
		Kind:       ScopeKindCluster,
		ProjectIDs: singleUint64(projectID),
		ClusterIDs: singleUint64(clusterID),
		Namespaces: singleString(namespace),
	}, "")
}

func NewHostAuthorizeRequest(roles []string, userID uint64, action Action, projectID uint64, hostID uint64) AuthorizeRequest {
	return NewResourceAuthorizeRequest(roles, userID, action, ResourceRef{
		Type:      ResourceTypeHost,
		ProjectID: projectID,
		HostID:    hostID,
	}, Scope{
		Kind:       ScopeKindHost,
		ProjectIDs: singleUint64(projectID),
		HostIDs:    singleUint64(hostID),
	}, "")
}

func NewDatabaseAuthorizeRequest(roles []string, userID uint64, action Action, projectID uint64, databaseID uint64) AuthorizeRequest {
	return NewResourceAuthorizeRequest(roles, userID, action, ResourceRef{
		Type:       ResourceTypeDatabase,
		ProjectID:  projectID,
		DatabaseID: databaseID,
	}, Scope{
		Kind:        ScopeKindDatabase,
		ProjectIDs:  singleUint64(projectID),
		DatabaseIDs: singleUint64(databaseID),
	}, "")
}

func singleUint64(value uint64) []uint64 {
	if value == 0 {
		return nil
	}
	return []uint64{value}
}

func singleString(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}

func actionFromMethod(method string) Action {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD", "OPTIONS":
		return ActionRead
	case "POST":
		return ActionCreate
	case "PUT", "PATCH":
		return ActionUpdate
	case "DELETE":
		return ActionDelete
	default:
		return ActionUnknown
	}
}

func methodFromAction(action Action) string {
	switch action {
	case ActionRead, ActionView:
		return "GET"
	case ActionCreate:
		return "POST"
	case ActionUpdate:
		return "PUT"
	case ActionDelete:
		return "DELETE"
	case ActionDeploy, ActionApprove, ActionRollback, ActionExecute, ActionTerminal, ActionSSH, ActionSQLExec, ActionManage:
		return "POST"
	default:
		return ""
	}
}

func (s Scope) AllowsResource(resource ResourceRef) bool {
	if s.Kind == ScopeKindGlobal {
		return true
	}
	if !s.HasAnyRestriction() {
		return false
	}
	if len(s.OrgIDs) > 0 && (resource.OrgID == 0 || !containsUint64(s.OrgIDs, resource.OrgID)) {
		return false
	}
	if len(s.DeptIDs) > 0 && (resource.DeptID == 0 || !containsUint64(s.DeptIDs, resource.DeptID)) {
		return false
	}
	if len(s.ProjectIDs) > 0 && (resource.ProjectID == 0 || !containsUint64(s.ProjectIDs, resource.ProjectID)) {
		return false
	}
	if len(s.EnvironmentIDs) > 0 && (resource.EnvironmentID == 0 || !containsUint64(s.EnvironmentIDs, resource.EnvironmentID)) {
		return false
	}
	if len(s.ClusterIDs) > 0 && (resource.ClusterID == 0 || !containsUint64(s.ClusterIDs, resource.ClusterID)) {
		return false
	}
	if len(s.Namespaces) > 0 && (resource.Namespace == "" || !containsStringFold(s.Namespaces, resource.Namespace)) {
		return false
	}
	if len(s.HostIDs) > 0 && (resource.HostID == 0 || !containsUint64(s.HostIDs, resource.HostID)) {
		return false
	}
	if len(s.DatabaseIDs) > 0 && (resource.DatabaseID == 0 || !containsUint64(s.DatabaseIDs, resource.DatabaseID)) {
		return false
	}
	return true
}

func containsUint64(values []uint64, want uint64) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsStringFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}
