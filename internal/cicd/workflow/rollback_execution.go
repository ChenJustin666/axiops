package workflow

import (
	"context"
	"encoding/json"
	"fmt"
)

type rollbackExecutionValidator interface {
	ValidateRollbackExecution(context.Context, WorkflowRun, TaskRun, *DeploymentSnapshot) error
}

type rollbackSnapshotRecorder interface {
	DeploymentSnapshotStore
	MarkHelmApplied(context.Context, int64, string, string, int) error
}

func (s *SQLDeploymentSnapshotStore) ValidateRollbackExecution(ctx context.Context, run WorkflowRun, task TaskRun, target *DeploymentSnapshot) error {
	var frozen struct {
		Target       *EnvironmentTarget `json:"target"`
		Request      *RollbackRequest   `json:"rollback_request"`
		CurrentID    int64              `json:"expected_current_snapshot_id"`
		SnapshotID   int64              `json:"deployment_snapshot_id"`
		TargetDigest string             `json:"target_manifest_digest"`
	}
	if json.Unmarshal(run.ContextSnapshot, &frozen) != nil || frozen.Target == nil || frozen.Request == nil || frozen.CurrentID <= 0 || frozen.SnapshotID != target.ID || frozen.Request.ServiceID != target.ServiceID {
		return fmt.Errorf("回滚执行缺少可信的目标和版本快照，已停止")
	}
	if frozen.TargetDigest != target.ManifestDigest {
		return fmt.Errorf("回滚目标快照在提交后变化，已停止")
	}
	env, err := NewRepository(s.db).FindEnvironmentTarget(ctx, run.EnvironmentID)
	if err != nil {
		return err
	}
	if env == nil || *env != *frozen.Target || env.ClusterID != target.ClusterID || env.Namespace != target.Namespace || env.ProjectID != run.ProjectID || env.Status != "active" || env.BindingStatus != "validated" {
		return fmt.Errorf("回滚目标环境已改变，已停止")
	}
	if err := checkRollbackEnvironmentIsolation(ctx, s.db, env); err != nil {
		return err
	}
	if err := checkRollbackResourceOwnership(ctx, s.db, target); err != nil {
		return err
	}
	var latestID, latestRun int64
	var latestNode string
	if err := s.db.QueryRowContext(ctx, `SELECT id,workflow_run_id,node_key FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? ORDER BY id DESC LIMIT 1`, run.ProjectID, run.EnvironmentID, target.ServiceID).Scan(&latestID, &latestRun, &latestNode); err != nil {
		return err
	}
	// Recovery may already have written this operation's prepared/applied snapshot.
	if latestID != frozen.CurrentID && (latestRun != run.ID || latestNode != task.NodeKey) {
		return fmt.Errorf("回滚执行前发现新的部署版本，已停止；请重新预览")
	}
	return checkRollbackConflicts(ctx, s.db, run.ProjectID, run.EnvironmentID, []int64{target.ServiceID}, true, run.ID)
}

func rollbackOutputSnapshot(source *DeploymentSnapshot, run WorkflowRun, task TaskRun) *DeploymentSnapshot {
	output := *source
	output.ID = 0
	output.WorkflowRunID = run.ID
	output.WorkflowTaskRunID = task.ID
	output.NodeKey = task.NodeKey
	output.Status = "prepared"
	output.RenderedManifest = append([]byte(nil), source.RenderedManifest...)
	return &output
}
