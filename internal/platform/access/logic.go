package access

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

func (l *Logic) ListBindings(ctx context.Context, operatorID int64, subjectType string, subjectID int64) ([]RoleBinding, int) {
	if code := l.authorize(ctx, operatorID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	bindings, err := l.repo.ListBindings(ctx, subjectType, subjectID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return bindings, errcode.CodeSuccess
}

func (l *Logic) CreateBinding(ctx context.Context, operatorID int64, req *CreateBindingRequest) (int64, int) {
	if code := l.authorize(ctx, operatorID, "create"); code != errcode.CodeSuccess {
		return 0, code
	}
	if !validSubject(req.SubjectType) || !validScope(req.ScopeType, req.ScopeID) || req.SubjectID <= 0 || req.RoleID <= 0 {
		return 0, errcode.CodeParamInvalid
	}
	roleCode, err := l.repo.RoleCode(ctx, req.RoleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errcode.CodeRoleNotFound
		}
		return 0, errcode.CodeDBError
	}
	bindingScope, err := l.repo.RoleBindingScope(ctx, req.RoleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errcode.CodeRoleNotFound
		}
		return 0, errcode.CodeDBError
	}
	// Platform roles are assigned through user_role only. Scoped roles are
	// deliberately required for resource-level bindings to prevent privilege
	// escalation through a platform role.
	if !validRoleBinding(bindingScope, req.ScopeType, roleCode) {
		return 0, errcode.CodeParamInvalid
	}
	environment := strings.TrimSpace(req.Environment)
	if req.ScopeType == authz.ScopeEnvironment && (environment == "" || environment == "*") {
		return 0, errcode.CodeParamInvalid
	}
	if environment == "" {
		environment = "*"
	}
	id, err := l.repo.CreateBinding(ctx, &RoleBinding{
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		RoleID:      req.RoleID,
		ScopeType:   req.ScopeType,
		ScopeID:     req.ScopeID,
		Environment: environment,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   operatorID,
	})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

func (l *Logic) UpdateBinding(ctx context.Context, operatorID, id int64, req *UpdateBindingRequest) int {
	if code := l.authorize(ctx, operatorID, "update"); code != errcode.CodeSuccess {
		return code
	}
	if id <= 0 || (req.Status != nil && *req.Status != 0 && *req.Status != 1) {
		return errcode.CodeParamInvalid
	}
	if code := l.authorizeProtectedBinding(ctx, operatorID, id); code != errcode.CodeSuccess {
		return code
	}
	var expiresAt interface{}
	if req.ClearExpiresAt {
		expiresAt = nil
	} else if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt
	}
	if err := l.repo.UpdateBinding(ctx, id, req.Status, expiresAt, req.ClearExpiresAt); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) DeleteBinding(ctx context.Context, operatorID, id int64) int {
	if code := l.authorize(ctx, operatorID, "delete"); code != errcode.CodeSuccess {
		return code
	}
	if id <= 0 {
		return errcode.CodeParamInvalid
	}
	if code := l.authorizeProtectedBinding(ctx, operatorID, id); code != errcode.CodeSuccess {
		return code
	}
	if err := l.repo.DeleteBinding(ctx, id); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorizeProtectedBinding(ctx context.Context, operatorID, bindingID int64) int {
	roleCode, err := l.repo.BindingRoleCode(ctx, bindingID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errcode.CodeNotFound
		}
		return errcode.CodeDBError
	}
	if roleCode != "super_admin" {
		return errcode.CodeSuccess
	}
	allowed, err := l.authorizer.Authorize(ctx, operatorID, authz.Request{Resource: "system_role", Action: "assign"})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func (l *Logic) CurrentMenus(ctx context.Context, userID int64) ([]Menu, int) {
	menus, err := l.repo.ListMenusForUser(ctx, userID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return menus, errcode.CodeSuccess
}

func (l *Logic) CurrentAccess(ctx context.Context, userID int64) ([]authz.Policy, int) {
	policies, err := l.authorizer.Policies(ctx, userID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return policies, errcode.CodeSuccess
}

func (l *Logic) ListMenus(ctx context.Context, operatorID int64) ([]Menu, int) {
	allowed, err := l.authorizer.Authorize(ctx, operatorID, authz.Request{Resource: "menu", Action: "view"})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	menus, err := l.repo.ListMenus(ctx)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return menus, errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "role_binding", Action: action})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func validSubject(subjectType string) bool {
	return subjectType == "user" || subjectType == "department"
}

func validScope(scopeType string, scopeID int64) bool {
	switch scopeType {
	case authz.ScopePlatform:
		return scopeID == 0
	case authz.ScopeProject, authz.ScopeService, authz.ScopeBuild, authz.ScopeWorkflow, authz.ScopeCluster, authz.ScopeEnvironment:
		return scopeID > 0
	default:
		return false
	}
}

func validRoleBinding(bindingScope, scopeType, roleCode string) bool {
	return bindingScope == "scoped" && scopeType != authz.ScopePlatform && roleCode != "super_admin"
}
