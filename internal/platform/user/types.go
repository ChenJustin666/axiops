package user

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username     string  `json:"username" validate:"required,min=2,max=64"`
	Password     string  `json:"password" validate:"required,min=6,max=64"`
	Nickname     string  `json:"nickname" validate:"max=64"`
	Email        string  `json:"email" validate:"omitempty,email"`
	Phone        string  `json:"phone" validate:"max=32"`
	DepartmentID int64   `json:"department_id"`
	RoleIDs      []int64 `json:"role_ids"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Avatar       string `json:"avatar"`
	Status       *int8  `json:"status"`
	DepartmentID *int64 `json:"department_id"`
}

// UpdatePasswordRequest 修改密码请求
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=64"`
}

// ResetPasswordRequest 重置密码（管理员）
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=6,max=64"`
}

// SetUserRolesRequest 设置用户角色
type SetUserRolesRequest struct {
	RoleIDs []int64 `json:"role_ids" validate:"required"`
}

// ListUsersRequest 用户列表查询
type ListUsersRequest struct {
	Keyword  string `form:"keyword"`
	Status   int    `form:"status,default=-1"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}
