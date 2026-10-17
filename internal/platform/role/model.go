package role

import "time"

// Role 角色模型
type Role struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Description  string    `json:"description"`
	Type         string    `json:"type"`
	BindingScope string    `json:"binding_scope"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RolePermission 角色权限策略
type RolePermission struct {
	ID        int64     `json:"id"`
	RoleID    int64     `json:"role_id"`
	Resource  string    `json:"resource"`
	Action    string    `json:"action"`
	Effect    string    `json:"effect"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleWithPermissions 带权限的角色
type RoleWithPermissions struct {
	Role
	Permissions []RolePermission `json:"permissions"`
	MenuIDs     []int64          `json:"menu_ids"`
}

// CreateRoleRequest 创建角色
type CreateRoleRequest struct {
	Name         string           `json:"name" validate:"required,max=64"`
	Code         string           `json:"code" validate:"required,max=64"`
	Description  string           `json:"description"`
	BindingScope string           `json:"binding_scope"`
	Permissions  []PermissionItem `json:"permissions"`
	MenuIDs      []int64          `json:"menu_ids"`
}

// UpdateRoleRequest 更新角色
type UpdateRoleRequest struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	BindingScope string           `json:"binding_scope"`
	Permissions  []PermissionItem `json:"permissions"`
	MenuIDs      []int64          `json:"menu_ids"`
}

// PermissionItem 权限条目
type PermissionItem struct {
	Resource string `json:"resource" validate:"required"`
	Action   string `json:"action" validate:"required"`
	Effect   string `json:"effect" validate:"required,oneof=allow deny"`
}
