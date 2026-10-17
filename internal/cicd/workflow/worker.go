package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TaskStore interface {
	ListActiveRuns(context.Context, int) ([]WorkflowRun, error)
	ClaimTask(context.Context, int64, string, string, time.Duration) (bool, error)
	CompleteTask(context.Context, int64, string, string, string, string) error
	MarkTaskSubmitted(context.Context, int64, string, string, ExecutionResult) error
	CompleteSubmittedTask(context.Context, int64, string, string, string) error
}

type ExecutionRequest struct {
	Run            WorkflowRun
	Task           TaskRun
	ReportProgress func(string) error
}

type TaskExecutor interface {
	NodeType() string
	Execute(context.Context, ExecutionRequest) (ExecutionResult, error)
}

type TaskReconciler interface {
	Reconcile(context.Context, ExecutionRequest) (ExecutionResult, error)
}

// TaskProgressStore keeps deployment evidence separate from error status.
type TaskProgressStore interface {
	AppendTaskLog(context.Context, int64, string, string) error
}

type ExecutionResult struct {
	Retryable     bool   `json:"-"`
	Status        string `json:"status"`
	ProviderRunID string `json:"provider_run_id"`
	LogRef        string `json:"log_ref"`
	LogText       string `json:"log_text"`
	ErrorMessage  string `json:"error_message"`
}

type ExecutorRegistry struct {
	executors map[string]TaskExecutor
}

func NewExecutorRegistry(executors ...TaskExecutor) (*ExecutorRegistry, error) {
	registry := &ExecutorRegistry{executors: make(map[string]TaskExecutor, len(executors))}
	for _, executor := range executors {
		if executor == nil || executor.NodeType() == "" {
			return nil, fmt.Errorf("task executor and node type are required")
		}
		if _, exists := registry.executors[executor.NodeType()]; exists {
			return nil, fmt.Errorf("duplicate task executor %q", executor.NodeType())
		}
		registry.executors[executor.NodeType()] = executor
	}
	return registry, nil
}

func (r *ExecutorRegistry) Get(nodeType string) (TaskExecutor, bool) {
	executor, ok := r.executors[nodeType]
	return executor, ok
}

type Worker struct {
	store     TaskStore
	executors *ExecutorRegistry
	lease     time.Duration
	batchSize int
}

func NewWorker(store TaskStore, executors *ExecutorRegistry, lease time.Duration, batchSize int) *Worker {
	if lease <= 0 {
		lease = 2 * time.Minute
	}
	if batchSize <= 0 {
		batchSize = 20
	}
	return &Worker{store: store, executors: executors, lease: lease, batchSize: batchSize}
}

// Tick claims and executes one batch. ClaimTask is optimistic, so multiple
// workers can inspect a run without executing one node twice.
func (w *Worker) Tick(ctx context.Context, workerID string) error {
	if w == nil || w.store == nil || w.executors == nil || workerID == "" {
		return fmt.Errorf("worker, store, executors, and worker id are required")
	}
	runs, err := w.store.ListActiveRuns(ctx, w.batchSize)
	if err != nil {
		return err
	}
	var failures []error
	for _, run := range runs {
		if err := w.processRun(ctx, workerID, run); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (w *Worker) processRun(ctx context.Context, workerID string, run WorkflowRun) error {
	var failures []error
	if run.Status == RunCancelling {
		return w.reconcileCancellation(ctx, run)
	}
	if err := w.reconcileSubmitted(ctx, run); err != nil {
		return err
	}
	plan, err := Plan(run)
	if err != nil {
		return err
	}
	if hasOrderedDeployments(run.Tasks) && len(plan.Skippable) > 0 {
		store, ok := w.store.(interface {
			SkipTask(context.Context, int64, string) error
		})
		if !ok {
			return fmt.Errorf("ordered workflow task store cannot skip failed dependencies")
		}
		for _, nodeKey := range plan.Skippable {
			if err := store.SkipTask(ctx, run.ID, nodeKey); err != nil {
				return err
			}
		}
	}
	runnable := append([]string(nil), plan.Runnable...)
	for _, task := range run.Tasks {
		if task.Status == RunRunning && task.ProviderRunID == "" && task.LeaseUntil != nil && task.LeaseUntil.Before(time.Now()) {
			runnable = append(runnable, task.NodeKey)
		}
	}
	for _, nodeKey := range runnable {
		task, ok := taskByKey(run.Tasks, nodeKey)
		if !ok {
			return fmt.Errorf("runnable task %q is missing from run %d", nodeKey, run.ID)
		}
		if task.NextRetryAt != nil && task.NextRetryAt.After(time.Now()) {
			continue
		}
		claimed, err := w.store.ClaimTask(ctx, run.ID, nodeKey, workerID, w.lease)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		if task.NodeType == NodeYAMLDeploy || task.NodeType == NodeHelmDeploy || task.NodeType == NodeHostDeploy || task.NodeType == NodeYAMLRollback || task.NodeType == NodeHelmRollback || task.NodeType == NodeWorkloadRestart {
			message := "[start] 部署任务已开始，正在核对目标环境"
			if task.Status == RunRunning {
				message = "[recover] 执行租约已过期，正在恢复原部署任务"
			}
			if task.NodeType == NodeWorkloadRestart {
				message = "[start] 滚动重启任务已开始，正在核对目标环境"
				if task.Status == RunRunning {
					message = "[recover] 执行租约已过期，正在核对原重启请求"
				}
			}
			if err := w.recordTaskLog(ctx, run.ID, nodeKey, message); err != nil {
				return err
			}
		}
		if err := w.validateEnvironmentTarget(ctx, run, task); err != nil {
			if completeErr := w.store.CompleteTask(ctx, run.ID, nodeKey, workerID, RunFailed, err.Error()); completeErr != nil {
				return completeErr
			}
			continue
		}
		executor, exists := w.executors.Get(task.NodeType)
		if !exists {
			_ = w.store.CompleteTask(ctx, run.ID, nodeKey, workerID, RunFailed, "no executor registered for node type "+task.NodeType)
			continue
		}
		request := ExecutionRequest{Run: run, Task: task}
		request.ReportProgress = func(message string) error {
			logCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			defer stop()
			return w.recordTaskLog(logCtx, run.ID, nodeKey, message)
		}
		result, err := w.executeWithLease(ctx, workerID, executor, request)
		if errors.Is(err, ErrSubmissionUncertain) || ctx.Err() != nil {
			// Leave the lease to expire. Recovery checks the same durable key;
			// neither claim a definite failure nor create a new retry identity.
			_ = w.recordTaskLog(ctx, run.ID, nodeKey, "[recover] 提交结果待确认，将使用原请求标识恢复，不重复提交操作")
			failures = append(failures, err)
			continue
		}
		if errors.Is(err, errTaskLeaseLost) {
			_ = w.recordTaskLog(ctx, run.ID, nodeKey, "[recover] 执行租约暂不可用，等待核对后恢复原任务")
			failures = append(failures, err)
			continue
		}
		if errors.Is(err, errTaskCancelled) {
			if result.ProviderRunID != "" {
				if saveErr := w.store.MarkTaskSubmitted(ctx, run.ID, nodeKey, workerID, result); saveErr != nil {
					failures = append(failures, saveErr)
				}
			} else if leases, ok := w.store.(taskLeaseStore); ok {
				if releaseErr := leases.ReleaseTaskLease(ctx, run.ID, nodeKey, workerID); releaseErr != nil {
					failures = append(failures, releaseErr)
				}
			}
			_ = w.recordTaskLog(ctx, run.ID, nodeKey, "[cancel] 收到取消请求，停止当前执行并核对已提交操作")
			continue
		}
		if logErr := w.recordTaskLog(ctx, run.ID, task.NodeKey, result.LogText); logErr != nil {
			return logErr
		}
		if err != nil {
			if logErr := w.recordTaskLog(ctx, run.ID, task.NodeKey, "[failed] "+safeDiagnostic(err.Error())); logErr != nil {
				return logErr
			}
			if completeErr := w.store.CompleteTask(ctx, run.ID, nodeKey, workerID, RunFailed, err.Error()); completeErr != nil {
				return completeErr
			}
			continue
		}
		switch result.Status {
		case RunRunning:
			if result.ProviderRunID == "" {
				return fmt.Errorf("executor %q returned running without provider run id", task.NodeType)
			}
			if err := w.store.MarkTaskSubmitted(ctx, run.ID, nodeKey, workerID, result); err != nil {
				return err
			}
		case RunSucceeded, "":
			if err := w.store.CompleteTask(ctx, run.ID, nodeKey, workerID, RunSucceeded, ""); err != nil {
				return err
			}
		default:
			return fmt.Errorf("executor %q returned unsupported status %q", task.NodeType, result.Status)
		}
	}
	return errors.Join(failures...)
}

func (w *Worker) validateEnvironmentTarget(ctx context.Context, run WorkflowRun, task TaskRun) error {
	if task.NodeType != NodeYAMLDeploy && task.NodeType != NodeHelmDeploy && task.NodeType != NodeHostDeploy && task.NodeType != NodeYAMLRollback && task.NodeType != NodeHelmRollback && task.NodeType != NodeWorkloadRestart {
		return nil
	}
	var snapshot struct {
		Target *EnvironmentTarget `json:"target"`
	}
	if len(run.ContextSnapshot) > 0 {
		if err := json.Unmarshal(run.ContextSnapshot, &snapshot); err != nil {
			return fmt.Errorf("invalid environment snapshot")
		}
	}
	// Existing runs predate target snapshots; their deployment resolvers retain validation.
	if snapshot.Target == nil {
		return nil
	}
	reader, ok := w.store.(interface {
		FindEnvironmentTarget(context.Context, int64) (*EnvironmentTarget, error)
	})
	if !ok {
		return fmt.Errorf("environment target validator is unavailable")
	}
	current, err := reader.FindEnvironmentTarget(ctx, run.EnvironmentID)
	if err != nil {
		return fmt.Errorf("cannot verify deployment environment: %w", err)
	}
	if current == nil || *current != *snapshot.Target || current.ProjectID != run.ProjectID || current.Status != "active" || current.BindingStatus != "validated" {
		return fmt.Errorf("目标环境在提交后发生变化；已阻止部署，请检查集群、命名空间或主机组后重新执行")
	}
	return nil
}

func (w *Worker) reconcileSubmitted(ctx context.Context, run WorkflowRun) error {
	for _, task := range run.Tasks {
		if task.Status != RunRunning || task.ProviderRunID == "" || task.LockedBy != "" {
			continue
		}
		executor, exists := w.executors.Get(task.NodeType)
		if !exists {
			return fmt.Errorf("no executor registered for submitted node type %q", task.NodeType)
		}
		reconciler, ok := executor.(TaskReconciler)
		if !ok {
			return fmt.Errorf("executor %q cannot reconcile asynchronous tasks", task.NodeType)
		}
		result, err := reconciler.Reconcile(ctx, ExecutionRequest{Run: run, Task: task})
		if err != nil {
			if logErr := w.recordTaskLog(ctx, run.ID, task.NodeKey, "[check] status read failed; retrying: "+safeDiagnostic(err.Error())); logErr != nil {
				return logErr
			}
			return err
		}
		if err := w.recordTaskLog(ctx, run.ID, task.NodeKey, result.LogText); err != nil {
			return err
		}
		if result.Status == RunRunning {
			continue
		}
		if result.Status != RunSucceeded && result.Status != RunFailed && result.Status != RunCancelled {
			return fmt.Errorf("executor %q returned unsupported reconcile status %q", task.NodeType, result.Status)
		}
		if result.Status == RunFailed && result.Retryable && retryAllowed(task) {
			if store, ok := w.store.(taskRetryStore); ok {
				scheduled, err := store.ScheduleTaskRetry(ctx, run, task, result.ErrorMessage)
				if err != nil {
					return err
				}
				if scheduled {
					continue
				}
			}
		}
		if err := w.completeObservedTask(ctx, run.ID, task, result); err != nil {
			// A stale run snapshot can be reconciled by another lane after the
			// provider has already reached a terminal state. The durable update
			// is idempotent; do not turn that harmless race into a worker tick error.
			if strings.Contains(err.Error(), "is no longer running") {
				continue
			}
			return err
		}
	}
	return nil
}

func (w *Worker) completeObservedTask(ctx context.Context, runID int64, task TaskRun, result ExecutionResult) error {
	if store, ok := w.store.(interface {
		CompleteObservedTask(context.Context, int64, TaskRun, string, string) error
	}); ok {
		return store.CompleteObservedTask(ctx, runID, task, result.Status, result.ErrorMessage)
	}
	return w.store.CompleteSubmittedTask(ctx, runID, task.NodeKey, result.Status, result.ErrorMessage)
}

func (w *Worker) recordTaskLog(ctx context.Context, runID int64, nodeKey, text string) error {
	if text == "" {
		return nil
	}
	if store, ok := w.store.(TaskProgressStore); ok {
		return store.AppendTaskLog(ctx, runID, nodeKey, text)
	}
	return nil
}

func taskByKey(tasks []TaskRun, key string) (TaskRun, bool) {
	for _, task := range tasks {
		if task.NodeKey == key {
			return task, true
		}
	}
	return TaskRun{}, false
}
