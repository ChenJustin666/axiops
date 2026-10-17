package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type TaskCanceller interface {
	Cancel(context.Context, ExecutionRequest) (ExecutionResult, error)
}

func defaultFailureMessage(message string) string {
	if message == "" {
		return "工作流节点执行失败"
	}
	return message
}

type cancellationStore interface {
	FinishCancellation(context.Context, int64) error
	CompleteCancellingTask(context.Context, int64, string) error
	RecoverTaskSubmission(context.Context, WorkflowRun, TaskRun) (string, error)
}

func (w *Worker) reconcileCancellation(ctx context.Context, run WorkflowRun) error {
	store, ok := w.store.(cancellationStore)
	if !ok {
		return fmt.Errorf("workflow cancellation store unavailable")
	}
	var failures []error
	for _, task := range run.Tasks {
		if task.Status != RunRunning {
			continue
		}
		if task.LockedBy != "" && task.LeaseUntil != nil && task.LeaseUntil.After(time.Now()) {
			continue
		}
		request := ExecutionRequest{Run: run, Task: task}
		executor, exists := w.executors.Get(task.NodeType)
		if !exists {
			failures = append(failures, fmt.Errorf("cannot confirm stop of node %s: executor unavailable", task.NodeKey))
			continue
		}
		var result ExecutionResult
		var err error
		if canceller, ok := executor.(TaskCanceller); ok {
			result, err = canceller.Cancel(ctx, request)
		} else {
			// Observe existing deployments instead of applying, deleting or rolling back
			// resources as a side effect of cancelling a pipeline.
			if request.Task.ProviderRunID == "" {
				request.Task.ProviderRunID, err = store.RecoverTaskSubmission(ctx, run, task)
			}
			if err == nil && request.Task.ProviderRunID != "" {
				reconciler, ok := executor.(TaskReconciler)
				if !ok {
					err = fmt.Errorf("cannot observe in-flight node %s", task.NodeKey)
				} else {
					result, err = reconciler.Reconcile(ctx, request)
				}
				result.LogText = "[cancel] 后续节点已停止；已提交部署等待状态确认，不删除资源或自动回滚\n" + result.LogText
			} else if err == nil {
				result = ExecutionResult{Status: RunCancelled, LogText: "[cancel] 节点已停止；执行进程异常退出时请结合资源状态确认既有变更"}
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("cancel %d/%s: %w", run.ID, task.NodeKey, err))
			_ = w.recordTaskLog(ctx, run.ID, task.NodeKey, "[cancel] 状态确认失败，将重试："+safeDiagnostic(err.Error()))
			continue
		}
		if err = w.recordTaskLog(ctx, run.ID, task.NodeKey, result.LogText); err != nil {
			failures = append(failures, err)
			continue
		}
		if result.Status == RunRunning || result.Status == RunPending {
			continue
		}
		if result.Status != RunSucceeded && result.Status != RunFailed && result.Status != RunCancelled {
			failures = append(failures, fmt.Errorf("invalid cancellation state %q", result.Status))
			continue
		}
		if result.ErrorMessage != "" {
			_ = w.recordTaskLog(ctx, run.ID, task.NodeKey, "[cancel] 执行结果："+safeDiagnostic(result.ErrorMessage))
		}
		if err = store.CompleteCancellingTask(ctx, run.ID, task.NodeKey); err != nil {
			failures = append(failures, err)
		}
	}
	if err := store.FinishCancellation(ctx, run.ID); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}

func (r *Repository) CompleteCancellingTask(ctx context.Context, runID int64, key string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE workflow_task_run task JOIN workflow_run run ON run.id=task.workflow_run_id
 SET task.status='cancelled',task.locked_by='',task.lease_until=NULL,task.finished_at=CURRENT_TIMESTAMP
 WHERE run.id=? AND run.status='cancelling' AND task.node_key=? AND task.status='running'
 AND (task.locked_by='' OR task.lease_until<CURRENT_TIMESTAMP)`, runID, key)
	return err
}

func (r *Repository) FinishCancellation(ctx context.Context, runID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&status); err != nil {
		return err
	}
	if status != RunCancelling {
		return nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE workflow_task_run SET status='cancelled',finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND status IN ('pending','pending_approval')`, runID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_run SET status=IF(error_message<>'','failed','cancelled'),finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='cancelling' AND NOT EXISTS(SELECT 1 FROM workflow_task_run WHERE workflow_run_id=? AND status='running')`, runID, runID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		if err = enqueueWorkflowTerminal(ctx, tx, runID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Find previously submitted operations after a crash, without submitting again.
func (r *Repository) RecoverTaskSubmission(ctx context.Context, run WorkflowRun, task TaskRun) (string, error) {
	var id int64
	var err error
	switch task.NodeType {
	case NodeYAMLDeploy:
		err = r.db.QueryRowContext(ctx, `SELECT id FROM deployment_snapshot WHERE workflow_run_id=? AND node_key=? AND provider=?`, run.ID, task.NodeKey, NodeYAMLDeploy).Scan(&id)
		if err == nil {
			return "yaml:" + workflowBuildKey(run, task), nil
		}
	case NodeYAMLRollback:
		var node Node
		var cfg YAMLRollbackConfig
		if json.Unmarshal(task.NodeSnapshot, &node) != nil || json.Unmarshal(node.Config, &cfg) != nil || cfg.SnapshotID <= 0 {
			return "", fmt.Errorf("invalid rollback snapshot")
		}
		return fmt.Sprintf("yaml-rollback:workflow:%d:%s:rollback:%d", run.ID, task.NodeKey, cfg.SnapshotID), nil
	case NodeHelmDeploy, NodeHelmRollback:
		err = r.db.QueryRowContext(ctx, `SELECT id FROM helm_execution WHERE workflow_run_id=? AND node_key=?`, run.ID, task.NodeKey).Scan(&id)
		if err == nil {
			return fmt.Sprintf("helm:%d", id), nil
		}
	case NodeHostDeploy:
		err = r.db.QueryRowContext(ctx, `SELECT id FROM host_deployment_command WHERE workflow_run_id=? AND task_key=? LIMIT 1`, run.ID, task.NodeKey).Scan(&id)
		if err == nil {
			return fmt.Sprintf("host:%d:%s", run.ID, task.NodeKey), nil
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return "", err
}
