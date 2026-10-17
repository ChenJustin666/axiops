package role

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Repository 角色数据访问层
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindByID 根据 ID 查找角色
func (r *Repository) FindByID(ctx context.Context, id int64) (*Role, error) {
	query := `SELECT id, name, code, description, type, binding_scope, created_by, created_at, updated_at FROM role WHERE id = ?`
	var role Role
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID, &role.Name, &role.Code, &role.Description, &role.Type, &role.BindingScope,
		&role.CreatedBy, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// FindByCode 根据编码查找角色
func (r *Repository) FindByCode(ctx context.Context, code string) (*Role, error) {
	query := `SELECT id, name, code, description, type, binding_scope, created_by, created_at, updated_at FROM role WHERE code = ?`
	var role Role
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&role.ID, &role.Name, &role.Code, &role.Description, &role.Type, &role.BindingScope,
		&role.CreatedBy, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// List 角色列表
func (r *Repository) List(ctx context.Context) ([]Role, error) {
	query := `SELECT id, name, code, description, type, binding_scope, created_by, created_at, updated_at FROM role ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.Code, &role.Description, &role.Type, &role.BindingScope,
			&role.CreatedBy, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// Create 创建角色
func (r *Repository) Create(ctx context.Context, role *Role) (int64, error) {
	query := `INSERT INTO role (name, code, description, type, binding_scope, created_by) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, role.Name, role.Code, role.Description, role.Type, role.BindingScope, role.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) CreateWithAssignments(ctx context.Context, role *Role, perms []PermissionItem, menuIDs []int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO role (name, code, description, type, binding_scope, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		role.Name, role.Code, role.Description, role.Type, role.BindingScope, role.CreatedBy)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := replacePermissions(ctx, tx, id, perms); err != nil {
		return 0, err
	}
	if err := replaceMenus(ctx, tx, id, menuIDs); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// Update 更新角色
func (r *Repository) Update(ctx context.Context, role *Role) error {
	query := `UPDATE role SET name=?, description=?, binding_scope=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, role.Name, role.Description, role.BindingScope, time.Now(), role.ID)
	return err
}

func (r *Repository) UpdateWithAssignments(ctx context.Context, role *Role, perms []PermissionItem, updatePerms bool, menuIDs []int64, updateMenus bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE role SET name=?, description=?, binding_scope=?, updated_at=? WHERE id=?`,
		role.Name, role.Description, role.BindingScope, time.Now(), role.ID); err != nil {
		return err
	}
	if updatePerms {
		if err := replacePermissions(ctx, tx, role.ID, perms); err != nil {
			return err
		}
	}
	if updateMenus {
		if err := replaceMenus(ctx, tx, role.ID, menuIDs); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

func replacePermissions(ctx context.Context, executor sqlExecutor, roleID int64, perms []PermissionItem) error {
	if _, err := executor.ExecContext(ctx, "DELETE FROM role_permission WHERE role_id = ?", roleID); err != nil {
		return err
	}
	if len(perms) == 0 {
		return nil
	}
	query := "INSERT INTO role_permission (role_id, resource, action, effect) VALUES "
	values := make([]string, 0, len(perms))
	args := make([]interface{}, 0, len(perms)*4)
	for _, permission := range perms {
		values = append(values, "(?, ?, ?, ?)")
		args = append(args, roleID, permission.Resource, permission.Action, permission.Effect)
	}
	_, err := executor.ExecContext(ctx, query+strings.Join(values, ","), args...)
	return err
}

func replaceMenus(ctx context.Context, executor sqlExecutor, roleID int64, menuIDs []int64) error {
	if _, err := executor.ExecContext(ctx, "DELETE FROM role_menu WHERE role_id = ?", roleID); err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if menuID <= 0 {
			continue
		}
		if _, err := executor.ExecContext(ctx, "INSERT INTO role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID); err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除角色
func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM role WHERE id = ? AND type != 'system'", id)
	return err
}

func (r *Repository) CanUseBindingScope(ctx context.Context, roleID int64, bindingScope string) (bool, error) {
	var count int
	switch bindingScope {
	case "platform":
		err := r.db.QueryRowContext(ctx, `SELECT
			(SELECT COUNT(*) FROM project_member WHERE role_id = ?) +
			(SELECT COUNT(*) FROM role_binding WHERE role_id = ?)`, roleID, roleID).Scan(&count)
		return count == 0, err
	case "scoped":
		err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_role WHERE role_id = ?", roleID).Scan(&count)
		return count == 0, err
	default:
		return false, nil
	}
}

// GetPermissions 获取角色权限
func (r *Repository) GetPermissions(ctx context.Context, roleID int64) ([]RolePermission, error) {
	query := `SELECT id, role_id, resource, action, effect, created_at FROM role_permission WHERE role_id = ?`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []RolePermission
	for rows.Next() {
		var p RolePermission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.Resource, &p.Action, &p.Effect, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// SetPermissions 设置角色权限（替换）
func (r *Repository) SetPermissions(ctx context.Context, roleID int64, perms []PermissionItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧权限
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_permission WHERE role_id = ?", roleID); err != nil {
		return err
	}

	// 插入新权限
	if len(perms) > 0 {
		query := "INSERT INTO role_permission (role_id, resource, action, effect) VALUES "
		var values []string
		var args []interface{}
		for _, p := range perms {
			values = append(values, "(?, ?, ?, ?)")
			args = append(args, roleID, p.Resource, p.Action, p.Effect)
		}
		query += strings.Join(values, ",")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetPermissionsByRoleIDs 批量获取角色权限（用于鉴权）
func (r *Repository) GetPermissionsByRoleIDs(ctx context.Context, roleIDs []int64) ([]RolePermission, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(roleIDs))
	args := make([]interface{}, len(roleIDs))
	for i, id := range roleIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `SELECT id, role_id, resource, action, effect, created_at FROM role_permission WHERE role_id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []RolePermission
	for rows.Next() {
		var p RolePermission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.Resource, &p.Action, &p.Effect, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *Repository) GetMenuIDs(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT menu_id FROM role_menu WHERE role_id = ? ORDER BY menu_id", roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var menuIDs []int64
	for rows.Next() {
		var menuID int64
		if err := rows.Scan(&menuID); err != nil {
			return nil, err
		}
		menuIDs = append(menuIDs, menuID)
	}
	return menuIDs, rows.Err()
}

func (r *Repository) SetMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM role_menu WHERE role_id = ?", roleID); err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		if menuID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
