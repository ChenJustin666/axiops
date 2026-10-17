package user

import "time"

// User 用户模型
type User struct {
	ID           int64      `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Password     string     `json:"-" db:"password"`
	Nickname     string     `json:"nickname" db:"nickname"`
	Email        string     `json:"email" db:"email"`
	Phone        string     `json:"phone" db:"phone"`
	Avatar       string     `json:"avatar" db:"avatar"`
	Status       int8       `json:"status" db:"status"`
	DepartmentID int64      `json:"department_id" db:"department_id"`
	LastLoginAt  *time.Time `json:"last_login_at" db:"last_login_at"`
	LastLoginIP  string     `json:"last_login_ip" db:"last_login_ip"`
	CreatedBy    int64      `json:"created_by" db:"created_by"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// UserRole 用户角色绑定
type UserRole struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	RoleID    int64     `json:"role_id" db:"role_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	CreatedBy int64     `json:"created_by" db:"created_by"`
}

// UserWithRoles 带角色信息的用户
type UserWithRoles struct {
	User
	RoleIDs   []int64  `json:"role_ids"`
	RoleNames []string `json:"role_names"`
}
