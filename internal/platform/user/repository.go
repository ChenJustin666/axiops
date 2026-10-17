package user

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Repository 用户数据访问层
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindByUsername 根据用户名查找用户
func (r *Repository) FindByUsername(ctx context.Context, username string) (*User, error) {
	query := `SELECT id, username, password, nickname, email, phone, avatar, status, 
		department_id, last_login_at, last_login_ip, created_by, created_at, updated_at 
		FROM user WHERE username = ? AND deleted_at IS NULL`

	var u User
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Email, &u.Phone,
		&u.Avatar, &u.Status, &u.DepartmentID, &u.LastLoginAt, &u.LastLoginIP,
		&u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// FindByID 根据 ID 查找用户
func (r *Repository) FindByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, username, password, nickname, email, phone, avatar, status, 
		department_id, last_login_at, last_login_ip, created_by, created_at, updated_at 
		FROM user WHERE id = ? AND deleted_at IS NULL`

	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Email, &u.Phone,
		&u.Avatar, &u.Status, &u.DepartmentID, &u.LastLoginAt, &u.LastLoginIP,
		&u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// List 分页查询用户列表
func (r *Repository) List(ctx context.Context, keyword string, status int, page, pageSize int) ([]User, int64, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "deleted_at IS NULL")

	if keyword != "" {
		conditions = append(conditions, "(username LIKE ? OR nickname LIKE ? OR email LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	if status >= 0 {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	where := strings.Join(conditions, " AND ")

	// 查总数
	countQuery := "SELECT COUNT(*) FROM user WHERE " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 查数据
	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, username, password, nickname, email, phone, avatar, status, 
		department_id, last_login_at, last_login_ip, created_by, created_at, updated_at 
		FROM user WHERE ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Password, &u.Nickname, &u.Email, &u.Phone,
			&u.Avatar, &u.Status, &u.DepartmentID, &u.LastLoginAt, &u.LastLoginIP,
			&u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

// Create 创建用户
func (r *Repository) Create(ctx context.Context, u *User) (int64, error) {
	query := `INSERT INTO user (username, password, nickname, email, phone, avatar, status, department_id, created_by) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		u.Username, u.Password, u.Nickname, u.Email, u.Phone, u.Avatar, u.Status, u.DepartmentID, u.CreatedBy,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) CreateWithRoles(ctx context.Context, u *User, roleIDs []int64, createdBy int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO user (username, password, nickname, email, phone, avatar, status, department_id, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.Password, u.Nickname, u.Email, u.Phone, u.Avatar, u.Status, u.DepartmentID, u.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if len(roleIDs) > 0 {
		values := make([]string, 0, len(roleIDs))
		args := make([]interface{}, 0, len(roleIDs)*3)
		for _, roleID := range roleIDs {
			values = append(values, "(?, ?, ?)")
			args = append(args, id, roleID, createdBy)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO user_role (user_id, role_id, created_by) VALUES "+strings.Join(values, ","), args...); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// Update 更新用户
func (r *Repository) Update(ctx context.Context, u *User) error {
	query := `UPDATE user SET nickname=?, email=?, phone=?, avatar=?, status=?, department_id=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query,
		u.Nickname, u.Email, u.Phone, u.Avatar, u.Status, u.DepartmentID, time.Now(), u.ID,
	)
	return err
}

// UpdatePassword 更新密码
func (r *Repository) UpdatePassword(ctx context.Context, id int64, hashedPassword string) error {
	query := `UPDATE user SET password=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, hashedPassword, time.Now(), id)
	return err
}

// UpdateLoginInfo 更新登录信息
func (r *Repository) UpdateLoginInfo(ctx context.Context, id int64, ip string) error {
	query := `UPDATE user SET last_login_at=?, last_login_ip=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), ip, id)
	return err
}

// Delete 软删除用户
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `UPDATE user SET deleted_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// GetUserRoleIDs 获取用户的角色 ID 列表
func (r *Repository) GetUserRoleIDs(ctx context.Context, userID int64) ([]int64, error) {
	query := `SELECT role_id FROM user_role WHERE user_id = ?`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []int64
	for rows.Next() {
		var roleID int64
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs, nil
}

func (r *Repository) ValidatePlatformRoleIDs(ctx context.Context, roleIDs []int64) (bool, error) {
	if len(roleIDs) == 0 {
		return true, nil
	}
	placeholders := make([]string, len(roleIDs))
	args := make([]interface{}, len(roleIDs))
	for i, roleID := range roleIDs {
		placeholders[i] = "?"
		args[i] = roleID
	}
	var count int
	query := "SELECT COUNT(*) FROM role WHERE binding_scope = 'platform' AND id IN (" + strings.Join(placeholders, ",") + ")"
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count == len(roleIDs), nil
}

func (r *Repository) ContainsRoleCode(ctx context.Context, roleIDs []int64, code string) (bool, error) {
	if len(roleIDs) == 0 {
		return false, nil
	}
	placeholders := make([]string, len(roleIDs))
	args := make([]interface{}, 0, len(roleIDs)+1)
	for _, roleID := range roleIDs {
		placeholders[len(args)] = "?"
		args = append(args, roleID)
	}
	args = append(args, code)
	var count int
	query := "SELECT COUNT(*) FROM role WHERE id IN (" + strings.Join(placeholders, ",") + ") AND code = ?"
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) ContainsProtectedRolePermissions(ctx context.Context, roleIDs []int64) (bool, error) {
	if len(roleIDs) == 0 {
		return false, nil
	}
	placeholders := make([]string, len(roleIDs))
	args := make([]interface{}, len(roleIDs))
	for i, roleID := range roleIDs {
		placeholders[i] = "?"
		args[i] = roleID
	}
	var count int
	query := `SELECT COUNT(*) FROM role_permission
		WHERE role_id IN (` + strings.Join(placeholders, ",") + `)
		  AND effect = 'allow' AND resource IN ('*', 'system_role')`
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) UserHasRoleCode(ctx context.Context, userID int64, code string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_role ur
		JOIN role r ON r.id = ur.role_id
		WHERE ur.user_id = ? AND r.code = ?`, userID, code).Scan(&count)
	return count > 0, err
}

// SetUserRoles 设置用户角色
func (r *Repository) SetUserRoles(ctx context.Context, userID int64, roleIDs []int64, createdBy int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧角色
	if _, err := tx.ExecContext(ctx, "DELETE FROM user_role WHERE user_id = ?", userID); err != nil {
		return err
	}

	// 插入新角色
	if len(roleIDs) > 0 {
		query := "INSERT INTO user_role (user_id, role_id, created_by) VALUES "
		var values []string
		var args []interface{}
		for _, roleID := range roleIDs {
			values = append(values, "(?, ?, ?)")
			args = append(args, userID, roleID, createdBy)
		}
		query += strings.Join(values, ",")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}
