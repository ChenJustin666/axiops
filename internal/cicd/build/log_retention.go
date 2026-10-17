package build

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) FindExpiredLog(ctx context.Context) (*BuildLog, error) {
	var l BuildLog
	err := r.db.QueryRowContext(ctx, `SELECT bl.id,bl.build_run_id,COALESCE(bl.log_url,'') FROM build_log bl
		JOIN build_run br ON br.id=bl.build_run_id JOIN build_platform_settings s ON s.id=1
		WHERE bl.log_expired=0 AND br.status IN ('success','failed','timeout','cancelled') AND br.cleanup_pending=0
		AND COALESCE(JSON_EXTRACT(s.execution_policy,'$.log_retention_days'),0)>0
		AND TIMESTAMPDIFF(DAY,br.finished_at,CURRENT_TIMESTAMP)>=JSON_EXTRACT(s.execution_policy,'$.log_retention_days')
		ORDER BY br.finished_at,bl.id LIMIT 1`).Scan(&l.ID, &l.BuildRunID, &l.LogURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &l, err
}

func (r *Repository) ExpireLog(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE build_log SET log_expired=1,log_text='[system] 日志已按平台保留期限清理；构建状态和产物记录仍保留。',log_url='',line_count=1,truncated=1 WHERE id=?`, id)
	return err
}

func (r *Reconciler) ExpireLogOne(ctx context.Context) (bool, error) {
	store, ok := r.store.(interface {
		FindExpiredLog(context.Context) (*BuildLog, error)
		ExpireLog(context.Context, int64) error
	})
	if !ok {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	entry, err := store.FindExpiredLog(ctx)
	if err != nil || entry == nil {
		return false, err
	}
	if strings.HasPrefix(entry.LogURL, "s3://") {
		remover, ok := r.archive.(LogArchiveRemover)
		if !ok {
			return true, fmt.Errorf("log retention needs the original archive connection for build %d", entry.BuildRunID)
		}
		if err = remover.Delete(ctx, entry.LogURL); err != nil {
			return true, err
		}
	}
	return true, store.ExpireLog(ctx, entry.ID)
}
