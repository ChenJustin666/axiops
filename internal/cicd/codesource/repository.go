package codesource

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, source *CodeSource) (int64, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO code_source
		(name, type, url, credential_id, webhook_credential_id, status, project_id, created_by)
		VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`, source.Name, source.Type, source.URL, source.CredentialID, source.WebhookCredentialID, source.ProjectID, source.CreatedBy)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*CodeSource, error) {
	var source CodeSource
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, name, type, url, credential_id, COALESCE(webhook_credential_id,0), status, created_by, created_at, updated_at
		FROM code_source WHERE id=?`, id).Scan(&source.ID, &source.ProjectID, &source.Name, &source.Type, &source.URL,
		&source.CredentialID, &source.WebhookCredentialID, &source.Status, &source.CreatedBy, &source.CreatedAt, &source.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &source, err
}

func (r *Repository) List(ctx context.Context, projectID int64, scope authz.ScopeSet) ([]CodeSource, error) {
	predicate, args := scope.SQLPredicate("project_id", "", "")
	// Platform sources are reusable by project-scoped users. Keep the scope
	// predicate for project sources, while allowing the explicit global row.
	if predicate != "1 = 0" {
		predicate = "(project_id = 0 OR " + predicate + ")"
	}
	query := `SELECT id, project_id, name, type, url, credential_id, COALESCE(webhook_credential_id,0), status, created_by, created_at, updated_at
		FROM code_source WHERE ` + predicate
	if projectID > 0 {
		// Global sources are reusable by every project. Project-scoped sources
		// remain restricted to the requested project.
		query += " AND (project_id = 0 OR project_id = ?)"
		args = append(args, projectID)
	} else {
		query += " AND project_id = 0"
	}
	query += " ORDER BY id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []CodeSource
	for rows.Next() {
		var source CodeSource
		if err := rows.Scan(&source.ID, &source.ProjectID, &source.Name, &source.Type, &source.URL,
			&source.CredentialID, &source.WebhookCredentialID, &source.Status, &source.CreatedBy, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func (r *Repository) Update(ctx context.Context, source *CodeSource) error {
	result, err := r.db.ExecContext(ctx, `UPDATE code_source SET name=?, url=?, credential_id=?, webhook_credential_id=?, status=? WHERE id=?`,
		source.Name, source.URL, source.CredentialID, source.WebhookCredentialID, source.Status, source.ID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		// MySQL reports zero rows when an UPDATE keeps all values unchanged.
		// Confirm the row exists before treating that idempotent update as success.
		var exists int
		if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM code_source WHERE id=?`, source.ID).Scan(&exists); err == sql.ErrNoRows {
			return fmt.Errorf("code source update affected no rows")
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM code_source WHERE id=?
		AND NOT EXISTS (SELECT 1 FROM app_service WHERE code_source_id=?)`, id, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("code source is still referenced or does not exist")
	}
	return nil
}

func (r *Repository) FindCredential(ctx context.Context, id int64) (*CredentialRef, error) {
	var credential CredentialRef
	err := r.db.QueryRowContext(ctx, `SELECT id, type, data, project_id FROM credential WHERE id=?`, id).
		Scan(&credential.ID, &credential.Type, &credential.Data, &credential.ProjectID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &credential, err
}
