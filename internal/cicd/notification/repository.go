package notification

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) Create(ctx context.Context, s Subscription) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO notification_subscription(project_id,user_id,name,channel,credential_id,recipient,on_success,on_failure,enabled,created_by) VALUES(?,?,?,?,?,?,?,?,1,?)`, s.ProjectID, s.UserID, s.Name, s.Channel, s.CredentialID, s.Recipient, s.OnSuccess, s.OnFailure, s.CreatedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
func (r *Repository) List(ctx context.Context, projectID int64, scope authz.ScopeSet) ([]Subscription, error) {
	predicate, args := scope.SQLPredicate("project_id", "", "")
	query := `SELECT id,project_id,user_id,name,channel,credential_id,recipient,on_success,on_failure,enabled,created_by,created_at,updated_at FROM notification_subscription WHERE ` + predicate
	if projectID > 0 {
		query += " AND project_id=?"
		args = append(args, projectID)
	}
	query += " ORDER BY id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.UserID, &s.Name, &s.Channel, &s.CredentialID, &s.Recipient, &s.OnSuccess, &s.OnFailure, &s.Enabled, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}
func (r *Repository) Delete(ctx context.Context, id, projectID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM notification_subscription WHERE id=? AND project_id=?`, id, projectID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

func (r *Repository) ListDeliveries(ctx context.Context, projectID int64, status string, page, pageSize int, scope authz.ScopeSet) ([]Delivery, int64, error) {
	predicate, args := scope.SQLPredicate("o.project_id", "", "")
	where := " WHERE " + predicate
	if projectID > 0 {
		where += " AND o.project_id=?"
		args = append(args, projectID)
	}
	if status != "" {
		where += " AND o.status=?"
		args = append(args, status)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_outbox o`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT o.id,o.subscription_id,o.project_id,COALESCE(s.name,''),COALESCE(s.channel,''),
		o.event_type,o.event_id,o.status,o.attempts,o.next_attempt_at,o.last_error,o.created_at,o.updated_at
		FROM notification_outbox o LEFT JOIN notification_subscription s ON s.id=o.subscription_id` + where +
		" ORDER BY o.id DESC LIMIT ? OFFSET ?"
	queryArgs := append(append([]interface{}{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Delivery, 0)
	for rows.Next() {
		var item Delivery
		if err := rows.Scan(&item.ID, &item.SubscriptionID, &item.ProjectID, &item.SubscriptionName,
			&item.Channel, &item.EventType, &item.EventID, &item.Status, &item.Attempts,
			&item.NextAttemptAt, &item.LastError, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
func (r *Repository) FindCredential(ctx context.Context, id int64) (*Credential, error) {
	var c Credential
	err := r.db.QueryRowContext(ctx, `SELECT type,data,project_id FROM credential WHERE id=?`, id).Scan(&c.Type, &c.Data, &c.ProjectID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}
func (r *Repository) Claim(ctx context.Context, worker string, lease time.Duration) (*Outbox, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM notification_outbox WHERE (status='pending' AND next_attempt_at<=CURRENT_TIMESTAMP) OR (status='processing' AND locked_at<DATE_SUB(CURRENT_TIMESTAMP,INTERVAL ? SECOND)) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, int(lease/time.Second)).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE notification_outbox SET status='processing',attempts=attempts+1,locked_by=?,locked_at=CURRENT_TIMESTAMP WHERE id=?`, worker, id); err != nil {
		return nil, err
	}
	var item Outbox
	err = tx.QueryRowContext(ctx, `SELECT id,subscription_id,project_id,event_type,event_id,payload,status,attempts,locked_by,last_error,delivery_key FROM notification_outbox WHERE id=?`, id).Scan(&item.ID, &item.SubscriptionID, &item.ProjectID, &item.EventType, &item.EventID, &item.Payload, &item.Status, &item.Attempts, &item.LockedBy, &item.LastError, &item.DeliveryKey)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &item, nil
}
func (r *Repository) DeliveryTarget(ctx context.Context, subscriptionID int64) (Subscription, *Credential, error) {
	var s Subscription
	err := r.db.QueryRowContext(ctx, `SELECT id,project_id,user_id,name,channel,credential_id,recipient,on_success,on_failure,enabled,created_by,created_at,updated_at FROM notification_subscription WHERE id=? AND enabled=1`, subscriptionID).Scan(&s.ID, &s.ProjectID, &s.UserID, &s.Name, &s.Channel, &s.CredentialID, &s.Recipient, &s.OnSuccess, &s.OnFailure, &s.Enabled, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, nil, err
	}
	c, err := r.FindCredential(ctx, s.CredentialID)
	return s, c, err
}
func (r *Repository) Complete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notification_outbox SET status='succeeded',locked_by='',locked_at=NULL,last_error='' WHERE id=? AND status='processing'`, id)
	return err
}
func (r *Repository) Fail(ctx context.Context, id int64, attempts, maxAttempts int, message string) error {
	status := "pending"
	if attempts >= maxAttempts {
		status = "dead"
	}
	_, err := r.db.ExecContext(ctx, `UPDATE notification_outbox SET status=?,next_attempt_at=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL LEAST(300,POW(2,attempts)) SECOND),locked_by='',locked_at=NULL,last_error=? WHERE id=? AND status='processing'`, status, message, id)
	return err
}
