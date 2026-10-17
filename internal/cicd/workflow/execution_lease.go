package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var errTaskCancelled = errors.New("workflow cancellation requested")
var errTaskLeaseLost = errors.New("workflow task lease lost")

type taskLeaseStore interface {
	RenewTaskLease(context.Context, int64, string, string, time.Duration) (string, error)
	ReleaseTaskLease(context.Context, int64, string, string) error
}

func (r *Repository) RenewTaskLease(ctx context.Context, runID int64, key, owner string, lease time.Duration) (string, error) {
	var status string
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		status, err = r.renewTaskLease(ctx, runID, key, owner, lease)
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || (mysqlErr.Number != 1213 && mysqlErr.Number != 1205) || attempt == 2 {
			return status, err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
	return status, err
}

func (r *Repository) renewTaskLease(ctx context.Context, runID int64, key, owner string, lease time.Duration) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&status); err != nil {
		return "", err
	}
	if status != RunRunning && status != RunPending {
		return status, nil
	}
	seconds := int(lease / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET lease_until=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL ? SECOND) WHERE workflow_run_id=? AND node_key=? AND locked_by=? AND status='running' AND lease_until>=CURRENT_TIMESTAMP`, seconds, runID, key, owner)
	if err != nil {
		return "", err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if n != 1 {
		// MySQL reports changed rows; renewing twice in one second can be a
		// no-op. Distinguish that from actual ownership loss under the same lock.
		var owned int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_task_run WHERE workflow_run_id=? AND node_key=? AND locked_by=? AND status='running' AND lease_until>=CURRENT_TIMESTAMP`, runID, key, owner).Scan(&owned); err != nil {
			return "", err
		}
		if owned != 1 {
			return "", errTaskLeaseLost
		}
	}
	return status, tx.Commit()
}

func (r *Repository) ReleaseTaskLease(ctx context.Context, runID int64, key, owner string) error {
	// Retain owner until recovery, but expire the lease now. Do not acknowledge a
	// submission whose provider result might have been lost with cancellation.
	_, err := r.db.ExecContext(ctx, `UPDATE workflow_task_run SET lease_until=DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 1 SECOND) WHERE workflow_run_id=? AND node_key=? AND locked_by=? AND status='running'`, runID, key, owner)
	return err
}

func (w *Worker) executeWithLease(ctx context.Context, owner string, executor TaskExecutor, request ExecutionRequest) (ExecutionResult, error) {
	if recoverer, ok := executor.(interface {
		Recover(context.Context, ExecutionRequest) (ExecutionResult, bool, error)
	}); ok {
		result, found, err := recoverer.Recover(ctx, request)
		if found || err != nil {
			return result, err
		}
	}
	var node Node
	if len(request.Task.NodeSnapshot) > 0 && json.Unmarshal(request.Task.NodeSnapshot, &node) != nil {
		return ExecutionResult{}, fmt.Errorf("invalid node snapshot")
	}
	timeout := time.Hour
	if node.TimeoutSeconds > 0 {
		timeout = time.Duration(node.TimeoutSeconds) * time.Second
	}
	if request.Task.StartedAt != nil {
		timeout -= time.Since(*request.Task.StartedAt)
	}
	if timeout <= 0 {
		return ExecutionResult{}, fmt.Errorf("node execution timed out before recovery")
	}
	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	store, ok := w.store.(taskLeaseStore)
	if !ok {
		return executor.Execute(taskCtx, request)
	}
	check := func() error {
		checkCtx, stop := context.WithTimeout(ctx, 5*time.Second)
		defer stop()
		status, err := store.RenewTaskLease(checkCtx, request.Run.ID, request.Task.NodeKey, owner, w.lease)
		if err != nil {
			return fmt.Errorf("%w: %v", errTaskLeaseLost, err)
		}
		if status == RunCancelling || status == RunCancelled {
			return errTaskCancelled
		}
		if status != RunPending && status != RunRunning {
			return errTaskLeaseLost
		}
		return nil
	}
	if err := check(); err != nil {
		return ExecutionResult{}, err
	}
	interval := w.lease / 3
	if interval > 2*time.Second {
		interval = 2 * time.Second
	}
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	done := make(chan struct{})
	observed := make(chan error, 1)
	go func() {
		timer := time.NewTicker(interval)
		defer timer.Stop()
		for {
			select {
			case <-done:
				observed <- nil
				return
			case <-taskCtx.Done():
				observed <- nil
				return
			case <-timer.C:
				if err := check(); err != nil {
					cancel()
					observed <- err
					return
				}
			}
		}
	}()
	result, err := executor.Execute(taskCtx, request)
	close(done)
	if leaseErr := <-observed; leaseErr != nil {
		return result, leaseErr
	}
	if errors.Is(err, ErrSubmissionUncertain) {
		return result, err
	}
	if result.ProviderRunID != "" && result.Status == RunRunning {
		return result, err
	}
	if taskCtx.Err() != nil {
		return result, fmt.Errorf("node execution interrupted: %w", taskCtx.Err())
	}
	return result, err
}
