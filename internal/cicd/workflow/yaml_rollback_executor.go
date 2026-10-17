package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// NodeYAMLRollback is reserved for system-created rollback runs. It is not
// accepted by ValidateDefinition and therefore cannot be added by a client.
const NodeYAMLRollback = "yaml_rollback"

type YAMLRollbackConfig struct {
	SnapshotID int64 `json:"snapshot_id"`
}

type YAMLRollbackExecutor struct {
	snapshots DeploymentSnapshotReader
	provider  YAMLDeployProvider
	preparer  YAMLDeploymentPreparer
}

func (e *YAMLRollbackExecutor) WithDeploymentPreparer(p YAMLDeploymentPreparer) *YAMLRollbackExecutor {
	e.preparer = p
	return e
}

func NewYAMLRollbackExecutor(snapshots DeploymentSnapshotReader, provider YAMLDeployProvider) *YAMLRollbackExecutor {
	return &YAMLRollbackExecutor{snapshots: snapshots, provider: provider}
}

func (e *YAMLRollbackExecutor) NodeType() string { return NodeYAMLRollback }

func (e *YAMLRollbackExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.snapshots == nil || e.provider == nil {
		return ExecutionResult{}, fmt.Errorf("yaml rollback executor is not configured")
	}
	config, err := yamlRollbackConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	snapshot, err := e.snapshots.FindApplied(ctx, config.SnapshotID)
	if err != nil {
		return ExecutionResult{}, err
	}
	if snapshot == nil {
		return ExecutionResult{}, fmt.Errorf("applied deployment snapshot %d was not found", config.SnapshotID)
	}
	if snapshot.Provider != NodeYAMLDeploy || snapshot.ProjectID != request.Run.ProjectID || snapshot.EnvironmentID != request.Run.EnvironmentID {
		return ExecutionResult{}, fmt.Errorf("deployment snapshot does not belong to the rollback run target")
	}
	if !snapshot.VerifyManifestDigest() {
		return ExecutionResult{}, fmt.Errorf("deployment snapshot manifest digest mismatch")
	}
	if _, err := ValidateYAMLManifest(snapshot.RenderedManifest, snapshot.Namespace); err != nil {
		return ExecutionResult{}, fmt.Errorf("validate rollback manifest: %w", err)
	}
	if validator, ok := e.snapshots.(rollbackExecutionValidator); ok {
		if err := validator.ValidateRollbackExecution(ctx, request.Run, request.Task, snapshot); err != nil {
			return ExecutionResult{}, err
		}
	}
	if request.ReportProgress != nil {
		if err := request.ReportProgress(fmt.Sprintf("[prepare] 正在回滚服务 #%d 到运行记录 %d 的成功版本；目标 %s", snapshot.ServiceID, snapshot.WorkflowRunID, snapshot.Namespace)); err != nil {
			return ExecutionResult{}, err
		}
	}
	target := YAMLTarget{ClusterID: snapshot.ClusterID, Namespace: snapshot.Namespace, Manifest: append([]byte(nil), snapshot.RenderedManifest...)}
	if e.preparer != nil {
		target, err = e.preparer.PrepareYAMLDeployment(ctx, request.Run, YAMLDeployConfig{ServiceID: snapshot.ServiceID, ServiceRevisionID: snapshot.ServiceRevisionID}, target)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("回滚依赖检查失败: %w", err)
		}
	}
	if previewer, ok := e.provider.(interface {
		Preview(context.Context, YAMLProviderRequest) ([]YAMLResourceChange, error)
	}); ok {
		if _, err := previewer.Preview(ctx, YAMLProviderRequest{ClusterID: target.ClusterID, Namespace: target.Namespace, Manifest: target.Manifest}); err != nil {
			return ExecutionResult{}, fmt.Errorf("回滚 Kubernetes 预检失败: %w", err)
		}
	}
	if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
		output := rollbackOutputSnapshot(snapshot, request.Run, request.Task)
		output.RenderedManifest = target.Manifest
		hash := sha256.Sum256(output.RenderedManifest)
		output.ManifestDigest = hex.EncodeToString(hash[:])
		if err := recorder.Prepare(ctx, output); err != nil {
			return ExecutionResult{}, err
		}
	}
	execution, err := e.provider.Apply(ctx, YAMLProviderRequest{
		RunID: request.Run.ID, NodeKey: request.Task.NodeKey, ClusterID: snapshot.ClusterID,
		Namespace: snapshot.Namespace, Manifest: target.Manifest,
		IdempotencyKey: fmt.Sprintf("workflow:%d:%s:rollback:%d", request.Run.ID, request.Task.NodeKey, snapshot.ID),
		ReportProgress: request.ReportProgress,
	})
	if err != nil {
		return ExecutionResult{LogText: execution.LogText}, err
	}
	if execution.ExternalID == "" {
		return ExecutionResult{}, fmt.Errorf("yaml rollback provider returned empty external id")
	}
	if execution.Completed {
		if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
			if err := recorder.MarkApplied(ctx, request.Run.ID, request.Task.NodeKey, NodeYAMLDeploy); err != nil {
				return ExecutionResult{}, err
			}
		}
		return ExecutionResult{Status: RunSucceeded, ProviderRunID: "yaml-rollback:" + execution.ExternalID, LogRef: execution.LogRef, LogText: execution.LogText}, nil
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: "yaml-rollback:" + execution.ExternalID, LogRef: execution.LogRef, LogText: execution.LogText}, nil
}

func (e *YAMLRollbackExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	externalID, ok := strings.CutPrefix(request.Task.ProviderRunID, "yaml-rollback:")
	if !ok || externalID == "" {
		return ExecutionResult{}, fmt.Errorf("invalid yaml rollback provider run id %q", request.Task.ProviderRunID)
	}
	state, err := e.provider.Status(ctx, externalID)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case RunSucceeded:
		if recorder, ok := e.snapshots.(rollbackSnapshotRecorder); ok {
			if err := recorder.MarkApplied(ctx, request.Run.ID, request.Task.NodeKey, NodeYAMLDeploy); err != nil {
				return ExecutionResult{}, err
			}
		}
		return ExecutionResult{Status: RunSucceeded, LogText: state.LogText}, nil
	case RunFailed:
		return ExecutionResult{Status: RunFailed, ErrorMessage: state.ErrorMessage, LogText: state.LogText}, nil
	case RunCancelled:
		return ExecutionResult{Status: RunCancelled}, nil
	case RunPending, RunRunning:
		return ExecutionResult{Status: RunRunning, LogText: state.LogText}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported yaml rollback provider status %q", state.Status)
	}
}

func yamlRollbackConfig(raw json.RawMessage) (YAMLRollbackConfig, error) {
	var node Node
	if len(raw) == 0 || json.Unmarshal(raw, &node) != nil {
		return YAMLRollbackConfig{}, fmt.Errorf("invalid yaml rollback node snapshot")
	}
	var config YAMLRollbackConfig
	if len(node.Config) == 0 || json.Unmarshal(node.Config, &config) != nil || config.SnapshotID <= 0 {
		return YAMLRollbackConfig{}, fmt.Errorf("yaml rollback node requires snapshot_id")
	}
	return config, nil
}
