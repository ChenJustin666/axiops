package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// NodeHelmRollback is reserved for audited, system-created rollback runs.
const NodeHelmRollback = "helm_rollback"

type HelmRollbackConfig struct {
	SnapshotID int64 `json:"snapshot_id"`
}

type HelmRollbackExecutor struct {
	snapshots DeploymentSnapshotReader
	provider  HelmProvider
}

func NewHelmRollbackExecutor(snapshots DeploymentSnapshotReader, provider HelmProvider) *HelmRollbackExecutor {
	return &HelmRollbackExecutor{snapshots: snapshots, provider: provider}
}

func (e *HelmRollbackExecutor) NodeType() string { return NodeHelmRollback }

func (e *HelmRollbackExecutor) Execute(ctx context.Context, request ExecutionRequest) (result ExecutionResult, err error) {
	progress := newHelmTaskProgress(request.ReportProgress)
	defer func() { result.LogText = progress.text(result.LogText) }()
	if e == nil || e.snapshots == nil || e.provider == nil {
		return ExecutionResult{}, fmt.Errorf("Helm rollback executor is not configured")
	}
	config, err := helmRollbackConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	if err := progress.report("[prepare] 正在校验 Helm 历史部署快照和回滚目标"); err != nil {
		return ExecutionResult{}, err
	}
	snapshot, err := e.snapshots.FindApplied(ctx, config.SnapshotID)
	if err != nil || snapshot == nil {
		if err == nil {
			err = fmt.Errorf("applied Helm snapshot was not found")
		}
		return ExecutionResult{}, err
	}
	if snapshot.Provider != NodeHelmDeploy || snapshot.ProjectID != request.Run.ProjectID || snapshot.EnvironmentID != request.Run.EnvironmentID || snapshot.HelmReleaseName == "" || snapshot.HelmRevision <= 0 || !snapshot.VerifyManifestDigest() {
		return ExecutionResult{}, fmt.Errorf("Helm rollback snapshot does not match the workflow target")
	}
	if err := rejectHelmJobRollbackInventory(snapshot.RenderedManifest); err != nil {
		return ExecutionResult{}, err
	}
	if validator, ok := e.snapshots.(rollbackExecutionValidator); ok {
		if err := validator.ValidateRollbackExecution(ctx, request.Run, request.Task, snapshot); err != nil {
			return ExecutionResult{}, err
		}
	}
	if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
		if err := recorder.Prepare(ctx, rollbackOutputSnapshot(snapshot, request.Run, request.Task)); err != nil {
			return ExecutionResult{}, err
		}
	}
	if err := progress.report(fmt.Sprintf("[target] 集群 #%d · 命名空间 %s · Helm Release %s\n[waiting] 正在回滚到 revision %d；等待资源就绪", snapshot.ClusterID, snapshot.Namespace, snapshot.HelmReleaseName, snapshot.HelmRevision)); err != nil {
		return ExecutionResult{}, err
	}
	timeout := 600
	var frozen struct {
		CurrentRevision int `json:"expected_helm_revision"`
	}
	_ = json.Unmarshal(request.Run.ContextSnapshot, &frozen)
	var node Node
	if json.Unmarshal(request.Task.NodeSnapshot, &node) == nil && node.TimeoutSeconds > 0 {
		timeout = node.TimeoutSeconds
	}
	execution, err := e.provider.Rollback(ctx, HelmRollbackRequest{
		ExpectedCurrentRevision: frozen.CurrentRevision, ExpectedInventoryDigest: snapshot.ManifestDigest,
		RunID: request.Run.ID, NodeKey: request.Task.NodeKey, ClusterID: snapshot.ClusterID,
		Namespace: snapshot.Namespace, ReleaseName: snapshot.HelmReleaseName, Revision: snapshot.HelmRevision,
		Atomic: true, Wait: true, TimeoutSeconds: timeout,
		IdempotencyKey: fmt.Sprintf("workflow:%d:%s", request.Run.ID, request.Task.NodeKey),
	})
	if err != nil {
		return ExecutionResult{}, err
	}
	if execution.ExternalID == "" {
		return ExecutionResult{}, fmt.Errorf("Helm rollback provider returned empty external id")
	}
	if execution.Completed {
		if execution.Revision <= 0 {
			return ExecutionResult{}, fmt.Errorf("Helm rollback provider returned an invalid release revision")
		}
		if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
			if err := recorder.MarkHelmApplied(ctx, request.Run.ID, request.Task.NodeKey, snapshot.HelmReleaseName, execution.Revision); err != nil {
				return ExecutionResult{}, err
			}
		}
		return ExecutionResult{Status: RunSucceeded, ProviderRunID: "helm:" + execution.ExternalID, LogRef: execution.LogRef, LogText: fmt.Sprintf("[success] Helm 回滚已就绪 · 新 revision %d · 恢复自 revision %d", execution.Revision, snapshot.HelmRevision)}, nil
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: "helm:" + execution.ExternalID, LogRef: execution.LogRef, LogText: "[recover] 已找到原 Helm 回滚记录，继续核对结果，不重复回滚"}, nil
}

func (e *HelmRollbackExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	externalID, ok := strings.CutPrefix(request.Task.ProviderRunID, "helm:")
	if !ok || externalID == "" {
		return ExecutionResult{}, fmt.Errorf("invalid Helm rollback provider run id")
	}
	state, err := e.provider.Status(ctx, externalID)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case RunSucceeded:
		if state.Revision <= 0 {
			return ExecutionResult{}, fmt.Errorf("Helm rollback provider returned an invalid release revision")
		}
		if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
			cfg, err := helmRollbackConfig(request.Task.NodeSnapshot)
			if err != nil {
				return ExecutionResult{}, err
			}
			snapshot, err := e.snapshots.FindApplied(ctx, cfg.SnapshotID)
			if err != nil {
				return ExecutionResult{}, err
			}
			if snapshot == nil {
				return ExecutionResult{}, fmt.Errorf("Helm rollback source snapshot missing")
			}
			if err := recorder.MarkHelmApplied(ctx, request.Run.ID, request.Task.NodeKey, snapshot.HelmReleaseName, state.Revision); err != nil {
				return ExecutionResult{}, err
			}
		}
		return ExecutionResult{Status: RunSucceeded, LogText: fmt.Sprintf("[success] Helm 回滚已就绪 · 新 revision %d", state.Revision)}, nil
	case RunFailed:
		return ExecutionResult{Status: RunFailed, ErrorMessage: state.ErrorMessage, LogText: "[failed] Helm 回滚失败：" + safeDiagnostic(state.ErrorMessage)}, nil
	case RunPending, RunRunning:
		return ExecutionResult{Status: RunRunning}, nil
	case RunCancelled:
		return ExecutionResult{Status: RunCancelled, LogText: "[cancel] Helm 回滚已取消，请核对环境中的实际资源状态"}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported Helm rollback status %q", state.Status)
	}
}

func helmRollbackConfig(snapshot json.RawMessage) (HelmRollbackConfig, error) {
	var node Node
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &node) != nil {
		return HelmRollbackConfig{}, fmt.Errorf("invalid Helm rollback node snapshot")
	}
	var config HelmRollbackConfig
	if json.Unmarshal(node.Config, &config) != nil || config.SnapshotID <= 0 {
		return HelmRollbackConfig{}, fmt.Errorf("Helm rollback requires snapshot_id")
	}
	return config, nil
}
