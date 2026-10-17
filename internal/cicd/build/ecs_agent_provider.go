package build

import (
	"context"
	"fmt"
	"strings"
)

// ECSAgentDispatcher is deliberately separate from host deployment commands.
// A build agent receives source/build inputs and returns an immutable artifact;
// it must never be able to mutate a deployment target directly.
type ECSAgentDispatcher interface {
	SubmitBuild(context.Context, ProviderBuildRequest) (ProviderExecution, error)
	CancelBuild(context.Context, ProviderExecution) error
	StatusBuild(context.Context, ProviderExecution) (ProviderState, error)
	// Required for parity with Kubernetes: persistent identity lookup after a
	// worker/agent restart, and explicit acknowledgement before local cleanup.
	RecoverBuild(context.Context, int64, string) (ProviderExecution, bool, error)
	FinalizeBuild(context.Context, ProviderExecution) error
}

const ECSAgentProviderName = "ecs_agent"

type ECSAgentProvider struct {
	dispatcher ECSAgentDispatcher
}

func NewECSAgentProvider(dispatcher ECSAgentDispatcher) *ECSAgentProvider {
	return &ECSAgentProvider{dispatcher: dispatcher}
}

func (p *ECSAgentProvider) Name() string { return ECSAgentProviderName }

func (p *ECSAgentProvider) Recover(ctx context.Context, id int64, key string) (ProviderExecution, bool, error) {
	if p == nil || p.dispatcher == nil {
		return ProviderExecution{}, false, fmt.Errorf("ecs agent build provider is not configured")
	}
	return p.dispatcher.RecoverBuild(ctx, id, key)
}

func (p *ECSAgentProvider) Finalize(ctx context.Context, execution ProviderExecution) error {
	if p == nil || p.dispatcher == nil {
		return fmt.Errorf("ecs agent build provider is not configured")
	}
	return p.dispatcher.FinalizeBuild(ctx, execution)
}

func (p *ECSAgentProvider) Submit(ctx context.Context, req ProviderBuildRequest) (ProviderExecution, error) {
	if p == nil || p.dispatcher == nil {
		return ProviderExecution{}, fmt.Errorf("ecs agent build provider is not configured")
	}
	if req.BuildRunID <= 0 || strings.TrimSpace(req.IdempotencyKey) == "" {
		return ProviderExecution{}, fmt.Errorf("build run id and idempotency key are required")
	}
	return p.dispatcher.SubmitBuild(ctx, req)
}

func (p *ECSAgentProvider) Cancel(ctx context.Context, execution ProviderExecution) error {
	if p == nil || p.dispatcher == nil {
		return fmt.Errorf("ecs agent build provider is not configured")
	}
	return p.dispatcher.CancelBuild(ctx, execution)
}

func (p *ECSAgentProvider) Status(ctx context.Context, execution ProviderExecution) (ProviderState, error) {
	if p == nil || p.dispatcher == nil {
		return ProviderState{}, fmt.Errorf("ecs agent build provider is not configured")
	}
	return p.dispatcher.StatusBuild(ctx, execution)
}
