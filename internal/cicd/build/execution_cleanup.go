package build

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// The completion transaction queues cleanup. A crash after saving logs or
// after releasing remote resources can therefore be retried idempotently.
func (r *Repository) FindCleanupRun(ctx context.Context) (*BuildRun, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM build_run WHERE cleanup_pending=1 AND (cleanup_retry_at IS NULL OR cleanup_retry_at<=CURRENT_TIMESTAMP) AND status IN ('success','failed','timeout','cancelled') ORDER BY cleanup_retry_at,id LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindRunByID(ctx, id)
}

func (r *Repository) FinishCleanup(ctx context.Context, id int64, success bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET cleanup_pending=?,cleanup_retry_at=? WHERE id=?`, !success, time.Now().Add(time.Minute), id)
	return err
}

func (r *Repository) ObserveTerminal(ctx context.Context, id int64) (time.Time, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET terminal_observed_at=COALESCE(terminal_observed_at,CURRENT_TIMESTAMP) WHERE id=?`, id)
	if err != nil {
		return time.Time{}, err
	}
	var at time.Time
	err = r.db.QueryRowContext(ctx, `SELECT terminal_observed_at FROM build_run WHERE id=?`, id).Scan(&at)
	return at, err
}

func (r *Reconciler) CleanupOne(ctx context.Context) (bool, error) {
	store, ok := r.store.(interface {
		FindCleanupRun(context.Context) (*BuildRun, error)
		FinishCleanup(context.Context, int64, bool) error
	})
	if !ok {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	run, err := store.FindCleanupRun(ctx)
	if err != nil || run == nil {
		return false, err
	}
	provider, err := r.providers.Get(run.Provider)
	if err == nil {
		if finalizer, ok := provider.(ProviderFinalizer); ok {
			err = finalizer.Finalize(ctx, ProviderExecution{BuildRunID: run.ID, JobName: run.JobName, ExternalID: run.JobName, PodName: run.PodName, NodeName: run.NodeName})
		} else {
			err = fmt.Errorf("build provider %q does not support durable cleanup acknowledgement", run.Provider)
		}
	}
	if err != nil {
		_, _ = r.store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "cleanup", LogText: "[system] Cleanup will retry: " + err.Error(), LineCount: 1})
	} else {
		// Replace an old retry diagnostic once the idempotent cleanup succeeds.
		if _, saveErr := r.store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "cleanup", LogText: "[system] 资源回收已按保留策略处理，已保存的日志和构建记录继续保留。", LineCount: 1}); saveErr != nil {
			return true, saveErr
		}
	}
	if saveErr := store.FinishCleanup(ctx, run.ID, err == nil); saveErr != nil {
		return true, saveErr
	}
	return true, err
}
