package user

import (
	"context"
	"database/sql"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"golang.org/x/crypto/bcrypt"
)

// Logic 用户业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{
		repo:       NewRepository(db),
		authorizer: authorizer,
	}
}

// List 用户列表
func (l *Logic) List(ctx context.Context, operatorID int64, req *ListUsersRequest) ([]User, int64, int) {
	if code := l.authorize(ctx, operatorID, "view"); code != errcode.CodeSuccess {
		return nil, 0, code
	}
	users, total, err := l.repo.List(ctx, req.Keyword, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	return users, total, errcode.CodeSuccess
}

// GetByID 获取用户详情
func (l *Logic) GetByID(ctx context.Context, operatorID, id int64) (*UserWithRoles, int) {
	if code := l.authorize(ctx, operatorID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	u, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if u == nil {
		return nil, errcode.CodeUserNotFound
	}

	roleIDs, err := l.repo.GetUserRoleIDs(ctx, u.ID)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	return &UserWithRoles{
		User:    *u,
		RoleIDs: roleIDs,
	}, errcode.CodeSuccess
}

// Create 创建用户
func (l *Logic) Create(ctx context.Context, req *CreateUserRequest, createdBy int64) (int64, int) {
	if code := l.authorize(ctx, createdBy, "create"); code != errcode.CodeSuccess {
		return 0, code
	}
	if len(req.RoleIDs) > 0 {
		valid, err := l.repo.ValidatePlatformRoleIDs(ctx, req.RoleIDs)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !valid {
			return 0, errcode.CodeParamInvalid
		}
		if code := l.authorizeSuperAdminAssignment(ctx, createdBy, req.RoleIDs); code != errcode.CodeSuccess {
			return 0, code
		}
	}
	// 检查用户名是否已存在
	existing, err := l.repo.FindByUsername(ctx, req.Username)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if existing != nil {
		return 0, errcode.CodeParamInvalid
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, errcode.CodeInternalError
	}

	u := &User{
		Username:     req.Username,
		Password:     string(hashedPassword),
		Nickname:     req.Nickname,
		Email:        req.Email,
		Phone:        req.Phone,
		Status:       1,
		DepartmentID: req.DepartmentID,
		CreatedBy:    createdBy,
	}

	id, err := l.repo.CreateWithRoles(ctx, u, req.RoleIDs, createdBy)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	return id, errcode.CodeSuccess
}

// Update 更新用户
func (l *Logic) Update(ctx context.Context, operatorID, id int64, req *UpdateUserRequest) int {
	if code := l.authorize(ctx, operatorID, "update"); code != errcode.CodeSuccess {
		return code
	}
	if code := l.authorizeProtectedUser(ctx, operatorID, id); code != errcode.CodeSuccess {
		return code
	}
	u, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if u == nil {
		return errcode.CodeUserNotFound
	}

	if req.Nickname != "" {
		u.Nickname = req.Nickname
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Phone != "" {
		u.Phone = req.Phone
	}
	if req.Avatar != "" {
		u.Avatar = req.Avatar
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	if req.DepartmentID != nil {
		u.DepartmentID = *req.DepartmentID
	}

	if err := l.repo.Update(ctx, u); err != nil {
		return errcode.CodeDBError
	}

	return errcode.CodeSuccess
}

// Delete 删除用户
func (l *Logic) Delete(ctx context.Context, operatorID, id int64) int {
	if code := l.authorize(ctx, operatorID, "delete"); code != errcode.CodeSuccess {
		return code
	}
	if code := l.authorizeProtectedUser(ctx, operatorID, id); code != errcode.CodeSuccess {
		return code
	}
	if err := l.repo.Delete(ctx, id); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// ResetPassword 重置密码
func (l *Logic) ResetPassword(ctx context.Context, operatorID, id int64, newPassword string) int {
	if code := l.authorize(ctx, operatorID, "update"); code != errcode.CodeSuccess {
		return code
	}
	if code := l.authorizeProtectedUser(ctx, operatorID, id); code != errcode.CodeSuccess {
		return code
	}
	return l.updatePassword(ctx, id, newPassword)
}

func (l *Logic) updatePassword(ctx context.Context, id int64, newPassword string) int {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errcode.CodeInternalError
	}
	if err := l.repo.UpdatePassword(ctx, id, string(hashedPassword)); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// UpdatePassword 用户自己修改密码
func (l *Logic) UpdatePassword(ctx context.Context, userID int64, req *UpdatePasswordRequest) int {
	u, err := l.repo.FindByID(ctx, userID)
	if err != nil {
		return errcode.CodeDBError
	}
	if u == nil {
		return errcode.CodeUserNotFound
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.OldPassword)); err != nil {
		return errcode.CodeLoginFailed
	}

	return l.updatePassword(ctx, userID, req.NewPassword)
}

// SetRoles 设置用户角色
func (l *Logic) SetRoles(ctx context.Context, userID int64, roleIDs []int64, operatorID int64) int {
	if code := l.authorize(ctx, operatorID, "update"); code != errcode.CodeSuccess {
		return code
	}
	if code := l.authorizeProtectedUser(ctx, operatorID, userID); code != errcode.CodeSuccess {
		return code
	}
	valid, err := l.repo.ValidatePlatformRoleIDs(ctx, roleIDs)
	if err != nil {
		return errcode.CodeDBError
	}
	if !valid {
		return errcode.CodeParamInvalid
	}
	if code := l.authorizeSuperAdminAssignment(ctx, operatorID, roleIDs); code != errcode.CodeSuccess {
		return code
	}
	if err := l.repo.SetUserRoles(ctx, userID, roleIDs, operatorID); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorizeProtectedUser(ctx context.Context, operatorID, targetUserID int64) int {
	protected, err := l.repo.UserHasRoleCode(ctx, targetUserID, "super_admin")
	if err != nil {
		return errcode.CodeDBError
	}
	if !protected {
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

func (l *Logic) authorizeSuperAdminAssignment(ctx context.Context, operatorID int64, roleIDs []int64) int {
	contains, err := l.repo.ContainsRoleCode(ctx, roleIDs, "super_admin")
	if err != nil {
		return errcode.CodeDBError
	}
	protectedPermissions, err := l.repo.ContainsProtectedRolePermissions(ctx, roleIDs)
	if err != nil {
		return errcode.CodeDBError
	}
	if !contains && !protectedPermissions {
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

func (l *Logic) authorize(ctx context.Context, userID int64, action string) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "user", Action: action})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}
