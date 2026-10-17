package build

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type CommandStore interface {
	ClaimCommand(context.Context, string) (*BuildCommand, error)
	CompleteCommand(context.Context, int64, string) error
	FailCommand(context.Context, int64, string, string, int, time.Duration) error
}

// ExecutionResolver is implemented by the worker's data access layer. It is
// deliberately separate from the provider so credentials never enter the
// command queue or provider registry by accident.
type ExecutionResolver interface {
	BuildRequest(context.Context, int64) (ProviderBuildRequest, error)
	Execution(context.Context, int64) (ProviderExecution, error)
	MarkSubmitted(context.Context, int64, ProviderExecution) error
	MarkCancelled(context.Context, int64) error
}

type Worker struct {
	store            CommandStore
	providers        *ProviderRegistry
	resolver         ExecutionResolver
	maxAttempts      int
	retryAfter       time.Duration
	operationTimeout time.Duration
	logStore         RunningBuildStore
	archive          LogArchive
}

func NewWorker(store CommandStore, providers *ProviderRegistry, resolver ExecutionResolver, maxAttempts int, retryAfter time.Duration) *Worker {
	if maxAttempts < 1 {
		maxAttempts = 3
	}
	return &Worker{store: store, providers: providers, resolver: resolver, maxAttempts: maxAttempts, retryAfter: normalizeRetryAfter(retryAfter), operationTimeout: 20 * time.Second}
}

func (w *Worker) WithLogPersistence(store RunningBuildStore, archive LogArchive) *Worker {
	w.logStore, w.archive = store, archive
	return w
}

func (w *Worker) WithOperationTimeout(timeout time.Duration) *Worker {
	if timeout > 0 {
		w.operationTimeout = timeout
	}
	return w
}

// ProcessOne handles at most one command. The bool is false when the queue is
// empty, allowing callers to choose a low-cost poll interval.
func (w *Worker) ProcessOne(ctx context.Context, workerID string) (bool, error) {
	if w.store == nil || w.providers == nil || w.resolver == nil {
		return false, errors.New("build worker is not configured")
	}
	command, err := w.store.ClaimCommand(ctx, workerID)
	if err != nil {
		return false, err
	}
	if command == nil {
		return false, nil
	}
	opCtx, cancel := context.WithTimeout(ctx, w.operationTimeout)
	defer cancel()

	provider, err := w.providers.Get(command.Provider)
	if err != nil {
		return true, w.failCommand(ctx, workerID, command, err)
	}

	switch command.CommandType {
	case CommandSubmit:
		var execution ProviderExecution
		var found bool
		if recovery, ok := provider.(ProviderRecoverer); ok {
			execution, found, err = recovery.Recover(opCtx, command.BuildRunID, command.IdempotencyKey)
		}
		if err == nil && !found {
			var request ProviderBuildRequest
			request, err = w.resolver.BuildRequest(opCtx, command.BuildRunID)
			if err == nil {
				request.IdempotencyKey = command.IdempotencyKey
				execution, err = provider.Submit(opCtx, request)
			}
		}
		if err == nil {
			err = w.resolver.MarkSubmitted(opCtx, command.BuildRunID, execution)
			if err != nil {
				err = &ExecutionUncertainError{Cause: err}
			}
		}
		if err != nil {
			return true, w.failCommand(ctx, workerID, command, err)
		}
	case CommandCancel:
		execution, err := w.resolver.Execution(opCtx, command.BuildRunID)
		if err == nil && execution.JobName == "" && execution.ExternalID == "" {
			if recovery, ok := provider.(ProviderRecoverer); ok {
				execution, _, err = recovery.Recover(opCtx, command.BuildRunID, fmt.Sprintf("build:%d:submit", command.BuildRunID))
			}
		}
		if err == nil && (execution.ExternalID != "" || execution.JobName != "") {
			// Capture available output before cancellation removes the execution.
			if observer, ok := provider.(ProviderObserver); ok && w.logStore != nil {
				logCtx, stopLogs := context.WithTimeout(opCtx, 5*time.Second)
				state, observeErr := observer.Status(logCtx, execution)
				stopLogs()
				if observeErr != nil {
					state.DiagnosticText = "[system] Cancellation log collection: " + observeErr.Error()
				}
				_, err = persistBuildLogs(opCtx, w.logStore, nil, &BuildRun{ID: command.BuildRunID}, state)
			}
			if err == nil {
				err = provider.Cancel(opCtx, execution)
			}
		}
		if err == nil {
			err = w.resolver.MarkCancelled(ctx, command.BuildRunID)
		}
		if err != nil {
			return true, w.failCommand(ctx, workerID, command, err)
		}
	default:
		return true, w.failCommand(ctx, workerID, command, fmt.Errorf("unsupported build command %q", command.CommandType))
	}

	if err := w.store.CompleteCommand(ctx, command.ID, workerID); err != nil {
		return true, err
	}
	return true, nil
}

func (w *Worker) failCommand(ctx context.Context, workerID string, command *BuildCommand, cause error) error {
	log.Printf("build command failed: run=%d command=%d type=%s provider=%s attempt=%d: %v", command.BuildRunID, command.ID, command.CommandType, command.Provider, command.Attempts, cause)
	var uncertain *ExecutionUncertainError
	if errors.As(cause, &uncertain) || command.CommandType == CommandCancel {
		if retry, ok := w.store.(interface {
			RetryUncertainCommand(context.Context, int64, string, string, time.Duration) error
		}); ok {
			return retry.RetryUncertainCommand(ctx, command.ID, workerID, cause.Error(), w.retryDelay(command.Attempts))
		}
	}
	if err := w.store.FailCommand(ctx, command.ID, workerID, cause.Error(), w.maxAttempts, w.retryDelay(command.Attempts)); err != nil {
		return err
	}
	return nil
}

func (w *Worker) retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	delay := w.retryAfter * time.Duration(1<<uint(attempt-1))
	if delay > time.Minute {
		delay = time.Minute
	}
	return delay
}

func (w *Worker) Run(ctx context.Context, workerID string, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	for {
		processed, err := w.ProcessOne(ctx, workerID)
		if err != nil {
			return err
		}
		if processed {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
