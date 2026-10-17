package release

import (
	"context"
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*ReleaseRecord, error) {
	query := `SELECT id, service_id, project_id, environment_id, cluster_id, namespace, release_number, 
		image_tag, image_full, strategy, status, approval_status, approval_user_id, deploy_user_id, 
		rollback_from, duration_sec, started_at, finished_at, created_at FROM release_record WHERE id = ?`
	var rec ReleaseRecord
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rec.ID, &rec.ServiceID, &rec.ProjectID, &rec.EnvironmentID, &rec.ClusterID,
		&rec.Namespace, &rec.ReleaseNumber, &rec.ImageTag, &rec.ImageFull, &rec.Strategy,
		&rec.Status, &rec.ApprovalStatus, &rec.ApprovalUserID, &rec.DeployUserID,
		&rec.RollbackFrom, &rec.DurationSec, &rec.StartedAt, &rec.FinishedAt, &rec.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) ListByService(ctx context.Context, serviceID int64, page, pageSize int) ([]ReleaseRecord, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM release_record WHERE service_id = ?", serviceID).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `SELECT id, service_id, project_id, environment_id, cluster_id, namespace, release_number, 
		image_tag, image_full, strategy, status, approval_status, approval_user_id, deploy_user_id, 
		rollback_from, duration_sec, started_at, finished_at, created_at 
		FROM release_record WHERE service_id = ? ORDER BY release_number DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, serviceID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []ReleaseRecord
	for rows.Next() {
		var rec ReleaseRecord
		if err := rows.Scan(
			&rec.ID, &rec.ServiceID, &rec.ProjectID, &rec.EnvironmentID, &rec.ClusterID,
			&rec.Namespace, &rec.ReleaseNumber, &rec.ImageTag, &rec.ImageFull, &rec.Strategy,
			&rec.Status, &rec.ApprovalStatus, &rec.ApprovalUserID, &rec.DeployUserID,
			&rec.RollbackFrom, &rec.DurationSec, &rec.StartedAt, &rec.FinishedAt, &rec.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	return records, total, nil
}

func (r *Repository) GetNextReleaseNumber(ctx context.Context, serviceID int64) (int, error) {
	var maxNum int
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(release_number), 0) FROM release_record WHERE service_id = ?", serviceID).Scan(&maxNum)
	if err != nil {
		return 0, err
	}
	return maxNum + 1, nil
}

func (r *Repository) Create(ctx context.Context, rec *ReleaseRecord) (int64, error) {
	query := `INSERT INTO release_record (service_id, project_id, environment_id, cluster_id, namespace, 
		release_number, image_tag, image_full, strategy, status, approval_status, deploy_user_id, rollback_from) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		rec.ServiceID, rec.ProjectID, rec.EnvironmentID, rec.ClusterID, rec.Namespace,
		rec.ReleaseNumber, rec.ImageTag, rec.ImageFull, rec.Strategy, rec.Status,
		rec.ApprovalStatus, rec.DeployUserID, rec.RollbackFrom,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status string, durationSec int) error {
	now := time.Now()
	if status == "deploying" {
		_, err := r.db.ExecContext(ctx, "UPDATE release_record SET status=?, started_at=? WHERE id=?", status, now, id)
		return err
	}
	_, err := r.db.ExecContext(ctx, "UPDATE release_record SET status=?, duration_sec=?, finished_at=? WHERE id=?", status, durationSec, now, id)
	return err
}

func (r *Repository) UpdateApproval(ctx context.Context, id int64, approvalStatus string, approvalUserID int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE release_record SET approval_status=?, approval_user_id=? WHERE id=?", approvalStatus, approvalUserID, id)
	return err
}
