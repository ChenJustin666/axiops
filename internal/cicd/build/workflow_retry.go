package build

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

func validSourceCommit(commit string) bool {
	if len(commit) != 40 && len(commit) != 64 {
		return false
	}
	_, err := hex.DecodeString(commit)
	return err == nil
}

func (r *Repository) UpdateResolvedCommit(ctx context.Context, id int64, previous, commit string) error {
	if !validSourceCommit(commit) || (previous != "" && !strings.EqualFold(previous, commit)) {
		return fmt.Errorf("checked-out commit differs from requested commit")
	}
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET git_commit=?,git_commit_short=? WHERE id=? AND git_commit=?`, commit, commit[:8], id, previous)
	return err
}

// Freeze the original inputs rather than using current service settings.
func workflowRetryRun(previous BuildRun, request workflow.BuildTriggerRequest, number int, userID int64) (*BuildRun, error) {
	if previous.ServiceID != request.ServiceID || previous.TriggerType != "workflow" || previous.Status != StatusFailed && previous.Status != StatusTimeout {
		return nil, fmt.Errorf("only a failed build of this workflow service can be retried")
	}
	baseKey := fmt.Sprintf("workflow:%d:%s", request.WorkflowRunID, request.NodeKey)
	if previous.TriggerKey != baseKey && !strings.HasPrefix(previous.TriggerKey, baseKey+":retry:") {
		return nil, fmt.Errorf("retry build belongs to a different workflow node")
	}
	if !validSourceCommit(previous.GitCommit) || previous.ConfigSnapshot == nil {
		return nil, fmt.Errorf("无法固定上次构建的 Commit 或配置快照，请检查代码检出记录后重新运行")
	}
	imageURL := previous.ImageURL
	if imageURL != "" {
		if colon := strings.LastIndex(imageURL, ":"); colon > strings.LastIndex(imageURL, "/") {
			imageURL = imageURL[:colon] + ":" + automaticImageTag(previous.GitBranch, number, time.Now())
		} else {
			return nil, fmt.Errorf("invalid original build image reference")
		}
	}
	return &BuildRun{ServiceID: previous.ServiceID, ProjectID: previous.ProjectID, BuildNumber: number, ConfigSnapshot: previous.ConfigSnapshot,
		WorkflowRunID: request.WorkflowRunID, WorkflowNodeKey: request.NodeKey, TriggerType: "workflow", TriggerUserID: userID, TriggerKey: request.TriggerKey,
		Status: StatusQueued, Provider: previous.Provider, GitBranch: previous.GitBranch, GitCommit: previous.GitCommit, GitCommitShort: previous.GitCommit[:8],
		ImageURL: imageURL}, nil
}

func (a *WorkflowAdapter) retryBuild(ctx context.Context, request workflow.BuildTriggerRequest, userID int64) (int64, error) {
	previous, err := a.logic.repo.FindRunByID(ctx, request.RetryOfBuildID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	if previous == nil {
		return 0, fmt.Errorf("original build was not found")
	}
	allowed, err := a.logic.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "run", ProjectID: previous.ProjectID, ServiceID: previous.ServiceID})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	if !allowed {
		return 0, fmt.Errorf("build permission denied")
	}
	number, err := a.logic.repo.AllocateBuildNumber(ctx, previous.ServiceID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	run, err := workflowRetryRun(*previous, request, number, userID)
	if err != nil {
		return 0, err
	}
	id, err := a.logic.repo.CreateRunWithSubmitCommand(ctx, run)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
	}
	return id, nil
}
