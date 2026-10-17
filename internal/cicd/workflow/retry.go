package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type taskRetryStore interface {
	ScheduleTaskRetry(context.Context, WorkflowRun, TaskRun, string) (bool, error)
}

func retryAllowed(task TaskRun) bool {
	var node Node
	return task.NodeType == NodeBuild && json.Unmarshal(task.NodeSnapshot, &node) == nil && task.RetryCount < node.RetryLimit && !buildNodeTimedOut(task)
}

func retryDelay(count int) time.Duration {
	if count < 0 {
		count = 0
	}
	if count > 4 {
		count = 4
	}
	return time.Duration(10<<count) * time.Second
}

func (r *Repository) ScheduleTaskRetry(ctx context.Context, run WorkflowRun, task TaskRun, reason string) (bool, error) {
	if !retryAllowed(task) {
		return false, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var state string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, run.ID).Scan(&state); err != nil {
		return false, err
	}
	if state != RunRunning {
		return false, nil
	}
	history := append(append([]TaskRetry(nil), task.RetryHistory...), TaskRetry{ProviderRunID: task.ProviderRunID, ErrorMessage: truncateDiagnostic(safeDiagnostic(reason), 1000), FinishedAt: time.Now()})
	data, err := json.Marshal(history)
	if err != nil {
		return false, err
	}
	seconds := int(retryDelay(task.RetryCount) / time.Second)
	message := fmt.Sprintf("\n[retry] %s 失败：%s；%d 秒后进行第 %d 次重试，旧构建日志保留\n", task.ProviderRunID, truncateDiagnostic(safeDiagnostic(reason), 1000), seconds, task.RetryCount+1)
	result, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET status='pending',retry_count=retry_count+1,next_retry_at=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL ? SECOND),retry_history=?,provider_run_id='',log_ref='',error_message='',locked_by='',locked_at=NULL,lease_until=NULL,started_at=NULL,finished_at=NULL,log_text=RIGHT(CONCAT(COALESCE(log_text,''),?),262144) WHERE workflow_run_id=? AND node_key=? AND status='running' AND locked_by='' AND provider_run_id=? AND retry_count=?`, seconds, data, message, run.ID, task.NodeKey, task.ProviderRunID, task.RetryCount)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return true, nil
	} // Another observer already advanced this generation.
	return true, tx.Commit()
}
