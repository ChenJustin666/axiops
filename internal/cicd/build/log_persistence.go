package build

import (
	"context"
	"fmt"
	"time"
)

func persistBuildLogs(ctx context.Context, store RunningBuildStore, archive LogArchive, run *BuildRun, state ProviderState) (bool, error) {
	if state.DiagnosticText != "" {
		if _, err := store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "system", LogText: state.DiagnosticText, LineCount: countLogLines(state.DiagnosticText), StartedAt: run.StartedAt}); err != nil {
			return false, err
		}
	}
	if state.LogText == "" {
		return false, nil
	}
	finished := run.FinishedAt
	if IsTerminalStatus(state.Status) {
		now := time.Now()
		finished = &now
	}
	url := fmt.Sprintf("/api/v1/builds/%d/logs", run.ID)
	content := state.LogText
	archiveFailed := false
	if archive != nil {
		archiveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		archived, err := archive.Put(archiveCtx, fmt.Sprintf("projects/%d/builds/%d/build.log", run.ProjectID, run.ID), []byte(content))
		cancel()
		if err != nil {
			archiveFailed = true
			// A storage outage cannot hide a completed build. SQL is the fallback;
			// report its bounded tail instead of claiming full archival succeeded.
			content = inlineLogTail(content, 1<<20)
			warning := "[system] Object storage unavailable; collected logs saved in database (up to 1 MiB): " + err.Error()
			if _, saveErr := store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "archive", LogText: warning, LineCount: 1}); saveErr != nil {
				return true, saveErr
			}
		} else {
			url = archived
			content = inlineLogTail(content, 64<<10)
		}
	}
	_, err := store.CreateLog(ctx, &BuildLog{BuildRunID: run.ID, Stage: "build", LogURL: url, LogText: content, LineCount: countLogLines(state.LogText), Truncated: state.LogTruncated || len(content) < len(state.LogText), StartedAt: run.StartedAt, FinishedAt: finished})
	return archiveFailed, err
}
