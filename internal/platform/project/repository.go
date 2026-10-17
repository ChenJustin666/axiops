package project

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

// Repository 项目数据访问层
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// FindByID 根据 ID 查找项目
func (r *Repository) FindByID(ctx context.Context, id int64) (*Project, error) {
	query := `SELECT id, name, code, description, status, delivery_type, delivery_status, owner_user_id, created_by, initialization_status, initialization_step, initialized_at, created_at, updated_at FROM project WHERE id = ?`
	var p Project
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Code, &p.Description, &p.Status, &p.DeliveryType, &p.DeliveryStatus,
		&p.OwnerUserID, &p.CreatedBy, &p.InitializationStatus, &p.InitializationStep, &p.InitializedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// FindByCode 查找仍占用编码的项目；已删除历史不占用编码。
func (r *Repository) FindByCode(ctx context.Context, code string) (*Project, error) {
	query := `SELECT id, name, code, description, status, delivery_type, delivery_status, owner_user_id, created_by, initialization_status, initialization_step, initialized_at, created_at, updated_at FROM project WHERE code = ? AND status <> 'deleted'`
	var p Project
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&p.ID, &p.Name, &p.Code, &p.Description, &p.Status, &p.DeliveryType, &p.DeliveryStatus,
		&p.OwnerUserID, &p.CreatedBy, &p.InitializationStatus, &p.InitializationStep, &p.InitializedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// List 分页查询
func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, keyword, status string, page, pageSize int) ([]Project, int64, error) {
	var conditions []string
	var args []interface{}

	scopePredicate, scopeArgs := scope.SQLPredicate("id", "", "")
	conditions = append(conditions, scopePredicate)
	args = append(args, scopeArgs...)

	if keyword != "" {
		conditions = append(conditions, "(name LIKE ? OR code LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	} else {
		conditions = append(conditions, "status <> 'deleted'")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// 总数
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM project"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 数据
	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, name, code, description, status, delivery_type, delivery_status, owner_user_id, created_by, initialization_status, initialization_step, initialized_at, created_at, updated_at FROM project` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.Description, &p.Status, &p.DeliveryType, &p.DeliveryStatus, &p.OwnerUserID, &p.CreatedBy, &p.InitializationStatus, &p.InitializationStep, &p.InitializedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		projects = append(projects, p)
	}
	return projects, total, nil
}

// ListByUserID 查询用户参与的项目
func (r *Repository) ListByUserID(ctx context.Context, userID int64) ([]Project, error) {
	query := `SELECT p.id, p.name, p.code, p.description, p.status, p.delivery_type, p.delivery_status, p.owner_user_id, p.created_by, p.initialization_status, p.initialization_step, p.initialized_at, p.created_at, p.updated_at 
		FROM project p 
		INNER JOIN project_member pm ON p.id = pm.project_id 
		WHERE pm.user_id = ? AND p.status <> 'deleted'
		ORDER BY p.id DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.Description, &p.Status, &p.DeliveryType, &p.DeliveryStatus, &p.OwnerUserID, &p.CreatedBy, &p.InitializationStatus, &p.InitializationStep, &p.InitializedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// Create 创建项目
func (r *Repository) Create(ctx context.Context, p *Project) (int64, error) {
	query := `INSERT INTO project (name, code, description, status, delivery_type, delivery_status, owner_user_id, created_by, initialization_status, initialization_step, initialized_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, p.Name, p.Code, p.Description, p.Status, p.DeliveryType, p.DeliveryStatus, p.OwnerUserID, p.CreatedBy, p.InitializationStatus, p.InitializationStep, p.InitializedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) CreateWithOwnerRole(ctx context.Context, p *Project, roleID int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO project (name, code, description, status, delivery_type, delivery_status, owner_user_id, created_by, initialization_status, initialization_step, initialized_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Code, p.Description, p.Status, p.DeliveryType, p.DeliveryStatus, p.OwnerUserID, p.CreatedBy, p.InitializationStatus, p.InitializationStep, p.InitializedAt)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project_member (project_id, user_id, role_id, created_by) VALUES (?, ?, ?, ?)`,
		id, p.CreatedBy, roleID, p.CreatedBy); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// Update 更新项目
func (r *Repository) Update(ctx context.Context, p *Project) error {
	// Evaluate the existing status before assigning it: a stale request loaded
	// before final completion must not overwrite the completed state/timestamp.
	query := `UPDATE project SET name=?, description=?, status=?, owner_user_id=?,
		initialized_at=IF(initialization_status='completed', initialized_at, ?),
		initialization_step=IF(initialization_status='completed', initialization_step, ?),
		initialization_status=IF(initialization_status='completed', initialization_status, ?),
		updated_at=? WHERE id=? AND status<>'deleted'`
	result, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.Status, p.OwnerUserID, p.InitializedAt, p.InitializationStep, p.InitializationStatus, time.Now(), p.ID)
	if err == nil {
		if affected, e := result.RowsAffected(); e != nil {
			return e
		} else if affected == 0 {
			var status string
			if err := r.db.QueryRowContext(ctx, "SELECT status FROM project WHERE id=?", p.ID).Scan(&status); err != nil {
				return err
			}
			if status == "deleted" {
				return projectstate.ErrDeleted
			}
		}
	}
	return err
}

// ConfirmDeliveryProfile atomically converts one legacy project to an active type.
func (r *Repository) ConfirmDeliveryProfile(ctx context.Context, projectID int64, deliveryType string) (bool, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE project SET delivery_type=?, delivery_status=?, updated_at=? WHERE id=? AND status<>'deleted' AND delivery_type=? AND delivery_status=?`,
		deliveryType, DeliveryStatusActive, time.Now(), projectID, DeliveryTypeUnclassified, DeliveryStatusPendingConfirmation)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// --- 项目成员 ---

// GetMembers 获取项目成员
func (r *Repository) GetMembers(ctx context.Context, projectID int64) ([]ProjectMember, error) {
	query := `SELECT id, project_id, user_id, role_id, joined_at, created_by FROM project_member WHERE project_id = ?`
	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ProjectMember
	for rows.Next() {
		var m ProjectMember
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.UserID, &m.RoleID, &m.JoinedAt, &m.CreatedBy); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// AddMember 添加成员
func (r *Repository) AddMember(ctx context.Context, m *ProjectMember) error {
	query := `INSERT INTO project_member (project_id, user_id, role_id, created_by) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE role_id=VALUES(role_id)`
	_, err := r.db.ExecContext(ctx, query, m.ProjectID, m.UserID, m.RoleID, m.CreatedBy)
	return err
}

// RemoveMember 移除成员
func (r *Repository) RemoveMember(ctx context.Context, projectID, userID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM project_member WHERE project_id = ? AND user_id = ?", projectID, userID)
	return err
}

func (r *Repository) IsScopedRole(ctx context.Context, roleID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM role WHERE id = ? AND binding_scope = 'scoped'", roleID).Scan(&count)
	return count == 1, err
}

func (r *Repository) RoleIDByCode(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, "SELECT id FROM role WHERE code = ? AND binding_scope = 'scoped'", code).Scan(&id)
	return id, err
}
