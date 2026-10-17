package authz

import (
	"context"
	"database/sql"
	"testing"

	"github.com/casbin/casbin/v2"
	managerservice "github.com/ChenJustin666/axiops/application/manager-rpc/client/managerservice"
	model2 "github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc"
)

func TestAuthorizeAPIPathCompatibility(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("viewer", "/manager/v1/cluster/*", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	manager := &CasbinRBACManager{enforcer: enforcer}
	allowed, err := manager.Authorize(context.Background(), NewAPIAuthorizeRequest([]string{"viewer"}, "/manager/v1/cluster/1", "GET"))
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed")
	}
}

func TestAuthorizeResourceWithoutScopeDenied(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{enforcer: enforcer}
	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectRoles: []string{"viewer"},
		Action:       ActionDeploy,
		Resource: ResourceRef{
			Type:      ResourceTypeProject,
			ProjectID: 1,
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if allowed {
		t.Fatal("expected deny")
	}
}

func TestAuthorizeProjectScopeChecksMembership(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{9}},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionRead,
		Resource: ResourceRef{
			Type:      ResourceTypeProject,
			ProjectID: 9,
		},
		Scope: Scope{
			Kind:       ScopeKindProject,
			ProjectIDs: []uint64{9},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed")
	}
}

func TestAuthorizeResourceWriteRequiresAPIPath(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{9}},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionDeploy,
		Resource: ResourceRef{
			Type:      ResourceTypeProject,
			ProjectID: 9,
		},
		Scope: Scope{
			Kind:       ScopeKindProject,
			ProjectIDs: []uint64{9},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if allowed {
		t.Fatal("expected deny")
	}
}

func TestAuthorizeResourceRequiresPathAndScope(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("developer", "/project/9/deploy", "POST"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{9}},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionDeploy,
		Resource: ResourceRef{
			Type:      ResourceTypeProject,
			ProjectID: 9,
			Path:      "/project/9/deploy",
			Method:    "POST",
		},
		Scope: Scope{
			Kind:       ScopeKindProject,
			ProjectIDs: []uint64{9},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed")
	}
}

func TestAuthorizeClusterScopeResolvesProjectAndNamespace(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{9}},
		resourceLookup: fakeResourceLookup{
			clusterProjects: map[uint64]uint64{31: 9},
			namespaces:      map[uint64]map[string]bool{31: {"prod": true}},
		},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionRead,
		Resource: ResourceRef{
			Type:      ResourceTypeCluster,
			ClusterID: 31,
			Namespace: "prod",
		},
		Scope: Scope{
			Kind:       ScopeKindCluster,
			ClusterIDs: []uint64{31},
			Namespaces: []string{"prod"},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed")
	}
}

func TestAuthorizeClusterScopeRejectsMismatchedProject(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{8, 9}},
		resourceLookup: fakeResourceLookup{
			clusterProjects: map[uint64]uint64{31: 9},
			namespaces:      map[uint64]map[string]bool{31: {"prod": true}},
		},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionRead,
		Resource: ResourceRef{
			Type:      ResourceTypeCluster,
			ProjectID: 8,
			ClusterID: 31,
			Namespace: "prod",
		},
		Scope: Scope{
			Kind:       ScopeKindCluster,
			ProjectIDs: []uint64{8},
			ClusterIDs: []uint64{31},
			Namespaces: []string{"prod"},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if allowed {
		t.Fatal("expected deny")
	}
}

func TestAuthorizeNamespaceScopeRequiresExistingWorkspace(t *testing.T) {
	enforcer, err := casbin.NewSyncedEnforcer(newRBACModel())
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		projectMemberBinding: fakeProjectMemberBindingModel{projectIDs: []uint64{9}},
		resourceLookup: fakeResourceLookup{
			clusterProjects: map[uint64]uint64{31: 9},
			namespaces:      map[uint64]map[string]bool{31: {"prod": true}},
		},
	}

	allowed, err := manager.Authorize(context.Background(), AuthorizeRequest{
		SubjectUserID: 100,
		SubjectRoles:  []string{"developer"},
		Action:        ActionRead,
		Resource: ResourceRef{
			Type:      ResourceTypeNamespace,
			ProjectID: 9,
			ClusterID: 31,
			Namespace: "missing",
		},
		Scope: Scope{
			Kind:       ScopeKindNamespace,
			ProjectIDs: []uint64{9},
			ClusterIDs: []uint64{31},
			Namespaces: []string{"missing"},
		},
	})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if allowed {
		t.Fatal("expected deny")
	}
}

func TestScopeAllowsResource(t *testing.T) {
	tests := []struct {
		name     string
		scope    Scope
		resource ResourceRef
		want     bool
	}{
		{
			name:  "global allows all",
			scope: Scope{Kind: ScopeKindGlobal},
			resource: ResourceRef{
				Type:          ResourceTypeProject,
				ProjectID:     1,
				EnvironmentID: 2,
			},
			want: true,
		},
		{
			name: "project scope matches",
			scope: Scope{
				Kind:       ScopeKindProject,
				ProjectIDs: []uint64{11},
			},
			resource: ResourceRef{
				Type:      ResourceTypeProject,
				ProjectID: 11,
			},
			want: true,
		},
		{
			name: "environment scope matches",
			scope: Scope{
				Kind:           ScopeKindEnvironment,
				ProjectIDs:     []uint64{11},
				EnvironmentIDs: []uint64{21},
			},
			resource: ResourceRef{
				Type:          ResourceTypeEnvironment,
				ProjectID:     11,
				EnvironmentID: 21,
			},
			want: true,
		},
		{
			name: "cluster scope matches",
			scope: Scope{
				Kind:       ScopeKindCluster,
				ProjectIDs: []uint64{11},
				ClusterIDs: []uint64{31},
				Namespaces: []string{"prod"},
			},
			resource: ResourceRef{
				Type:      ResourceTypeCluster,
				ProjectID: 11,
				ClusterID: 31,
				Namespace: "prod",
			},
			want: true,
		},
		{
			name: "host scope matches",
			scope: Scope{
				Kind:    ScopeKindHost,
				HostIDs: []uint64{41},
			},
			resource: ResourceRef{
				Type:   ResourceTypeHost,
				HostID: 41,
			},
			want: true,
		},
		{
			name: "database scope matches",
			scope: Scope{
				Kind:        ScopeKindDatabase,
				DatabaseIDs: []uint64{51},
			},
			resource: ResourceRef{
				Type:       ResourceTypeDatabase,
				DatabaseID: 51,
			},
			want: true,
		},
		{
			name: "project scope mismatch",
			scope: Scope{
				Kind:       ScopeKindProject,
				ProjectIDs: []uint64{11},
			},
			resource: ResourceRef{
				Type:      ResourceTypeProject,
				ProjectID: 12,
			},
			want: false,
		},
		{
			name:  "empty scope denies",
			scope: Scope{},
			resource: ResourceRef{
				Type:      ResourceTypeProject,
				ProjectID: 11,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scope.AllowsResource(tt.resource)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionClassification(t *testing.T) {
	if !ActionRead.IsReadOnly() || !ActionView.IsReadOnly() || !ActionSQLQuery.IsReadOnly() {
		t.Fatal("expected read, view and sql_query to be read-only")
	}
	if ActionDeploy.IsReadOnly() || ActionSQLExec.IsReadOnly() || ActionSSH.IsReadOnly() {
		t.Fatal("expected deploy, sql_exec and ssh to be write/high-risk actions")
	}
	if !ActionDeploy.IsHighRisk() || !ActionApprove.IsHighRisk() || !ActionRollback.IsHighRisk() ||
		!ActionTerminal.IsHighRisk() || !ActionSSH.IsHighRisk() || !ActionSQLExec.IsHighRisk() {
		t.Fatal("expected production operations to be high-risk")
	}
	if ActionRead.IsHighRisk() || ActionSQLQuery.IsHighRisk() {
		t.Fatal("read-only actions must not be high-risk")
	}
}

type fakeProjectMemberBindingModel struct {
	projectIDs []uint64
}

type fakeResourceLookup struct {
	clusterProjects map[uint64]uint64
	namespaces      map[uint64]map[string]bool
}

func (f fakeResourceLookup) ProjectClusterGetById(_ context.Context, in *managerservice.GetOnecProjectClusterByIdReq, _ ...grpc.CallOption) (*managerservice.GetOnecProjectClusterByIdResp, error) {
	if in == nil {
		return &managerservice.GetOnecProjectClusterByIdResp{}, nil
	}
	projectID, ok := f.clusterProjects[in.Id]
	if !ok {
		return &managerservice.GetOnecProjectClusterByIdResp{}, nil
	}
	return &managerservice.GetOnecProjectClusterByIdResp{
		Data: &managerservice.OnecProjectCluster{Id: in.Id, ProjectId: projectID},
	}, nil
}

func (f fakeResourceLookup) ProjectWorkspaceSearch(_ context.Context, in *managerservice.SearchOnecProjectWorkspaceReq, _ ...grpc.CallOption) (*managerservice.SearchOnecProjectWorkspaceResp, error) {
	if in == nil || !f.namespaces[in.ProjectClusterId][in.Namespace] {
		return &managerservice.SearchOnecProjectWorkspaceResp{}, nil
	}
	return &managerservice.SearchOnecProjectWorkspaceResp{
		Data: []*managerservice.OnecProjectWorkspace{{ProjectClusterId: in.ProjectClusterId, Namespace: in.Namespace}},
	}, nil
}

func (f fakeProjectMemberBindingModel) Insert(context.Context, *model2.ProjectMemberBinding) (sql.Result, error) {
	return nil, nil
}

func (f fakeProjectMemberBindingModel) FindOne(context.Context, uint64) (*model2.ProjectMemberBinding, error) {
	return nil, sqlx.ErrNotFound
}

func (f fakeProjectMemberBindingModel) Search(context.Context, string, bool, uint64, uint64, string, ...any) ([]*model2.ProjectMemberBinding, uint64, error) {
	return nil, 0, nil
}

func (f fakeProjectMemberBindingModel) SearchNoPage(context.Context, string, bool, string, ...any) ([]*model2.ProjectMemberBinding, error) {
	return nil, nil
}

func (f fakeProjectMemberBindingModel) FindOneByProjectIdUserId(context.Context, uint64, uint64) (*model2.ProjectMemberBinding, error) {
	return nil, sqlx.ErrNotFound
}

func (f fakeProjectMemberBindingModel) Update(context.Context, *model2.ProjectMemberBinding) error {
	return nil
}

func (f fakeProjectMemberBindingModel) Delete(context.Context, uint64) error {
	return nil
}

func (f fakeProjectMemberBindingModel) DeleteSoft(context.Context, uint64) error {
	return nil
}

func (f fakeProjectMemberBindingModel) TransCtx(context.Context, func(context.Context, sqlx.Session) error) error {
	return nil
}

func (f fakeProjectMemberBindingModel) TransOnSql(context.Context, sqlx.Session, uint64, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (f fakeProjectMemberBindingModel) ExecSql(context.Context, uint64, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (f fakeProjectMemberBindingModel) SetProjectMembers(context.Context, uint64, []*model2.ProjectMemberBinding, string) error {
	return nil
}

func (f fakeProjectMemberBindingModel) ListByProject(context.Context, uint64) ([]*model2.ProjectMemberBindingWithUser, error) {
	return nil, nil
}

func (f fakeProjectMemberBindingModel) ListProjectIdsByUser(context.Context, uint64) ([]uint64, error) {
	return append([]uint64(nil), f.projectIDs...), nil
}

func (f fakeProjectMemberBindingModel) ListUserIdsByPlatform(context.Context, uint64, uint64, uint64) ([]uint64, uint64, error) {
	return nil, 0, nil
}
