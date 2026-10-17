package authz

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	managerservice "github.com/ChenJustin666/axiops/application/manager-rpc/client/managerservice"
	model2 "github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
)

type whiteListRule struct {
	path   string
	method string
}

var whiteList = []whiteListRule{
	{path: "/portal/v1/user/info", method: "GET"},
	{path: "/portal/v1/auth/logout", method: "POST"},
	{path: "/portal/v1/auth/login", method: "POST"},
	{path: "/portal/v1/menu/roles/tree", method: "GET"},
	{path: "/ws/v1/site-messages/connect", method: "GET"},
	{path: "/portal/v1/platform/user/platforms", method: "GET"},
}

const (
	SuperAdminRole = "SUPER_ADMIN"
)

type resourceLookup interface {
	ProjectClusterGetById(ctx context.Context, in *managerservice.GetOnecProjectClusterByIdReq, opts ...grpc.CallOption) (*managerservice.GetOnecProjectClusterByIdResp, error)
	ProjectWorkspaceSearch(ctx context.Context, in *managerservice.SearchOnecProjectWorkspaceReq, opts ...grpc.CallOption) (*managerservice.SearchOnecProjectWorkspaceResp, error)
}

type CasbinRBACManager struct {
	enforcer             *casbin.SyncedEnforcer
	adapter              *CasbinAdapter
	watcher              *RedisWatcher
	sysRole              model2.SysRoleModel
	sysApi               model2.SysApiModel
	projectMemberBinding model2.ProjectMemberBindingModel
	resourceLookup       resourceLookup
	reloadMutex          sync.Mutex
}

func NewCasbinRBACManager(
	redisConf redis.RedisConf,
	sysRole model2.SysRoleModel,
	sysApi model2.SysApiModel,
	sysRoleApi model2.SysRoleApiModel,
	projectMemberBinding model2.ProjectMemberBindingModel,
	resourceLookup resourceLookup,
) (*CasbinRBACManager, error) {
	m := newRBACModel()
	adapter := NewCasbinAdapter(sysRole, sysApi, sysRoleApi)

	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("create enforcer failed: %w", err)
	}

	watcher, err := NewRedisWatcher(redisConf)
	if err != nil {
		return nil, fmt.Errorf("create watcher failed: %w", err)
	}

	if err := enforcer.SetWatcher(watcher); err != nil {
		return nil, fmt.Errorf("set watcher failed: %w", err)
	}

	manager := &CasbinRBACManager{
		enforcer:             enforcer,
		adapter:              adapter,
		watcher:              watcher,
		sysRole:              sysRole,
		sysApi:               sysApi,
		projectMemberBinding: projectMemberBinding,
		resourceLookup:       resourceLookup,
	}

	if err := watcher.SetUpdateCallback(func(msg string) {
		logx.Infof("[RBAC] received policy update: %s", msg)
		if err := manager.reloadPolicyInternal(); err != nil {
			logx.Errorf("[RBAC] reload policy failed: %v", err)
		}
	}); err != nil {
		return nil, fmt.Errorf("set watcher callback failed: %w", err)
	}

	if err := watcher.Start(); err != nil {
		return nil, fmt.Errorf("start watcher failed: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("load policy failed: %w", err)
	}

	logx.Info("[RBAC] casbin RBAC manager initialized")
	return manager, nil
}

func (m *CasbinRBACManager) Authorize(ctx context.Context, req AuthorizeRequest) (bool, error) {
	path := strings.TrimSpace(req.Resource.Path)
	method := strings.ToUpper(strings.TrimSpace(req.Resource.Method))
	if method == "" {
		method = methodFromAction(req.Action)
	}
	if path != "" {
		allowed, err := m.authorizeByPath(ctx, req.SubjectRoles, path, method)
		if err != nil || !allowed || req.Resource.Type == ResourceTypeAPI {
			return allowed, err
		}
	}

	if path == "" && !req.Action.IsReadOnly() {
		logx.WithContext(ctx).Infof("[RBAC] deny resource write without API permission: type=%s action=%s", req.Resource.Type, req.Action)
		return false, nil
	}

	if len(req.SubjectRoles) == 0 {
		logx.WithContext(ctx).Infof("[RBAC] deny resource access without roles: type=%s action=%s", req.Resource.Type, req.Action)
		return false, nil
	}

	for _, role := range req.SubjectRoles {
		if strings.EqualFold(strings.ToUpper(role), SuperAdminRole) {
			logx.WithContext(ctx).Infof("[RBAC] super admin allow resource access: role=%s type=%s action=%s", role, req.Resource.Type, req.Action)
			return true, nil
		}
	}

	allowed, err := m.authorizeByResourceScope(ctx, req)
	if err != nil {
		return false, err
	}
	if allowed {
		return true, nil
	}

	logx.WithContext(ctx).Infof("[RBAC] deny resource access: type=%s action=%s scope=%+v roles=%v userId=%d resource=%+v",
		req.Resource.Type, req.Action, req.Scope, req.SubjectRoles, req.SubjectUserID, req.Resource)
	return false, nil
}

func (m *CasbinRBACManager) authorizeByPath(ctx context.Context, userRoles []string, path string, method string) (bool, error) {
	if m.isInWhiteList(path, method) {
		logx.WithContext(ctx).Infof("[RBAC] allow whitelist request: %s %s", method, path)
		return true, nil
	}

	if len(userRoles) == 0 {
		logx.WithContext(ctx).Infof("[RBAC] deny path access without roles: %s %s", method, path)
		return false, nil
	}

	for _, role := range userRoles {
		if strings.EqualFold(strings.ToUpper(role), SuperAdminRole) {
			logx.WithContext(ctx).Infof("[RBAC] super admin allow path access: %s %s", method, path)
			return true, nil
		}
	}

	for _, role := range userRoles {
		allowed, err := m.enforcer.Enforce(role, path, method)
		if err != nil {
			logx.WithContext(ctx).Errorf("[RBAC] casbin enforce failed: role=%s path=%s method=%s error=%v", role, path, method, err)
			continue
		}
		if allowed {
			logx.WithContext(ctx).Infof("[RBAC] allow path access: role=%s %s %s", role, method, path)
			return true, nil
		}
	}

	logx.WithContext(ctx).Infof("[RBAC] deny path access: roles=%v %s %s", userRoles, method, path)
	return false, nil
}

func (m *CasbinRBACManager) CheckPermission(ctx context.Context, userRoles []string, path string, method string) (bool, error) {
	return m.Authorize(ctx, NewAPIAuthorizeRequest(userRoles, path, method))
}

func (m *CasbinRBACManager) authorizeByResourceScope(ctx context.Context, req AuthorizeRequest) (bool, error) {
	if req.Resource.Type == ResourceTypeAPI {
		return false, nil
	}
	if !req.Scope.AllowsResource(req.Resource) {
		return false, nil
	}

	projectID, err := m.resolveProjectIDForResource(ctx, req)
	if err != nil {
		return false, err
	}
	if projectID == 0 {
		return false, nil
	}

	if req.SubjectUserID == 0 || m.projectMemberBinding == nil {
		return false, nil
	}

	projectIDs, err := m.projectMemberBinding.ListProjectIdsByUser(ctx, req.SubjectUserID)
	if err != nil {
		logx.WithContext(ctx).Errorf("[RBAC] query user project scope failed: userId=%d error=%v", req.SubjectUserID, err)
		return false, err
	}
	if !containsUint64(projectIDs, projectID) {
		return false, nil
	}
	return true, nil
}

func (m *CasbinRBACManager) resolveProjectIDForResource(ctx context.Context, req AuthorizeRequest) (uint64, error) {
	if req.Resource.ProjectID > 0 {
		switch req.Resource.Type {
		case ResourceTypeCluster:
			resolved, err := m.resolveProjectIDFromCluster(ctx, req.Resource.ClusterID)
			if err != nil {
				return 0, err
			}
			if resolved == 0 || resolved != req.Resource.ProjectID {
				return 0, nil
			}
			if strings.TrimSpace(req.Resource.Namespace) != "" {
				resolved, err = m.resolveProjectIDFromNamespace(ctx, req.Resource.ClusterID, req.Resource.Namespace)
				if err != nil {
					return 0, err
				}
				if resolved == 0 || resolved != req.Resource.ProjectID {
					return 0, nil
				}
			}
		case ResourceTypeNamespace:
			resolved, err := m.resolveProjectIDFromNamespace(ctx, req.Resource.ClusterID, req.Resource.Namespace)
			if err != nil {
				return 0, err
			}
			if resolved == 0 || resolved != req.Resource.ProjectID {
				return 0, nil
			}
		}
		return req.Resource.ProjectID, nil
	}

	switch req.Resource.Type {
	case ResourceTypeCluster:
		return m.resolveProjectIDFromCluster(ctx, req.Resource.ClusterID)
	case ResourceTypeNamespace:
		return m.resolveProjectIDFromNamespace(ctx, req.Resource.ClusterID, req.Resource.Namespace)
	default:
		return 0, nil
	}
}

func (m *CasbinRBACManager) resolveProjectIDFromCluster(ctx context.Context, clusterID uint64) (uint64, error) {
	if clusterID == 0 || m.resourceLookup == nil {
		return 0, nil
	}
	resp, err := m.resourceLookup.ProjectClusterGetById(ctx, &managerservice.GetOnecProjectClusterByIdReq{Id: clusterID})
	if err != nil {
		return 0, err
	}
	if resp == nil || resp.GetData() == nil {
		return 0, nil
	}
	return resp.GetData().GetProjectId(), nil
}

func (m *CasbinRBACManager) resolveProjectIDFromNamespace(ctx context.Context, clusterID uint64, namespace string) (uint64, error) {
	if clusterID == 0 || strings.TrimSpace(namespace) == "" || m.resourceLookup == nil {
		return 0, nil
	}

	wsResp, err := m.resourceLookup.ProjectWorkspaceSearch(ctx, &managerservice.SearchOnecProjectWorkspaceReq{
		ProjectClusterId: clusterID,
		Namespace:        namespace,
	})
	if err != nil {
		return 0, err
	}
	if wsResp == nil || len(wsResp.GetData()) == 0 {
		return 0, nil
	}
	return m.resolveProjectIDFromCluster(ctx, clusterID)
}

func (m *CasbinRBACManager) reloadPolicyInternal() error {
	m.reloadMutex.Lock()
	defer m.reloadMutex.Unlock()
	return m.enforcer.LoadPolicy()
}

func (m *CasbinRBACManager) ReloadPolicy(ctx context.Context) error {
	logx.WithContext(ctx).Info("[RBAC] reloading policy")

	m.reloadMutex.Lock()
	defer m.reloadMutex.Unlock()

	if err := m.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("reload policy failed: %w", err)
	}
	if err := m.watcher.Update(); err != nil {
		logx.WithContext(ctx).Errorf("[RBAC] notify peers failed: %v", err)
	}

	logx.WithContext(ctx).Info("[RBAC] reload completed")
	return nil
}

func (m *CasbinRBACManager) GetAllPolicies() ([][]string, error) {
	return m.enforcer.GetPolicy()
}

func (m *CasbinRBACManager) GetRolePolicies(roleCode string) ([][]string, error) {
	return m.enforcer.GetFilteredPolicy(0, roleCode)
}

func (m *CasbinRBACManager) GetUserPermissions(ctx context.Context, userRoles []string) ([]Permission, error) {
	for _, role := range userRoles {
		if strings.EqualFold(role, SuperAdminRole) {
			return m.getAllPermissions(ctx)
		}
	}

	permMap := make(map[string]Permission)
	for _, role := range userRoles {
		policies, err := m.enforcer.GetFilteredPolicy(0, role)
		if err != nil {
			logx.WithContext(ctx).Errorf("[RBAC] get role policies failed: role=%s err=%v", role, err)
			continue
		}
		for _, policy := range policies {
			if len(policy) < 3 {
				continue
			}
			path := policy[1]
			method := policy[2]
			permMap[fmt.Sprintf("%s:%s", path, method)] = Permission{Path: path, Method: method}
		}
	}

	result := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		result = append(result, perm)
	}

	logx.WithContext(ctx).Infof("[RBAC] role permissions count=%d", len(result))
	return result, nil
}

func (m *CasbinRBACManager) getAllPermissions(ctx context.Context) ([]Permission, error) {
	apis, err := m.sysApi.SearchNoPage(ctx, "", true, "is_permission = ?", 1)
	if err != nil {
		return nil, fmt.Errorf("query permission apis failed: %w", err)
	}

	result := make([]Permission, 0, len(apis))
	for _, api := range apis {
		result = append(result, Permission{Path: api.Path, Method: api.Method})
	}
	return result, nil
}

func (m *CasbinRBACManager) BatchCheckPermission(
	ctx context.Context,
	userRoles []string,
	permissions []Permission,
) (map[string]bool, error) {
	result := make(map[string]bool, len(permissions))
	for _, perm := range permissions {
		key := fmt.Sprintf("%s:%s", perm.Path, perm.Method)
		allowed, err := m.CheckPermission(ctx, userRoles, perm.Path, perm.Method)
		if err != nil {
			logx.WithContext(ctx).Errorf("[RBAC] check permission failed: %s err=%v", key, err)
			result[key] = false
			continue
		}
		result[key] = allowed
	}
	return result, nil
}

func (m *CasbinRBACManager) Close() {
	if m.watcher != nil {
		m.watcher.Close()
	}
	logx.Info("[RBAC] RBAC manager closed")
}

type Permission struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (m *CasbinRBACManager) isInWhiteList(path string, method string) bool {
	for _, rule := range whiteList {
		if strings.EqualFold(method, rule.method) && path == rule.path {
			return true
		}
	}
	return false
}

func newRBACModel() model.Model {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "r.sub == p.sub && (keyMatch3(r.obj, p.obj) || keyMatch2(r.obj, p.obj)) && (r.act == p.act || p.act == \"*\")")
	return m
}
