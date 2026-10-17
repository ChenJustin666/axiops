package role

import (
	"context"
	"database/sql"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// Logic 角色业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

// List 角色列表
func (l *Logic) List(ctx context.Context, operatorID int64) ([]Role, int) {
	if code := l.authorize(ctx, operatorID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	roles, err := l.repo.List(ctx)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return roles, errcode.CodeSuccess
}

// GetByID 角色详情（含权限）
func (l *Logic) GetByID(ctx context.Context, operatorID, id int64) (*RoleWithPermissions, int) {
	if code := l.authorize(ctx, operatorID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	role, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if role == nil {
		return nil, errcode.CodeRoleNotFound
	}

	perms, err := l.repo.GetPermissions(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	menuIDs, err := l.repo.GetMenuIDs(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	return &RoleWithPermissions{
		Role:        *role,
		Permissions: perms,
		MenuIDs:     menuIDs,
	}, errcode.CodeSuccess
}

// Create 创建角色
func (l *Logic) Create(ctx context.Context, req *CreateRoleRequest, createdBy int64) (int64, int) {
	if code := l.authorize(ctx, createdBy, "create"); code != errcode.CodeSuccess {
		return 0, code
	}
	bindingScope := req.BindingScope
	if bindingScope == "" {
		bindingScope = "scoped"
	}
	if bindingScope != "scoped" && bindingScope != "platform" {
		return 0, errcode.CodeParamInvalid
	}
	if code := l.validatePermissions(ctx, createdBy, req.Permissions); code != errcode.CodeSuccess {
		return 0, code
	}
	existing, err := l.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if existing != nil {
		return 0, errcode.CodeParamInvalid
	}

	role := &Role{
		Name:         req.Name,
		Code:         req.Code,
		Description:  req.Description,
		Type:         "custom",
		BindingScope: bindingScope,
		CreatedBy:    createdBy,
	}

	id, err := l.repo.CreateWithAssignments(ctx, role, req.Permissions, req.MenuIDs)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	return id, errcode.CodeSuccess
}

// Update 更新角色
func (l *Logic) Update(ctx context.Context, operatorID, id int64, req *UpdateRoleRequest) int {
	if code := l.authorize(ctx, operatorID, "update"); code != errcode.CodeSuccess {
		return code
	}
	role, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if role == nil {
		return errcode.CodeRoleNotFound
	}
	if role.Type == "system" {
		allowed, authErr := l.authorizer.Authorize(ctx, operatorID, authz.Request{Resource: "system_role", Action: "update"})
		if authErr != nil {
			return errcode.CodeDBError
		}
		if !allowed {
			return errcode.CodeNoPermission
		}
	}
	if req.Permissions != nil {
		if code := l.validatePermissions(ctx, operatorID, req.Permissions); code != errcode.CodeSuccess {
			return code
		}
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}

	if req.BindingScope != "" {
		if req.BindingScope != "scoped" && req.BindingScope != "platform" {
			return errcode.CodeParamInvalid
		}
		if req.BindingScope != role.BindingScope {
			compatible, scopeErr := l.repo.CanUseBindingScope(ctx, role.ID, req.BindingScope)
			if scopeErr != nil {
				return errcode.CodeDBError
			}
			if !compatible {
				return errcode.CodeParamInvalid
			}
		}
		role.BindingScope = req.BindingScope
	}

	if err := l.repo.UpdateWithAssignments(ctx, role, req.Permissions, req.Permissions != nil, req.MenuIDs, req.MenuIDs != nil); err != nil {
		return errcode.CodeDBError
	}

	return errcode.CodeSuccess
}

// Delete 删除角色
func (l *Logic) Delete(ctx context.Context, operatorID, id int64) int {
	if code := l.authorize(ctx, operatorID, "delete"); code != errcode.CodeSuccess {
		return code
	}
	role, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if role == nil {
		return errcode.CodeRoleNotFound
	}
	if role.Type == "system" {
		return errcode.CodeNoPermission
	}

	if err := l.repo.Delete(ctx, id); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "role", Action: action})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func (l *Logic) validatePermissions(ctx context.Context, operatorID int64, permissions []PermissionItem) int {
	protected := false
	for _, permission := range permissions {
		if !validPermission(permission) {
			return errcode.CodeParamInvalid
		}
		if permission.Resource == "*" || permission.Resource == "system_role" {
			protected = true
		}
	}
	if !protected {
		return errcode.CodeSuccess
	}
	allowed, err := l.authorizer.Authorize(ctx, operatorID, authz.Request{Resource: "system_role", Action: "update"})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func validPermission(permission PermissionItem) bool {
	if permission.Resource == "" || (permission.Effect != "allow" && permission.Effect != "deny") {
		return false
	}
	switch permission.Action {
	case "*", "view", "create", "update", "delete", "run", "cancel", "deploy", "approve", "rollback", "export", "read", "write", "test", "manage_production", "restart", "manage":
		return true
	default:
		return false
	}
}
