package build

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/notification"
)

const (
	CommandSubmit = "submit"
	CommandCancel = "cancel"

	CommandPending    = "pending"
	CommandProcessing = "processing"
	CommandSucceeded  = "succeeded"
	CommandFailed     = "failed"
	CommandDead       = "dead"
)

// ClaimCommand atomically leases the oldest ready command to a worker. The
// SKIP LOCKED clause prevents one slow provider from blocking other workers.
func (r *Repository) ClaimCommand(ctx context.Context, workerID string) (*BuildCommand, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var command BuildCommand
	row := tx.QueryRowContext(ctx, `SELECT bc.id, bc.build_run_id, bc.command_type, bc.provider,
		bc.payload, bc.status, bc.attempts, bc.next_attempt_at, bc.locked_by, bc.locked_at, COALESCE(bc.last_error,''),
		bc.idempotency_key, bc.created_at, bc.updated_at
		FROM build_command bc
		JOIN build_run br ON br.id=bc.build_run_id
		WHERE bc.status='pending' AND bc.next_attempt_at<=CURRENT_TIMESTAMP
		  AND ((bc.command_type='cancel' AND br.status='cancelling'
		    AND NOT EXISTS (SELECT 1 FROM build_command submitting WHERE submitting.build_run_id=br.id AND submitting.command_type='submit' AND submitting.status='processing'))
		    OR (bc.command_type='submit' AND br.status='dispatching'))
		ORDER BY (bc.command_type='cancel') DESC,bc.id LIMIT 1 FOR UPDATE SKIP LOCKED`)
	err = row.Scan(
		&command.ID, &command.BuildRunID, &command.CommandType, &command.Provider,
		&command.Payload, &command.Status, &command.Attempts, &command.NextAttemptAt,
		&command.LockedBy, &command.LockedAt, &command.LastError,
		&command.IdempotencyKey, &command.CreatedAt, &command.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		var globalLimit, active int
		if err = tx.QueryRowContext(ctx, `SELECT max_running FROM build_concurrency_policy WHERE scope_type='platform' AND scope_id=0 FOR UPDATE`).Scan(&globalLimit); err != nil {
			return nil, err
		}
		if r.checkWorkerSettings {
			settings, err := findPlatformBuildSettings(ctx, tx)
			if err != nil {
				return nil, err
			}
			if workerSettingsJSON(settings) != r.workerSettingsJSON {
				return nil, tx.Commit()
			}
		}
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_run WHERE status IN ('dispatching','running','cancelling')`).Scan(&active); err != nil {
			return nil, err
		}
		if err = validateConcurrencyLimit(active, globalLimit); err != nil {
			if globalLimit > 0 && active >= globalLimit {
				return nil, tx.Commit()
			}
			return nil, err
		}
		row = tx.QueryRowContext(ctx, `SELECT bc.id,bc.build_run_id,bc.command_type,bc.provider,
			bc.payload,bc.status,bc.attempts,bc.next_attempt_at,bc.locked_by,bc.locked_at,COALESCE(bc.last_error,''),
			bc.idempotency_key,bc.created_at,bc.updated_at
			FROM build_command bc JOIN build_run br ON br.id=bc.build_run_id
			WHERE bc.status='pending' AND bc.next_attempt_at<=CURRENT_TIMESTAMP
			  AND bc.command_type='submit' AND br.status='queued'
			  AND COALESCE((SELECT enabled FROM build_platform_settings WHERE id=1),1)=1
			  AND (COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.config.cache_enabled')),'false')<>'true'
			    OR COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.config.cache_strategy')),'inherit')='emptydir'
			    OR (COALESCE(JSON_UNQUOTE(JSON_EXTRACT(br.config_snapshot,'$.config.cache_strategy')),'inherit') IN ('inherit','')
			        AND EXISTS (SELECT 1 FROM build_platform_settings WHERE id=1 AND cache_pvc_name=''))
			    OR NOT EXISTS (SELECT 1 FROM build_run cached WHERE cached.service_id=br.service_id AND cached.status IN ('dispatching','running','cancelling')))
			  AND (SELECT COUNT(*) FROM build_run active
			       WHERE active.project_id=br.project_id AND active.status IN ('dispatching','running','cancelling'))
			      < COALESCE((SELECT p.max_running FROM build_concurrency_policy p WHERE p.scope_type='project' AND p.scope_id=br.project_id),?)
			ORDER BY br.queued_at,bc.id LIMIT 1 FOR UPDATE SKIP LOCKED`, globalLimit)
		err = row.Scan(
			&command.ID, &command.BuildRunID, &command.CommandType, &command.Provider,
			&command.Payload, &command.Status, &command.Attempts, &command.NextAttemptAt,
			&command.LockedBy, &command.LockedAt, &command.LastError,
			&command.IdempotencyKey, &command.CreatedAt, &command.UpdatedAt,
		)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, tx.Commit()
		}
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE build_command
		SET status = 'processing', attempts = attempts + 1, locked_by = ?, locked_at = CURRENT_TIMESTAMP
		WHERE id = ?`, workerID, command.ID); err != nil {
		return nil, err
	}
	if command.CommandType == CommandSubmit {
		_, err := tx.ExecContext(ctx, `UPDATE build_run SET status='dispatching'
			WHERE id=? AND status IN ('queued','dispatching')`, command.BuildRunID)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	command.Status = CommandProcessing
	command.Attempts++
	command.LockedBy = workerID
	now := time.Now()
	command.LockedAt = &now
	return &command, nil
}

// ExpireQueuedRuns terminates builds that exceeded their inherited project
// queue timeout before consuming cluster capacity.
func (r *Repository) ExpireQueuedRuns(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT br.id,br.project_id,br.trigger_user_id,br.build_number
		FROM build_run br
		JOIN build_concurrency_policy gp ON gp.scope_type='platform' AND gp.scope_id=0
		LEFT JOIN build_concurrency_policy pp ON pp.scope_type='project' AND pp.scope_id=br.project_id
		WHERE br.status='queued' AND TIMESTAMPDIFF(SECOND,br.queued_at,CURRENT_TIMESTAMP)>=COALESCE(pp.queue_timeout_seconds,gp.queue_timeout_seconds)
		ORDER BY br.queued_at,br.id LIMIT ? FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	type expiredRun struct {
		id, projectID, userID int64
		buildNumber           int
	}
	var expired []expiredRun
	for rows.Next() {
		var item expiredRun
		if err := rows.Scan(&item.id, &item.projectID, &item.userID, &item.buildNumber); err != nil {
			rows.Close()
			return 0, err
		}
		expired = append(expired, item)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	reason := "build queue timeout exceeded"
	for _, item := range expired {
		result, err := tx.ExecContext(ctx, `UPDATE build_run SET status='failed',finished_at=CURRENT_TIMESTAMP,duration_seconds=TIMESTAMPDIFF(SECOND,queued_at,CURRENT_TIMESTAMP),error_message=? WHERE id=? AND status='queued'`, reason, item.id)
		if err != nil {
			return 0, err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE build_command SET status='dead',last_error=? WHERE build_run_id=? AND command_type='submit' AND status='pending'`, reason, item.id); err != nil {
			return 0, err
		}
		message := notification.Message{EventType: "build.terminal", Title: fmt.Sprintf("Build #%d failed", item.buildNumber), Content: reason, Status: StatusFailed, ProjectID: item.projectID, ResourceID: item.id, URL: fmt.Sprintf("/cicd/builds/%d", item.id)}
		if err := notification.EnqueueTerminal(ctx, tx, item.projectID, item.userID, "build.terminal", item.id, StatusFailed, message); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(expired)), nil
}

func (r *Repository) CompleteCommand(ctx context.Context, commandID int64, workerID string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE build_command
		SET status = 'succeeded', locked_by = '', locked_at = NULL, last_error = NULL
		WHERE id = ? AND status = 'processing' AND locked_by = ?`, commandID, workerID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("build command lease is no longer owned by worker")
	}
	return nil
}

func (r *Repository) FailCommand(ctx context.Context, commandID int64, workerID, lastError string, maxAttempts int, retryAfter time.Duration) error {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	retryAfter = normalizeRetryAfter(retryAfter)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var attempts int
	var buildRunID int64
	if err := tx.QueryRowContext(ctx, `SELECT attempts, build_run_id FROM build_command
		WHERE id = ? AND status = 'processing' AND locked_by = ? FOR UPDATE`, commandID, workerID,
	).Scan(&attempts, &buildRunID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("build command lease is no longer owned by worker")
		}
		return err
	}
	dead := attempts >= maxAttempts
	status := CommandPending
	if dead {
		status = CommandDead
	}
	nextAttemptAt := time.Now().Add(retryAfter)
	result, err := tx.ExecContext(ctx, `UPDATE build_command
		SET status = ?,
		    next_attempt_at = ?,
		    locked_by = '', locked_at = NULL, last_error = ?
		WHERE id = ? AND status = 'processing' AND locked_by = ?`,
		status, nextAttemptAt, lastError, commandID, workerID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("build command lease is no longer owned by worker")
	}
	if dead {
		now := time.Now()
		if _, err := tx.ExecContext(ctx, `UPDATE build_run
			SET status = 'failed', finished_at = ?,
			    cleanup_pending=(job_name<>'' OR provider='kube_job'),cleanup_retry_at=CURRENT_TIMESTAMP,
			    duration_seconds = TIMESTAMPDIFF(SECOND, COALESCE(started_at, queued_at), ?),
			    error_message = ?
			WHERE id = ? AND status IN ('queued', 'dispatching')`,
			now, now, lastError, buildRunID); err != nil {
			return err
		}
	} else {
		// ClaimCommand only selects queued submissions. Leaving a failed
		// attempt dispatching makes its pending retry permanently unclaimable.
		if _, err := tx.ExecContext(ctx, `UPDATE build_run SET status='queued',error_message=?
			WHERE id=? AND status='dispatching'`, lastError, buildRunID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// A process restart is not proof of execution failure. Reclaim the lease and
// inspect the original execution, even when the last request used its retry budget.
func (r *Repository) RequeueStaleCommands(ctx context.Context, staleBefore time.Time, maxAttempts int) (int64, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE build_command bc JOIN build_run br ON br.id=bc.build_run_id
		SET bc.status = CASE WHEN bc.command_type='submit' AND br.status IN ('running','success','failed','timeout','cancelled') THEN 'succeeded' ELSE 'pending' END,
		    bc.attempts=GREATEST(bc.attempts-1,0),bc.next_attempt_at = CURRENT_TIMESTAMP,
		    bc.locked_by = '', bc.locked_at = NULL,
		    bc.last_error = CASE WHEN bc.last_error IS NULL OR bc.last_error = '' THEN 'worker lease expired; recovering original execution' ELSE bc.last_error END
		WHERE bc.status = 'processing' AND bc.locked_at < ?`, staleBefore)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

// Keep capacity reserved when a remote submit/cancel outcome is unknown.
// Repeated probes do not rerun the build; the provider's own deadline remains
// responsible for stopping work even while the control plane is unavailable.
func (r *Repository) RetryUncertainCommand(ctx context.Context, id int64, workerID, reason string, delay time.Duration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE build_command SET status='pending',next_attempt_at=?,locked_by='',locked_at=NULL,last_error=? WHERE id=? AND status='processing' AND locked_by=?`, time.Now().Add(delay), reason, id, workerID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return errors.New("build command lease is no longer owned by worker")
	}
	_, err = tx.ExecContext(ctx, `UPDATE build_run br JOIN build_command bc ON bc.build_run_id=br.id SET br.error_message=? WHERE bc.id=? AND br.status IN ('dispatching','cancelling')`, reason, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeRetryAfter(delay time.Duration) time.Duration {
	if delay < time.Second {
		return time.Second
	}
	return delay
}
