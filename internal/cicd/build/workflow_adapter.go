package build

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

// WorkflowAdapter keeps workflow orchestration on top of the existing durable
// build queue instead of submitting a second provider job directly.
type WorkflowAdapter struct{ logic *Logic }

func NewWorkflowAdapter(logic *Logic) *WorkflowAdapter { return &WorkflowAdapter{logic: logic} }

func (a *WorkflowAdapter) TriggerFromWebhook(ctx context.Context, req *WebhookBuildRequest) (int64, error) {
	if a == nil || a.logic == nil {
		return 0, fmt.Errorf("build workflow adapter is not configured")
	}
	return a.logic.TriggerFromWebhook(ctx, req)
}

func (a *WorkflowAdapter) TriggerBuild(ctx context.Context, request workflow.BuildTriggerRequest, userID int64) (int64, error) {
	if a == nil || a.logic == nil {
		return 0, fmt.Errorf("build workflow adapter is not configured")
	}
	if request.TriggerKey != "" {
		existing, err := a.logic.repo.FindRunByTriggerKey(ctx, request.ServiceID, request.TriggerKey)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", workflow.ErrSubmissionUncertain, err)
		}
		if existing != nil {
			return existing.ID, nil
		}
	}
	if request.RetryOfBuildID > 0 {
		return a.retryBuild(ctx, request, userID)
	}
	if request.Test != nil {
		return a.triggerTest(ctx, request, userID)
	}
	id, code := a.logic.Trigger(ctx, &TriggerBuildRequest{
		WorkflowRunID: request.WorkflowRunID, WorkflowNodeKey: request.NodeKey,
		ServiceID: request.ServiceID, Branch: request.Branch, CommitID: request.CommitID, TriggerType: "workflow", TriggerKey: request.TriggerKey,
	}, userID)
	if code != errcode.CodeSuccess {
		if code == errcode.CodeDBError || ctx.Err() != nil {
			return 0, fmt.Errorf("%w: trigger build", workflow.ErrSubmissionUncertain)
		}
		return 0, fmt.Errorf("trigger build: %s", errcode.Message(code))
	}
	return id, nil
}

// The workflow API already authorizes cancellation. This internal adapter only
// accepts the build owned by that exact run/node and never starts a new build.
func (a *WorkflowAdapter) CancelWorkflowBuild(ctx context.Context, request workflow.BuildCancellationRequest) (workflow.BuildState, error) {
	if a == nil || a.logic == nil {
		return workflow.BuildState{}, fmt.Errorf("build workflow adapter is not configured")
	}
	var run *BuildRun
	var err error
	if request.BuildID > 0 {
		run, err = a.logic.repo.FindRunByID(ctx, request.BuildID)
	} else {
		run, err = a.logic.repo.FindRunByTriggerKey(ctx, request.ServiceID, request.TriggerKey)
	}
	if err != nil {
		return workflow.BuildState{}, err
	}
	if run == nil {
		return workflow.BuildState{Status: StatusCancelled}, nil
	}
	if run.ProjectID != request.ProjectID || run.ServiceID != request.ServiceID {
		return workflow.BuildState{}, fmt.Errorf("build does not belong to cancelling workflow service")
	}
	if run.Status == StatusSuccess || run.Status == StatusFailed || run.Status == StatusTimeout || run.Status == StatusCancelled {
		return workflow.BuildState{Status: run.Status}, nil
	}
	if run.TriggerType != "workflow" || (run.TriggerKey != "" && run.TriggerKey != request.TriggerKey) {
		return workflow.BuildState{}, fmt.Errorf("build does not belong to cancelling workflow node")
	}
	if err := a.logic.repo.RequestCancel(ctx, run.ID); err != nil {
		if !errors.Is(err, ErrInvalidTransition) {
			return workflow.BuildState{}, err
		}
		// A terminal transition raced cancellation. Observe it on the next poll.
	}
	return workflow.BuildState{Status: StatusCancelling}, nil
}

func (a *WorkflowAdapter) FindWorkflowBuild(ctx context.Context, serviceID int64, key string) (int64, error) {
	if a == nil || a.logic == nil {
		return 0, fmt.Errorf("build workflow adapter is not configured")
	}
	run, err := a.logic.repo.FindRunByTriggerKey(ctx, serviceID, key)
	if err != nil || run == nil {
		return 0, err
	}
	return run.ID, nil
}

func (a *WorkflowAdapter) BuildState(ctx context.Context, buildID int64) (workflow.BuildState, error) {
	if a == nil || a.logic == nil {
		return workflow.BuildState{}, fmt.Errorf("build workflow adapter is not configured")
	}
	run, err := a.logic.repo.FindRunByID(ctx, buildID)
	if err != nil {
		return workflow.BuildState{}, err
	}
	if run == nil {
		return workflow.BuildState{}, fmt.Errorf("build run %d not found", buildID)
	}
	return workflow.BuildState{Status: run.Status, ErrorMessage: run.ErrorMessage, Retryable: validSourceCommit(run.GitCommit) && run.ConfigSnapshot != nil}, nil
}
