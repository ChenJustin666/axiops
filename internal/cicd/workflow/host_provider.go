package workflow

import (
	"context"
	"fmt"

	"github.com/ChenJustin666/axiops/internal/cicd/hostagent"
)

type HostAgentProvider struct{ repo *hostagent.Repository }

func NewHostAgentProvider(repo *hostagent.Repository) *HostAgentProvider {
	return &HostAgentProvider{repo: repo}
}

func (p *HostAgentProvider) Submit(ctx context.Context, req HostProviderRequest) error {
	if p == nil || p.repo == nil || req.IdempotencyKey == "" {
		return fmt.Errorf("host Agent provider is not configured")
	}
	commands := make([]hostagent.DeploymentCommand, 0, len(req.Target.AgentIDs))
	for _, agentID := range req.Target.AgentIDs {
		commands = append(commands, hostagent.DeploymentCommand{
			WorkflowRunID: req.RunID, TaskKey: req.NodeKey, ProjectID: req.ProjectID,
			ServiceID: req.ServiceID, ServiceRevisionID: req.ServiceRevisionID, AgentID: agentID,
			ArtifactURL: req.Target.ArtifactURL, ArtifactSHA256: req.Target.ArtifactSHA256,
			DeployScript: req.Target.DeployScript, HealthScript: req.Target.HealthScript,
			RollbackScript: req.Target.RollbackScript, TimeoutSeconds: req.Target.TimeoutSeconds,
		})
	}
	return p.repo.Enqueue(ctx, commands, hostagent.RolloutPolicy{
		BatchSize: req.Target.BatchSize, IntervalSeconds: req.Target.BatchIntervalSeconds,
	})
}

func (p *HostAgentProvider) Status(ctx context.Context, runID int64, nodeKey string) (HostProviderState, error) {
	state, err := p.repo.DeploymentState(ctx, runID, nodeKey)
	if err != nil {
		return HostProviderState{}, err
	}
	if state.Total == 0 {
		return HostProviderState{}, fmt.Errorf("host deployment commands were not found")
	}
	if state.RolloutStatus == RunFailed || state.Failed > 0 || state.RolledBack > 0 {
		message := state.ErrorMessage
		if message == "" {
			message = fmt.Sprintf("host deployment failed on %d target(s), rolled back on %d target(s)", state.Failed, state.RolledBack)
		}
		return HostProviderState{Status: RunFailed, ErrorMessage: message}, nil
	}
	if state.RolloutStatus == RunSucceeded && state.Succeeded == state.Total {
		return HostProviderState{Status: RunSucceeded}, nil
	}
	return HostProviderState{Status: RunRunning}, nil
}
