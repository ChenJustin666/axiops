package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/notification"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

type Repository struct {
	db              *sql.DB
	lastActiveRunID int64
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindProjectDeliveryProfile(ctx context.Context, projectID int64) (*DeliveryProfile, error) {
	var profile DeliveryProfile
	err := r.db.QueryRowContext(ctx, `SELECT delivery_type, delivery_status FROM project WHERE id=?`, projectID).Scan(&profile.DeliveryType, &profile.DeliveryStatus)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

func (r *Repository) List(ctx context.Context, scope authz.ScopeSet, projectID int64) ([]Workflow, error) {
	predicate, args := scope.WorkflowSQLPredicate("workflow.project_id", "workflow.id", "workflow.target_environment_id",
		"(SELECT ae.code FROM project_environment ae WHERE ae.id=workflow.target_environment_id AND ae.project_id=workflow.project_id)",
		"(SELECT ae.cluster_id FROM project_environment ae WHERE ae.id=workflow.target_environment_id AND ae.project_id=workflow.project_id)")
	args = append(args, projectID)
	rows, err := r.db.QueryContext(ctx, `SELECT workflow.id,workflow.project_id,workflow.name,workflow.code,workflow.description,
		workflow.delivery_type,workflow.definition_version,workflow.status,workflow.created_by,workflow.updated_by,
		workflow.created_at,workflow.updated_at,workflow.target_environment_id,COUNT(workflow_node.id),COALESCE(MAX(workflow_node.node_type IN ('yaml_deploy','helm_deploy','host_deploy')),0)
		FROM workflow LEFT JOIN workflow_node ON workflow_node.workflow_id=workflow.id
		WHERE `+predicate+` AND workflow.project_id=?
		GROUP BY workflow.id,workflow.project_id,workflow.name,workflow.code,workflow.description,workflow.delivery_type,
		workflow.definition_version,workflow.status,workflow.created_by,workflow.updated_by,workflow.created_at,workflow.updated_at,workflow.target_environment_id
		ORDER BY workflow.updated_at DESC,workflow.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Workflow, 0)
	for rows.Next() {
		var item Workflow
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.Code, &item.Description,
			&item.DeliveryType, &item.DefinitionVersion, &item.Status, &item.CreatedBy, &item.UpdatedBy,
			&item.CreatedAt, &item.UpdatedAt, &item.TargetEnvironmentID, &item.NodeCount, &item.RequiresEnvironment); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Workflow, error) {
	var workflow Workflow
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, name, code, description, delivery_type, definition_version, status, created_by, updated_by, created_at, updated_at, target_environment_id FROM workflow WHERE id=?`, id).Scan(
		&workflow.ID, &workflow.ProjectID, &workflow.Name, &workflow.Code, &workflow.Description, &workflow.DeliveryType,
		&workflow.DefinitionVersion, &workflow.Status, &workflow.CreatedBy, &workflow.UpdatedBy, &workflow.CreatedAt, &workflow.UpdatedAt, &workflow.TargetEnvironmentID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT node_key, name, node_type, config_json, depends_on, timeout_seconds, retry_limit FROM workflow_node WHERE workflow_id=? ORDER BY ordinal, id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var node Node
		var config, dependencies []byte
		if err := rows.Scan(&node.Key, &node.Name, &node.Type, &config, &dependencies, &node.TimeoutSeconds, &node.RetryLimit); err != nil {
			return nil, err
		}
		node.Config = json.RawMessage(config)
		if err := json.Unmarshal(dependencies, &node.DependsOn); err != nil {
			return nil, err
		}
		workflow.Nodes = append(workflow.Nodes, node)
	}
	return &workflow, rows.Err()
}

func (r *Repository) Create(ctx context.Context, workflow *Workflow) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := lockNodeServices(ctx, tx, workflow.ProjectID, workflow.Nodes); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO workflow (project_id, name, code, description, delivery_type, definition_version, status, created_by, updated_by, target_environment_id) VALUES (?, ?, ?, ?, ?, 1, 'active', ?, ?, ?)`, workflow.ProjectID, workflow.Name, workflow.Code, workflow.Description, workflow.DeliveryType, workflow.CreatedBy, workflow.UpdatedBy, workflow.TargetEnvironmentID)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertNodes(ctx, tx, id, workflow.Nodes); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) Update(ctx context.Context, workflow *Workflow, expectedVersion int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if err := lockNodeServices(ctx, tx, workflow.ProjectID, workflow.Nodes); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow SET name=?, description=?, target_environment_id=?, definition_version=definition_version+1, updated_by=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND definition_version=? AND status='active'`, workflow.Name, workflow.Description, workflow.TargetEnvironmentID, workflow.UpdatedBy, workflow.ID, expectedVersion)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workflow_node WHERE workflow_id=?`, workflow.ID); err != nil {
		return false, err
	}
	if err := insertNodes(ctx, tx, workflow.ID, workflow.Nodes); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func insertNodes(ctx context.Context, tx *sql.Tx, workflowID int64, nodes []Node) error {
	for ordinal, node := range nodes {
		config := node.Config
		if len(config) == 0 {
			config = json.RawMessage(`{}`)
		}
		dependencies, err := json.Marshal(node.DependsOn)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_node (workflow_id, node_key, name, node_type, config_json, depends_on, timeout_seconds, retry_limit, ordinal) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, workflowID, node.Key, node.Name, node.Type, config, dependencies, node.TimeoutSeconds, node.RetryLimit, ordinal); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) FindEnvironmentTarget(ctx context.Context, id int64) (*EnvironmentTarget, error) {
	var target EnvironmentTarget
	err := r.db.QueryRowContext(ctx, `SELECT id, project_id, environment_type, binding_status, is_production, deploy_approval, status, code, cluster_id, namespace, host_group_ref FROM project_environment WHERE id=?`, id).
		Scan(&target.ID, &target.ProjectID, &target.EnvironmentType, &target.BindingStatus, &target.IsProduction, &target.DeployApproval, &target.Status, &target.Code, &target.ClusterID, &target.Namespace, &target.HostGroupRef)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &target, err
}

func (r *Repository) FindReleaseTarget(ctx context.Context, serviceID, revisionID int64) (*ReleaseTarget, error) {
	var target ReleaseTarget
	err := r.db.QueryRowContext(ctx, `SELECT svc.project_id,svc.delivery_type FROM app_service svc JOIN service_revision rev ON rev.service_id=svc.id WHERE svc.id=? AND rev.id=? AND svc.status='active'`, serviceID, revisionID).
		Scan(&target.ProjectID, &target.DeliveryType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &target, err
}

func (r *Repository) FindAppliedDeploymentSnapshot(ctx context.Context, id int64) (*DeploymentSnapshot, error) {
	return NewSQLDeploymentSnapshotStore(r.db).FindApplied(ctx, id)
}

func (r *Repository) CreateRun(ctx context.Context, run *WorkflowRun, tasks []TaskRun) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := projectstate.LockWritable(ctx, tx, run.ProjectID); err != nil {
		return 0, err
	}
	if err := lockTaskServices(ctx, tx, run.ProjectID, tasks); err != nil {
		return 0, err
	}
	var production struct {
		Submission *ProductionSubmission `json:"production_submission"`
	}
	if err := json.Unmarshal(run.ContextSnapshot, &production); err != nil {
		return 0, err
	}
	if production.Submission != nil {
		if id, err := r.findProductionRequest(ctx, tx, run.TriggerUserID, run.WorkflowID, *production.Submission); err != nil || id > 0 {
			return id, err
		}
	}
	if err := checkRollbackConflicts(ctx, tx, run.ProjectID, run.EnvironmentID, rollbackTaskServices(tasks), false, 0); err != nil {
		return 0, err
	}
	var version, boundEnvironment int64
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT definition_version,target_environment_id,status FROM workflow WHERE id=? AND project_id=? FOR UPDATE`, run.WorkflowID, run.ProjectID).Scan(&version, &boundEnvironment, &status); err != nil {
		return 0, err
	}
	if version != run.DefinitionVersion || status != "active" || boundEnvironment != run.EnvironmentID {
		return 0, fmt.Errorf("workflow configuration changed before submission")
	}
	var frozen struct {
		Target                  *EnvironmentTarget `json:"target"`
		ConfigurationDeployment *CurrentDeployment `json:"configuration_deployment"`
		ExpectedConfigDigest    string             `json:"expected_config_digest"`
		ServiceID               int64              `json:"service_id"`
	}
	if err := json.Unmarshal(run.ContextSnapshot, &frozen); err != nil {
		return 0, err
	}
	if frozen.Target != nil {
		var current EnvironmentTarget
		if err := tx.QueryRowContext(ctx, `SELECT id, project_id, environment_type, binding_status, is_production, deploy_approval, status, code, cluster_id, namespace, host_group_ref FROM project_environment WHERE id=? FOR UPDATE`, run.EnvironmentID).Scan(&current.ID, &current.ProjectID, &current.EnvironmentType, &current.BindingStatus, &current.IsProduction, &current.DeployApproval, &current.Status, &current.Code, &current.ClusterID, &current.Namespace, &current.HostGroupRef); err != nil {
			return 0, err
		}
		if current != *frozen.Target {
			return 0, fmt.Errorf("environment changed before submission")
		}
		if current.IsProduction {
			if err := validateProductionSubmission(ctx, tx, run, tasks); err != nil {
				return 0, err
			}
			if err := checkRollbackConflicts(ctx, tx, run.ProjectID, run.EnvironmentID, rollbackTaskServices(tasks), true, 0); err != nil {
				return 0, err
			}
		}
		// Serialize configuration releases with every deployment to this target.
		active, e := tx.QueryContext(ctx, `SELECT id FROM workflow_run WHERE project_id=? AND environment_id=? AND status IN ('pending','running','pending_approval') AND (?=1 OR JSON_UNQUOTE(JSON_EXTRACT(context_snapshot,'$.execution_mode'))='config_only') FOR UPDATE`, run.ProjectID, run.EnvironmentID, frozen.ConfigurationDeployment != nil)
		if e != nil {
			return 0, e
		}
		occupied := active.Next()
		readErr := active.Err()
		active.Close()
		if readErr != nil {
			return 0, readErr
		}
		if occupied {
			return 0, fmt.Errorf("当前环境已有发布任务，请结束后重新预览")
		}
		if frozen.ConfigurationDeployment != nil {
			var snapshotID int64
			var snapshotStatus string
			if e := tx.QueryRowContext(ctx, `SELECT id,status FROM deployment_snapshot WHERE project_id=? AND environment_id=? AND service_id=? AND provider=? ORDER BY id DESC LIMIT 1 FOR UPDATE`, run.ProjectID, run.EnvironmentID, frozen.ServiceID, NodeYAMLDeploy).Scan(&snapshotID, &snapshotStatus); e != nil {
				return 0, e
			}
			if snapshotID != frozen.ConfigurationDeployment.SnapshotID || snapshotStatus != "applied" {
				return 0, fmt.Errorf("上次部署已变化，请重新预览")
			}
			for _, task := range tasks {
				if task.NodeType != NodeYAMLDeploy {
					continue
				}
				cfg, e := yamlDeployConfig(task.NodeSnapshot)
				if e != nil {
					return 0, e
				}
				source, e := NewSQLYAMLSourceStoreTx(tx).LoadYAMLSource(ctx, run.EnvironmentID, cfg.ServiceID, cfg.ServiceRevisionID)
				if e != nil {
					return 0, e
				}
				if DeploymentConfigDigest(source) != frozen.ExpectedConfigDigest {
					return 0, fmt.Errorf("配置已变化，请重新预览")
				}
			}
		}
	}
	number, err := allocateRunNumber(ctx, tx, run.WorkflowID)
	if err != nil {
		return 0, err
	}
	run.RunNumber = number
	var requestKey interface{}
	if production.Submission != nil {
		requestKey = production.Submission.RequestKey
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO workflow_run (workflow_id, project_id, environment_id, definition_version, trigger_type, trigger_user_id, status, context_snapshot,run_number,request_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?,?,?)`, run.WorkflowID, run.ProjectID, run.EnvironmentID, run.DefinitionVersion, run.TriggerType, run.TriggerUserID, run.Status, run.ContextSnapshot, number, requestKey)
	if err != nil {
		return 0, err
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, task := range tasks {
		dependsOn, err := json.Marshal(task.DependsOn)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_task_run (workflow_run_id, node_key, node_type, depends_on, node_snapshot, status) VALUES (?, ?, ?, ?, ?, ?)`, runID, task.NodeKey, task.NodeType, dependsOn, task.NodeSnapshot, task.Status); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

func (r *Repository) CreateRollbackRun(ctx context.Context, run *WorkflowRun, tasks []TaskRun, snapshotID int64, requestKey string) (int64, error) {
	return r.createRollbackRun(ctx, run, tasks, snapshotID, requestKey, true)
}

func (r *Repository) createRollbackRun(ctx context.Context, run *WorkflowRun, tasks []TaskRun, snapshotID int64, requestKey string, enforceGuard bool) (int64, error) {
	if run == nil || snapshotID <= 0 || requestKey == "" {
		return 0, fmt.Errorf("rollback run, snapshot id, and request key are required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := projectstate.LockWritable(ctx, tx, run.ProjectID); err != nil {
		return 0, err
	}
	if err := lockTaskServices(ctx, tx, run.ProjectID, tasks); err != nil {
		return 0, err
	}
	// Serialize normal and rollback runs on the same workflow counter.
	if run.WorkflowID > 0 {
		if err := tx.QueryRowContext(ctx, `SELECT run_sequence FROM workflow WHERE id=? FOR UPDATE`, run.WorkflowID).Scan(&run.RunNumber); err != nil {
			return 0, err
		}
	}
	var priorID, priorSnapshot, priorUser, priorWorkflow int64
	var priorContext []byte
	err = tx.QueryRowContext(ctx, `SELECT id,source_snapshot_id,trigger_user_id,workflow_id,context_snapshot FROM workflow_run WHERE request_key=?`, requestKey).Scan(&priorID, &priorSnapshot, &priorUser, &priorWorkflow, &priorContext)
	if err == nil {
		if priorSnapshot != snapshotID || priorUser != run.TriggerUserID || priorWorkflow != run.WorkflowID {
			return 0, fmt.Errorf("rollback request key is already used by another request")
		}
		var before, now struct {
			Request *RollbackRequest `json:"rollback_request"`
		}
		_ = json.Unmarshal(priorContext, &before)
		_ = json.Unmarshal(run.ContextSnapshot, &now)
		if now.Request != nil && (before.Request == nil || *before.Request != *now.Request) {
			return 0, fmt.Errorf("rollback request key is already used by another request")
		}
		return priorID, tx.Commit()
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	var guarded struct {
		Request *RollbackRequest `json:"rollback_request"`
	}
	if json.Unmarshal(run.ContextSnapshot, &guarded) != nil {
		return 0, fmt.Errorf("invalid rollback context")
	}
	if enforceGuard || guarded.Request != nil {
		if err := validateRollbackSubmission(ctx, tx, run, snapshotID); err != nil {
			return 0, err
		}
	}
	number, err := allocateRunNumber(ctx, tx, run.WorkflowID)
	if err != nil {
		return 0, err
	}
	run.RunNumber = number
	result, err := tx.ExecContext(ctx, `INSERT INTO workflow_run
		(workflow_id, project_id, environment_id, definition_version, trigger_type, operation_type,
		 source_snapshot_id, request_key, trigger_user_id, status, context_snapshot,run_number)
		VALUES (?, ?, ?, ?, ?, 'rollback', ?, ?, ?, ?, ?,?)
		ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`, run.WorkflowID, run.ProjectID, run.EnvironmentID,
		run.DefinitionVersion, run.TriggerType, snapshotID, requestKey, run.TriggerUserID, run.Status, run.ContextSnapshot, runNumberValue(number))
	if err != nil {
		return 0, err
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	var existingSnapshotID, existingUserID int64
	if err := tx.QueryRowContext(ctx, `SELECT source_snapshot_id, trigger_user_id FROM workflow_run WHERE id=?`, runID).
		Scan(&existingSnapshotID, &existingUserID); err != nil {
		return 0, err
	}
	if existingSnapshotID != snapshotID || existingUserID != run.TriggerUserID {
		return 0, fmt.Errorf("rollback request key is already used by another request")
	}
	var existingTaskCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_task_run WHERE workflow_run_id=?`, runID).Scan(&existingTaskCount); err != nil {
		return 0, err
	}
	if existingTaskCount > 0 {
		return runID, tx.Commit()
	}
	for _, task := range tasks {
		dependsOn, err := json.Marshal(task.DependsOn)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_task_run
			(workflow_run_id, node_key, node_type, depends_on, node_snapshot, status)
			VALUES (?, ?, ?, ?, ?, ?)`, runID, task.NodeKey, task.NodeType, dependsOn, task.NodeSnapshot, task.Status); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

func (r *Repository) FindRun(ctx context.Context, id int64) (*WorkflowRun, error) {
	var run WorkflowRun
	err := r.db.QueryRowContext(ctx, `SELECT id, workflow_id, project_id, environment_id, definition_version, trigger_type, operation_type, COALESCE(source_snapshot_id, 0), trigger_user_id, status, context_snapshot, error_message, created_at, started_at, finished_at,COALESCE(run_number,0) FROM workflow_run WHERE id=?`, id).Scan(
		&run.ID, &run.WorkflowID, &run.ProjectID, &run.EnvironmentID, &run.DefinitionVersion, &run.TriggerType, &run.OperationType, &run.SourceSnapshotID, &run.TriggerUserID, &run.Status, &run.ContextSnapshot, &run.ErrorMessage, &run.CreatedAt, &run.StartedAt, &run.FinishedAt, &run.RunNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, workflow_run_id, node_key, node_type, depends_on, node_snapshot, status, attempt, provider_run_id, log_ref, COALESCE(log_text,''), error_message, locked_by, lease_until, created_at, started_at, finished_at,retry_count,next_retry_at,COALESCE(retry_history,JSON_ARRAY()) FROM workflow_task_run WHERE workflow_run_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var task TaskRun
		var dependencies []byte
		var retryHistory []byte
		if err := rows.Scan(&task.ID, &task.WorkflowRunID, &task.NodeKey, &task.NodeType, &dependencies, &task.NodeSnapshot, &task.Status, &task.Attempt, &task.ProviderRunID, &task.LogRef, &task.LogText, &task.ErrorMessage, &task.LockedBy, &task.LeaseUntil, &task.CreatedAt, &task.StartedAt, &task.FinishedAt, &task.RetryCount, &task.NextRetryAt, &retryHistory); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dependencies, &task.DependsOn); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(retryHistory, &task.RetryHistory); err != nil {
			return nil, err
		}
		run.Tasks = append(run.Tasks, task)
	}
	return &run, rows.Err()
}

func (r *Repository) ListRuns(ctx context.Context, scope authz.ScopeSet, projectID, workflowID int64, offset, limit int) ([]WorkflowRun, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	predicate, args := scope.WorkflowSQLPredicate("workflow_run.project_id", "workflow_run.workflow_id", "workflow_run.environment_id",
		"(SELECT ae.code FROM project_environment ae WHERE ae.id=workflow_run.environment_id AND ae.project_id=workflow_run.project_id)",
		"(SELECT ae.cluster_id FROM project_environment ae WHERE ae.id=workflow_run.environment_id AND ae.project_id=workflow_run.project_id)")
	whereArgs := append([]interface{}{}, args...)
	where := predicate + " AND workflow_run.project_id=? AND workflow_run.workflow_id=?"
	whereArgs = append(whereArgs, projectID, workflowID)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_run WHERE `+where, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append([]interface{}{}, whereArgs...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT id, workflow_id, project_id, environment_id, definition_version,
		trigger_type, operation_type, COALESCE(source_snapshot_id, 0), trigger_user_id, status,
		context_snapshot, error_message, created_at, started_at, finished_at,COALESCE(run_number,0)
		FROM workflow_run WHERE `+where+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]WorkflowRun, 0)
	for rows.Next() {
		var item WorkflowRun
		if err := rows.Scan(&item.ID, &item.WorkflowID, &item.ProjectID, &item.EnvironmentID, &item.DefinitionVersion,
			&item.TriggerType, &item.OperationType, &item.SourceSnapshotID, &item.TriggerUserID, &item.Status,
			&item.ContextSnapshot, &item.ErrorMessage, &item.CreatedAt, &item.StartedAt, &item.FinishedAt, &item.RunNumber); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListActiveRuns(ctx context.Context, limit int) ([]WorkflowRun, error) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	after := atomic.LoadInt64(&r.lastActiveRunID)
	read := func(after int64) ([]int64, error) {
		rows, err := r.db.QueryContext(ctx, "SELECT id FROM workflow_run WHERE status IN ('pending','running','cancelling') AND id>? ORDER BY id LIMIT ?", after, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ids := []int64{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, rows.Err()
	}
	ids, err := read(after)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 && after > 0 {
		ids, err = read(0)
		if err != nil {
			return nil, err
		}
	}
	runs := []WorkflowRun{}
	for _, id := range ids {
		run, err := r.FindRun(ctx, id)
		if err != nil {
			return nil, err
		}
		if run != nil {
			runs = append(runs, *run)
		}
		atomic.StoreInt64(&r.lastActiveRunID, id)
	}
	return runs, nil
}

// ClaimTask uses a short lease so a crashed worker can be replaced safely.
func (r *Repository) ClaimTask(ctx context.Context, runID int64, nodeKey, workerID string, lease time.Duration) (bool, error) {
	if runID <= 0 || nodeKey == "" || workerID == "" || lease <= 0 {
		return false, fmt.Errorf("invalid task claim")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Use the same run -> task lock order as renewal/completion. A joined
	// UPDATE can acquire task first while lease renewal already holds run.
	var runStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&runStatus); err != nil {
		return false, err
	}
	if runStatus != RunPending && runStatus != RunRunning {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workflow_run SET status='running', started_at=COALESCE(started_at, CURRENT_TIMESTAMP) WHERE id=? AND status='pending'`, runID); err != nil {
		return false, err
	}
	seconds := int(lease / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_task_run
		SET status='running', attempt=attempt+1, started_at=COALESCE(started_at, CURRENT_TIMESTAMP), locked_by=?, locked_at=CURRENT_TIMESTAMP, lease_until=DATE_ADD(CURRENT_TIMESTAMP, INTERVAL ? SECOND)
		WHERE workflow_run_id=? AND node_key=?
		AND (next_retry_at IS NULL OR next_retry_at<=CURRENT_TIMESTAMP)
		AND (status='pending' OR (status='running' AND provider_run_id='' AND lease_until < CURRENT_TIMESTAMP))`, workerID, seconds, runID, nodeKey)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) CompleteTask(ctx context.Context, runID int64, nodeKey, workerID, status, message string) error {
	// Legacy workflow/task error columns are VARCHAR(1024). Full evidence is
	// saved separately before completion; never let diagnostics block finishing.
	message = truncateDiagnostic(message, 1000)
	if status != RunSucceeded && status != RunFailed {
		return fmt.Errorf("invalid task completion status %q", status)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var runStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&runStatus); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET status=?, error_message=?, locked_by='', lease_until=NULL, finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND node_key=? AND status='running' AND locked_by=? AND lease_until>=CURRENT_TIMESTAMP`, status, message, runID, nodeKey, workerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("task %q was not owned by worker %q", nodeKey, workerID)
	}
	ordered, err := orderedRunTx(ctx, tx, runID)
	if err != nil {
		return err
	}
	if ordered {
		if status == RunFailed {
			if _, err := tx.ExecContext(ctx, `UPDATE workflow_run SET error_message=IF(error_message='', ?, error_message) WHERE id=? AND status='running'`, defaultFailureMessage(message), runID); err != nil {
				return err
			}
		}
		if err := finishOrderedRunTx(ctx, tx, runID); err != nil {
			return err
		}
	} else if status == RunFailed {
		if _, err := tx.ExecContext(ctx, `UPDATE workflow_run SET status='cancelling', error_message=? WHERE id=? AND status IN ('pending', 'running')`, defaultFailureMessage(message), runID); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE workflow_run SET status='succeeded', finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running' AND NOT EXISTS (SELECT 1 FROM workflow_task_run WHERE workflow_run_id=? AND status IN ('pending', 'running'))`, runID, runID); err != nil {
			return err
		}
	}
	if err := enqueueWorkflowTerminal(ctx, tx, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) MarkTaskSubmitted(ctx context.Context, runID int64, nodeKey, workerID string, result ExecutionResult) error {
	updated, err := r.db.ExecContext(ctx, `UPDATE workflow_task_run
		SET provider_run_id=?, log_ref=?, locked_by='', locked_at=NULL, lease_until=NULL
		WHERE workflow_run_id=? AND node_key=? AND status='running' AND locked_by=? AND lease_until>=CURRENT_TIMESTAMP`,
		result.ProviderRunID, result.LogRef, runID, nodeKey, workerID)
	if err != nil {
		return err
	}
	affected, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("task %q submission lease was lost", nodeKey)
	}
	return nil
}

func (r *Repository) CompleteSubmittedTask(ctx context.Context, runID int64, nodeKey, status, message string) error {
	return r.completeObservedTask(ctx, runID, nodeKey, "", status, message)
}

func (r *Repository) CompleteObservedTask(ctx context.Context, runID int64, task TaskRun, status, message string) error {
	return r.completeObservedTask(ctx, runID, task.NodeKey, task.ProviderRunID, status, message)
}

func (r *Repository) completeObservedTask(ctx context.Context, runID int64, nodeKey, providerID, status, message string) error {
	message = truncateDiagnostic(message, 1000)
	if status != RunSucceeded && status != RunFailed && status != RunCancelled {
		return fmt.Errorf("invalid submitted task status %q", status)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var runStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&runStatus); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET status=?, error_message=?, finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND node_key=? AND status='running' AND locked_by='' AND provider_run_id<>'' AND (?='' OR provider_run_id=?)`, status, message, runID, nodeKey, providerID, providerID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("submitted task %q is no longer running", nodeKey)
	}
	ordered, err := orderedRunTx(ctx, tx, runID)
	if err != nil {
		return err
	}
	if ordered && status != RunCancelled {
		if status == RunFailed {
			_, err = tx.ExecContext(ctx, `UPDATE workflow_run SET error_message=IF(error_message='', ?, error_message) WHERE id=? AND status='running'`, defaultFailureMessage(message), runID)
		}
		if err == nil {
			err = finishOrderedRunTx(ctx, tx, runID)
		}
	} else {
		switch status {
		case RunFailed:
			_, err = tx.ExecContext(ctx, `UPDATE workflow_run SET status='cancelling', error_message=? WHERE id=? AND status='running'`, defaultFailureMessage(message), runID)
		case RunCancelled:
			_, err = tx.ExecContext(ctx, `UPDATE workflow_run SET status='cancelling' WHERE id=? AND status='running'`, runID)
		default:
			_, err = tx.ExecContext(ctx, `UPDATE workflow_run SET status='succeeded', finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running' AND NOT EXISTS (SELECT 1 FROM workflow_task_run WHERE workflow_run_id=? AND status IN ('pending', 'running'))`, runID, runID)
		}
	}
	if err != nil {
		return err
	}
	if err := enqueueWorkflowTerminal(ctx, tx, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func orderedRunTx(ctx context.Context, tx *sql.Tx, runID int64) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT node_type, node_snapshot FROM workflow_task_run WHERE workflow_run_id=? AND node_type IN ('yaml_deploy','helm_deploy','host_deploy')`, runID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	var tasks []TaskRun
	for rows.Next() {
		var task TaskRun
		if err := rows.Scan(&task.NodeType, &task.NodeSnapshot); err != nil {
			return false, err
		}
		tasks = append(tasks, task)
	}
	return hasOrderedDeployments(tasks), rows.Err()
}

func finishOrderedRunTx(ctx context.Context, tx *sql.Tx, runID int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE workflow_run SET
		status=IF(EXISTS(SELECT 1 FROM workflow_task_run WHERE workflow_run_id=? AND status='failed'), 'failed', 'succeeded'),
		finished_at=CURRENT_TIMESTAMP
		WHERE id=? AND status='running'
		AND NOT EXISTS(SELECT 1 FROM workflow_task_run WHERE workflow_run_id=? AND status IN ('pending','running','pending_approval'))`, runID, runID, runID)
	return err
}

func (r *Repository) SkipTask(ctx context.Context, runID int64, nodeKey string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=? FOR UPDATE`, runID).Scan(&status); err != nil {
		return err
	}
	if status != RunRunning {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET status='skipped', finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND node_key=? AND status='pending'`, runID, nodeKey); err != nil {
		return err
	}
	if err := finishOrderedRunTx(ctx, tx, runID); err != nil {
		return err
	}
	if err := enqueueWorkflowTerminal(ctx, tx, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func enqueueWorkflowTerminal(ctx context.Context, tx *sql.Tx, runID int64) error {
	var projectID, triggerUserID int64
	var status, message string
	if err := tx.QueryRowContext(ctx, `SELECT project_id,trigger_user_id,status,error_message FROM workflow_run WHERE id=?`, runID).Scan(&projectID, &triggerUserID, &status, &message); err != nil {
		return err
	}
	if status != RunSucceeded && status != RunFailed && status != RunCancelled {
		return nil
	}
	if strings.TrimSpace(message) == "" {
		switch status {
		case RunSucceeded:
			message = "工作流执行成功，所有节点均已完成。"
		case RunCancelled:
			message = "工作流运行已取消。"
		default:
			message = "工作流执行失败，请打开运行详情查看失败节点和日志。"
		}
	}
	payload := notification.Message{EventType: "workflow.terminal", Title: fmt.Sprintf("Workflow run #%d %s", runID, status), Content: message, Status: status, ProjectID: projectID, ResourceID: runID, URL: fmt.Sprintf("/cicd/workflow-runs/%d", runID)}
	return notification.EnqueueTerminal(ctx, tx, projectID, triggerUserID, "workflow.terminal", runID, status, payload)
}

func (r *Repository) RequestCancel(ctx context.Context, id int64) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE workflow_run SET status='cancelling' WHERE id=? AND status IN ('pending', 'pending_approval', 'running')`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 1 {
		return affected == 1, err
	}
	var status string
	err = r.db.QueryRowContext(ctx, `SELECT status FROM workflow_run WHERE id=?`, id).Scan(&status)
	return status == RunCancelling || status == RunCancelled, err
}

func (r *Repository) ApproveRun(ctx context.Context, id int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE workflow_run SET status='pending' WHERE id=? AND status='pending_approval'`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workflow_task_run SET status='succeeded', finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND node_type='approval' AND status='pending_approval'`, id); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
