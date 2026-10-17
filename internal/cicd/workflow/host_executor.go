package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type HostDeployConfig struct {
	FrozenHost        *HostSource `json:"frozen_host,omitempty"`
	FrozenAgentIDs    []int64     `json:"frozen_agent_ids,omitempty"`
	ServiceID         int64       `json:"service_id"`
	ServiceRevisionID int64       `json:"service_revision_id"`
}

type HostTarget struct {
	AgentIDs             []int64
	ArtifactURL          string
	ArtifactSHA256       string
	DeployScript         string
	HealthScript         string
	RollbackScript       string
	TimeoutSeconds       int
	BatchSize            int
	BatchIntervalSeconds int
}

type HostRenderResolver interface {
	RenderHost(context.Context, WorkflowRun, HostDeployConfig) (HostTarget, error)
}

type HostProviderRequest struct {
	RunID             int64
	NodeKey           string
	ProjectID         int64
	ServiceID         int64
	ServiceRevisionID int64
	Target            HostTarget
	IdempotencyKey    string
}

type HostProviderState struct {
	Status       string
	ErrorMessage string
}

type HostDeployProvider interface {
	Submit(context.Context, HostProviderRequest) error
	Status(context.Context, int64, string) (HostProviderState, error)
}

type HostDeployExecutor struct {
	resolver HostRenderResolver
	provider HostDeployProvider
}

func NewHostDeployExecutor(resolver HostRenderResolver, provider HostDeployProvider) *HostDeployExecutor {
	return &HostDeployExecutor{resolver: resolver, provider: provider}
}

func (e *HostDeployExecutor) NodeType() string { return NodeHostDeploy }

func (e *HostDeployExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.resolver == nil || e.provider == nil {
		return ExecutionResult{}, fmt.Errorf("host deploy executor is not configured")
	}
	config, err := hostDeployConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	target, err := e.resolver.RenderHost(ctx, request.Run, config)
	if err != nil {
		return ExecutionResult{}, err
	}
	if len(target.AgentIDs) == 0 || target.ArtifactURL == "" || target.ArtifactSHA256 == "" || strings.TrimSpace(target.DeployScript) == "" {
		return ExecutionResult{}, fmt.Errorf("host deployment target is incomplete")
	}
	if err := e.provider.Submit(ctx, HostProviderRequest{
		RunID: request.Run.ID, NodeKey: request.Task.NodeKey, ProjectID: request.Run.ProjectID,
		ServiceID: config.ServiceID, ServiceRevisionID: config.ServiceRevisionID, Target: target,
		IdempotencyKey: fmt.Sprintf("workflow:%d:%s", request.Run.ID, request.Task.NodeKey),
	}); err != nil {
		return ExecutionResult{}, err
	}
	return ExecutionResult{
		Status: RunRunning, ProviderRunID: fmt.Sprintf("host:%d:%s", request.Run.ID, request.Task.NodeKey),
		LogRef: fmt.Sprintf("/api/v1/workflow-runs/%d", request.Run.ID),
	}, nil
}

func (e *HostDeployExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	prefix := fmt.Sprintf("host:%d:", request.Run.ID)
	nodeKey, ok := strings.CutPrefix(request.Task.ProviderRunID, prefix)
	if !ok || nodeKey != request.Task.NodeKey {
		return ExecutionResult{}, fmt.Errorf("invalid host provider run id %q", request.Task.ProviderRunID)
	}
	state, err := e.provider.Status(ctx, request.Run.ID, request.Task.NodeKey)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case RunSucceeded, RunFailed, RunCancelled:
		return ExecutionResult{Status: state.Status, ErrorMessage: state.ErrorMessage}, nil
	case RunPending, RunRunning:
		return ExecutionResult{Status: RunRunning}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported host provider status %q", state.Status)
	}
}

func hostDeployConfig(snapshot json.RawMessage) (HostDeployConfig, error) {
	var node Node
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &node) != nil {
		return HostDeployConfig{}, fmt.Errorf("invalid host deploy node snapshot")
	}
	var config HostDeployConfig
	if len(node.Config) == 0 || json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 || config.ServiceRevisionID <= 0 {
		return HostDeployConfig{}, fmt.Errorf("host deploy requires service_id and service_revision_id")
	}
	return config, nil
}
