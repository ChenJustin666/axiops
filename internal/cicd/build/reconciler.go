package build

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

type RunningBuildStore interface {
	FindNextRunningRun(context.Context) (*BuildRun, error)
	UpdateRunExecInfo(context.Context, int64, string, string, string) error
	UpdateRunOutputs(context.Context, int64, string, string, int64, string, string, int64) error
	UpdateRunVerification(context.Context, int64, OutputVerification) error
	CreateLog(context.Context, *BuildLog) (int64, error)
	TransitionRunStatus(context.Context, int64, string, string) error
}

type Reconciler struct {
	lastRunID int64
	store     RunningBuildStore
	providers *ProviderRegistry
	archive   LogArchive
	verifier  OutputVerifier
}

func (r *Reconciler) WithOutputVerifier(verifier OutputVerifier) *Reconciler {
	r.verifier = verifier
	return r
}

func NewReconciler(store RunningBuildStore, providers *ProviderRegistry) *Reconciler {
	return &Reconciler{store: store, providers: providers}
}

func (r *Reconciler) WithLogArchive(archive LogArchive) *Reconciler {
	r.archive = archive
	return r
}

func (r *Reconciler) ReconcileOne(ctx context.Context) (bool, error) {
	if r == nil || r.store == nil || r.providers == nil {
		return false, fmt.Errorf("build reconciler is not configured")
	}
	var run *BuildRun
	var err error
	if paged, ok := r.store.(interface {
		FindNextRunningRunAfter(context.Context, int64) (*BuildRun, error)
	}); ok {
		run, err = paged.FindNextRunningRunAfter(ctx, r.lastRunID)
	} else {
		run, err = r.store.FindNextRunningRun(ctx)
	}
	if err != nil || run == nil {
		return false, err
	}
	r.lastRunID = run.ID
	provider, err := r.providers.Get(run.Provider)
	if err != nil {
		return true, err
	}
	observer, ok := provider.(ProviderObserver)
	if !ok {
		return true, fmt.Errorf("build provider %q cannot observe execution state", run.Provider)
	}
	observeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	state, err := observer.Status(observeCtx, ProviderExecution{ExternalID: run.JobName, JobName: run.JobName, PodName: run.PodName, NodeName: run.NodeName})
	cancel()
	if err != nil {
		_, logErr := r.store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "system", LogText: "[system] Build status query failed: " + err.Error(), LineCount: 1})
		if logErr != nil {
			return true, fmt.Errorf("observe build: %v; persist diagnostic: %w", err, logErr)
		}
		if errors.Is(err, ErrExecutionMissing) {
			return true, r.store.TransitionRunStatus(ctx, run.ID, StatusFailed, err.Error())
		}
		return true, err
	}
	// A failed Pod query or expired Pod must not erase the last known identity.
	if state.PodName == "" {
		state.PodName = run.PodName
		state.NodeName = run.NodeName
		if state.Status == StatusRunning && state.ErrorMessage == "" {
			state.ErrorMessage = run.ErrorMessage
		}
	}
	if state.PodName != run.PodName || state.NodeName != run.NodeName {
		if err := r.store.UpdateRunExecInfo(ctx, run.ID, run.JobName, state.PodName, state.NodeName); err != nil {
			return true, err
		}
	}
	if state.Status == StatusTimeout && run.ErrorMessage != "" && !strings.Contains(state.ErrorMessage, run.ErrorMessage) {
		state.ErrorMessage += "; last observed wait reason: " + run.ErrorMessage
	}
	if state.ErrorMessage != run.ErrorMessage {
		log.Printf("[build.observe] run=%d job=%s status=%s reason=%s", run.ID, run.JobName, state.Status, state.ErrorMessage)
	}
	diagnostic := state.DiagnosticText
	if state.ErrorMessage != "" && !strings.Contains(diagnostic, state.ErrorMessage) {
		diagnostic += "[system] " + state.ErrorMessage + "\n"
	}
	state.DiagnosticText = diagnostic
	terminalObserved := time.Now()
	if IsTerminalStatus(state.Status) {
		if store, ok := r.store.(interface {
			ObserveTerminal(context.Context, int64) (time.Time, error)
		}); ok {
			terminalObserved, err = store.ObserveTerminal(ctx, run.ID)
			if err != nil {
				return true, err
			}
		}
	}
	if updater, ok := r.store.(interface {
		UpdateRunWaitReason(context.Context, int64, string) error
	}); ok {
		if err := updater.UpdateRunWaitReason(ctx, run.ID, state.ErrorMessage); err != nil {
			return true, err
		}
	}
	if _, err := persistBuildLogs(ctx, r.store, r.archive, run, state); err != nil {
		return true, err
	}
	if state.SourceCommit != "" && state.SourceCommit != run.GitCommit {
		if updater, ok := r.store.(interface {
			UpdateResolvedCommit(context.Context, int64, string, string) error
		}); ok {
			if err := updater.UpdateResolvedCommit(ctx, run.ID, run.GitCommit, state.SourceCommit); err != nil {
				return true, err
			}
		}
	}
	if state.GitChanges != nil {
		if saver, ok := r.store.(interface {
			SaveGitChanges(context.Context, BuildRun, *GitChangeSet) error
		}); ok {
			if err := saver.SaveGitChanges(ctx, *run, state.GitChanges); err != nil {
				return true, err
			}
		}
	}
	if IsTerminalStatus(state.Status) && state.LogsIncomplete && time.Since(terminalObserved) < 5*time.Minute {
		return true, nil // retry final container reads without deleting evidence
	}
	if state.ImageDigest != "" || state.ImageSize > 0 || state.ArtifactURL != "" || state.ArtifactSize > 0 {
		if err := r.store.UpdateRunOutputs(ctx, run.ID, run.ImageURL, state.ImageDigest, state.ImageSize, state.ArtifactURL, state.ArtifactSHA256, state.ArtifactSize); err != nil {
			return true, err
		}
	}
	if state.Status == StatusSuccess && r.verifier != nil {
		verifyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		verification, err := r.verifier.Verify(verifyCtx, *run, state)
		cancel()
		if err != nil {
			if time.Since(terminalObserved) < 5*time.Minute {
				_, saveErr := r.store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "verification", LogText: "[system] Retrying output verification: " + err.Error(), LineCount: 1})
				return true, saveErr
			}
			return true, r.store.TransitionRunStatus(ctx, run.ID, StatusFailed, "output verification could not complete within 5 minutes: "+err.Error())
		}
		if err := r.store.UpdateRunVerification(ctx, run.ID, verification); err != nil {
			return true, err
		}
		if verification.Failed() {
			return true, r.store.TransitionRunStatus(ctx, run.ID, StatusFailed, verification.Error)
		}
	}
	switch state.Status {
	case StatusRunning, StatusDispatching, StatusQueued:
		return true, nil
	case StatusSuccess, StatusFailed, StatusTimeout:
		return true, r.store.TransitionRunStatus(ctx, run.ID, state.Status, state.ErrorMessage)
	default:
		return true, fmt.Errorf("provider %q returned unsupported build status %q", run.Provider, state.Status)
	}
}

func countLogLines(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}
