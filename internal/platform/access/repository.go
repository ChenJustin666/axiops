package access

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListBindings(ctx context.Context, subjectType string, subjectID int64) ([]RoleBinding, error) {
	conditions := []string{"1 = 1"}
	var args []interface{}
	if subjectType != "" {
		conditions = append(conditions, "subject_type = ?")
		args = append(args, subjectType)
	}
	if subjectID > 0 {
		conditions = append(conditions, "subject_id = ?")
		args = append(args, subjectID)
	}
	query := `SELECT id, subject_type, subject_id, role_id, scope_type, scope_id, environment,
		status, expires_at, created_by, created_at, updated_at
		FROM role_binding WHERE ` + strings.Join(conditions, " AND ") + ` ORDER BY id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []RoleBinding
	for rows.Next() {
		var binding RoleBinding
		if err := rows.Scan(&binding.ID, &binding.SubjectType, &binding.SubjectID, &binding.RoleID,
			&binding.ScopeType, &binding.ScopeID, &binding.Environment, &binding.Status,
			&binding.ExpiresAt, &binding.CreatedBy, &binding.CreatedAt, &binding.UpdatedAt); err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, rows.Err()
}

func (r *Repository) CreateBinding(ctx context.Context, binding *RoleBinding) (int64, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO role_binding
		(subject_type, subject_id, role_id, scope_type, scope_id, environment, status, expires_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		binding.SubjectType, binding.SubjectID, binding.RoleID, binding.ScopeType, binding.ScopeID,
		binding.Environment, binding.ExpiresAt, binding.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdateBinding(ctx context.Context, id int64, status *int, expiresAt interface{}, clearExpiresAt bool) error {
	var fields []string
	var args []interface{}
	if status != nil {
		fields = append(fields, "status = ?")
		args = append(args, *status)
	}
	if expiresAt != nil {
		fields = append(fields, "expires_at = ?")
		args = append(args, expiresAt)
	} else if clearExpiresAt {
		fields = append(fields, "expires_at = NULL")
	}
	if len(fields) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := r.db.ExecContext(ctx, "UPDATE role_binding SET "+strings.Join(fields, ", ")+" WHERE id = ?", args...)
	return err
}

func (r *Repository) DeleteBinding(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM role_binding WHERE id = ?", id)
	return err
}

func (r *Repository) RoleCode(ctx context.Context, roleID int64) (string, error) {
	var code string
	err := r.db.QueryRowContext(ctx, "SELECT code FROM role WHERE id = ?", roleID).Scan(&code)
	return code, err
}

func (r *Repository) RoleBindingScope(ctx context.Context, roleID int64) (string, error) {
	var scope string
	err := r.db.QueryRowContext(ctx, "SELECT binding_scope FROM role WHERE id = ?", roleID).Scan(&scope)
	return scope, err
}

func (r *Repository) BindingRoleCode(ctx context.Context, bindingID int64) (string, error) {
	var code string
	err := r.db.QueryRowContext(ctx, `SELECT r.code FROM role_binding rb
		JOIN role r ON r.id = rb.role_id WHERE rb.id = ?`, bindingID).Scan(&code)
	return code, err
}

func (r *Repository) ListMenusForUser(ctx context.Context, userID int64) ([]Menu, error) {
	const query = `
		SELECT DISTINCT m.id, m.name, m.code, m.parent_id, m.path, m.icon, m.sort_order, m.type
		FROM menu m
		WHERE m.visible = 1
		  AND EXISTS (SELECT 1 FROM user active_user WHERE active_user.id = ? AND active_user.status = 1 AND active_user.deleted_at IS NULL)
		  AND (
			EXISTS (
				SELECT 1 FROM user_role ur
				JOIN role_permission rp ON rp.role_id = ur.role_id
				WHERE ur.user_id = ? AND rp.resource = '*' AND rp.action = '*' AND rp.effect = 'allow'
			)
			OR m.id IN (
				SELECT rm.menu_id FROM role_menu rm
				JOIN user_role ur ON ur.role_id = rm.role_id
				WHERE ur.user_id = ?
				UNION
				SELECT rm.menu_id FROM role_menu rm
				JOIN project_member pm ON pm.role_id = rm.role_id
				WHERE pm.user_id = ?
				UNION
				SELECT rm.menu_id FROM role_menu rm
				JOIN role_binding rb ON rb.role_id = rm.role_id
				JOIN user u ON u.id = ?
				WHERE rb.status = 1
				  AND (rb.expires_at IS NULL OR rb.expires_at > CURRENT_TIMESTAMP)
				  AND ((rb.subject_type = 'user' AND rb.subject_id = u.id)
				    OR (rb.subject_type = 'department' AND rb.subject_id = u.department_id))
			)
		)
		ORDER BY m.sort_order, m.id`
	rows, err := r.db.QueryContext(ctx, query, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []Menu
	for rows.Next() {
		var menu Menu
		if err := rows.Scan(&menu.ID, &menu.Name, &menu.Code, &menu.ParentID, &menu.Path,
			&menu.Icon, &menu.SortOrder, &menu.Type); err != nil {
			return nil, err
		}
		menus = append(menus, menu)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.withParentMenus(ctx, menus)
}

// withParentMenus makes the returned tree renderable when a role grants a
// child menu without explicitly granting its parent directory.
func (r *Repository) withParentMenus(ctx context.Context, menus []Menu) ([]Menu, error) {
	if len(menus) == 0 {
		return menus, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, code, parent_id, path, icon, sort_order, type
		FROM menu WHERE visible = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	all := make(map[int64]Menu)
	for rows.Next() {
		var menu Menu
		if err := rows.Scan(&menu.ID, &menu.Name, &menu.Code, &menu.ParentID, &menu.Path, &menu.Icon, &menu.SortOrder, &menu.Type); err != nil {
			return nil, err
		}
		all[menu.ID] = menu
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	selected := make(map[int64]Menu, len(menus))
	for _, menu := range menus {
		selected[menu.ID] = menu
	}
	for _, menu := range menus {
		for parentID := menu.ParentID; parentID > 0; {
			parent, ok := all[parentID]
			if !ok {
				break
			}
			if _, exists := selected[parent.ID]; exists {
				parentID = parent.ParentID
				continue
			}
			selected[parent.ID] = parent
			parentID = parent.ParentID
		}
	}
	result := make([]Menu, 0, len(selected))
	for _, menu := range selected {
		result = append(result, menu)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder == result[j].SortOrder {
			return result[i].ID < result[j].ID
		}
		return result[i].SortOrder < result[j].SortOrder
	})
	return result, nil
}

func (r *Repository) ListMenus(ctx context.Context) ([]Menu, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, code, parent_id, path, icon, sort_order, type
		FROM menu WHERE visible = 1 ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var menus []Menu
	for rows.Next() {
		var menu Menu
		if err := rows.Scan(&menu.ID, &menu.Name, &menu.Code, &menu.ParentID, &menu.Path,
			&menu.Icon, &menu.SortOrder, &menu.Type); err != nil {
			return nil, err
		}
		menus = append(menus, menu)
	}
	return menus, rows.Err()
}
