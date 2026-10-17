package credential

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

// Repository 凭据数据访问层
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Credential, error) {
	query := `SELECT id, name, type, description, data, project_id, created_by, created_at, updated_at FROM credential WHERE id = ?`
	var c Credential
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Type, &c.Description, &c.Data, &c.ProjectID, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, credType string, projectID int64) ([]Credential, error) {
	var conditions []string
	var args []interface{}

	scopePredicate, scopeArgs := scope.SQLPredicate("project_id", "", "")
	// Platform credentials can be referenced by project-scoped code sources.
	// A platform-only query (project_id=0) remains restricted to platform rows.
	if projectID != 0 && scopePredicate != "1 = 0" {
		scopePredicate = "(project_id = 0 OR " + scopePredicate + ")"
	}
	conditions = append(conditions, scopePredicate)
	args = append(args, scopeArgs...)

	if credType != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, credType)
	}
	if projectID >= 0 {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	query := `SELECT id, name, type, description, CASE WHEN type='secret' THEN data ELSE '' END, project_id, created_by, created_at, updated_at FROM credential` + where + ` ORDER BY id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Description, &c.Data, &c.ProjectID, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, nil
}

func (r *Repository) Create(ctx context.Context, c *Credential) (int64, error) {
	query := `INSERT INTO credential (name, type, description, data, project_id, created_by) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, c.Name, c.Type, c.Description, c.Data, c.ProjectID, c.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) Update(ctx context.Context, c *Credential) error {
	query := `UPDATE credential SET name=?, description=?, data=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, c.Name, c.Description, c.Data, time.Now(), c.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM credential WHERE id = ?", id)
	return err
}
