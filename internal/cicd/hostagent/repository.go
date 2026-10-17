package hostagent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func randomToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw)
	return token, hashToken(token), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) CreateRegistrationToken(ctx context.Context, projectID int64, group string, expires time.Time, maxUses int, createdBy int64) (string, error) {
	token, hash, err := randomToken()
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO host_agent_registration_token(project_id,host_group_ref,token_hash,expires_at,max_uses,created_by) VALUES(?,?,?,?,?,?)`, projectID, group, hash, expires, maxUses, createdBy)
	return token, err
}

func (r *Repository) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var projectID int64
	var group string
	err = tx.QueryRowContext(ctx, `SELECT project_id,host_group_ref FROM host_agent_registration_token WHERE token_hash=? AND expires_at>CURRENT_TIMESTAMP AND used_count<max_uses FOR UPDATE`, hashToken(req.Token)).Scan(&projectID, &group)
	if err != nil {
		return nil, err
	}
	agentToken, agentHash, err := randomToken()
	if err != nil {
		return nil, err
	}
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO host_agent(project_id,host_group_ref,asset_ref,name,token_hash,version,os,arch,status,metadata,last_heartbeat) VALUES(?,?,?,?,?,?,?,?, 'online',?,CURRENT_TIMESTAMP) ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id),host_group_ref=VALUES(host_group_ref),name=VALUES(name),token_hash=VALUES(token_hash),version=VALUES(version),os=VALUES(os),arch=VALUES(arch),status='online',metadata=VALUES(metadata),last_heartbeat=CURRENT_TIMESTAMP`, projectID, group, req.AssetRef, req.Name, agentHash, req.Version, req.OS, req.Arch, metadata)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE host_agent_registration_token SET used_count=used_count+1 WHERE token_hash=?`, hashToken(req.Token)); err != nil {
		return nil, err
	}
	// A host environment becomes deployable only after an Agent proves that the
	// project and host-group binding is reachable from the restricted network.
	if _, err = tx.ExecContext(ctx, `UPDATE project_environment SET binding_status='validated' WHERE project_id=? AND host_group_ref=? AND environment_type='host_group' AND status='active'`, projectID, group); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &RegisterResponse{AgentID: id, AgentToken: agentToken}, nil
}

func (r *Repository) Authenticate(ctx context.Context, token string) (*Agent, error) {
	var agent Agent
	err := r.db.QueryRowContext(ctx, `SELECT id,project_id,host_group_ref,asset_ref,name,version,os,arch,status,last_heartbeat FROM host_agent WHERE token_hash=?`, hashToken(token)).Scan(&agent.ID, &agent.ProjectID, &agent.HostGroupRef, &agent.AssetRef, &agent.Name, &agent.Version, &agent.OS, &agent.Arch, &agent.Status, &agent.LastHeartbeat)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &agent, err
}

func (r *Repository) Heartbeat(ctx context.Context, id int64, version string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE host_agent SET status='online',version=?,last_heartbeat=CURRENT_TIMESTAMP WHERE id=?`, version, id)
	return err
}

func (r *Repository) Poll(ctx context.Context, agentID int64, lease time.Duration) (*Command, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM host_deployment_command WHERE agent_id=? AND ((status='pending' AND (available_at IS NULL OR available_at<=CURRENT_TIMESTAMP)) OR (status IN ('claimed','running') AND lease_until<CURRENT_TIMESTAMP)) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, agentID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE host_deployment_command SET status='claimed',attempts=attempts+1,locked_at=CURRENT_TIMESTAMP,lease_until=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL ? SECOND),started_at=COALESCE(started_at,CURRENT_TIMESTAMP) WHERE id=?`, int(lease/time.Second), id); err != nil {
		return nil, err
	}
	var command Command
	err = tx.QueryRowContext(ctx, `SELECT id,artifact_url,artifact_sha256,deploy_script,health_script,rollback_script,timeout_seconds FROM host_deployment_command WHERE id=?`, id).Scan(&command.ID, &command.ArtifactURL, &command.ArtifactSHA256, &command.DeployScript, &command.HealthScript, &command.RollbackScript, &command.TimeoutSeconds)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &command, nil
}

func (r *Repository) Report(ctx context.Context, agentID, commandID int64, req ReportRequest) error {
	if req.Status != StatusRunning && req.Status != StatusSuccess && req.Status != StatusFailed && req.Status != StatusRolledBack {
		return fmt.Errorf("invalid command status")
	}
	logText, truncated := tail(req.LogText, 1<<20)
	errorMessage, _ := tail(req.ErrorMessage, 1024)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var runID int64
	var taskKey string
	if err = tx.QueryRowContext(ctx, `SELECT workflow_run_id,task_key FROM host_deployment_command WHERE id=? AND agent_id=?`, commandID, agentID).Scan(&runID, &taskKey); err != nil {
		return err
	}
	var rolloutStatus string
	var currentBatch int
	if err = tx.QueryRowContext(ctx, `SELECT status,current_batch FROM host_deployment_rollout WHERE workflow_run_id=? AND task_key=? FOR UPDATE`, runID, taskKey).Scan(&rolloutStatus, &currentBatch); err != nil {
		return err
	}
	var commandStatus string
	var commandBatch int
	if err = tx.QueryRowContext(ctx, `SELECT status,batch_number FROM host_deployment_command WHERE id=? AND agent_id=? FOR UPDATE`, commandID, agentID).Scan(&commandStatus, &commandBatch); err != nil {
		return err
	}
	if commandStatus == req.Status && isTerminalCommandStatus(req.Status) {
		return tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `UPDATE host_deployment_command SET status=?,log_text=?,log_truncated=?,error_message=?,lease_until=CASE WHEN ?='running' THEN DATE_ADD(CURRENT_TIMESTAMP,INTERVAL 120 SECOND) ELSE NULL END,finished_at=CASE WHEN ? IN ('success','failed','rolled_back') THEN CURRENT_TIMESTAMP ELSE finished_at END WHERE id=? AND agent_id=? AND status IN ('claimed','running')`, req.Status, logText, truncated, errorMessage, req.Status, req.Status, commandID, agentID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return errors.New("command is not owned by agent")
	}
	if rolloutStatus == StatusRunning && isTerminalCommandStatus(req.Status) {
		if decideRolloutTransition(req.Status, commandBatch, currentBatch, 0, 0) == rolloutFail {
			reason := errorMessage
			if reason == "" {
				reason = fmt.Sprintf("host Agent %d reported %s", agentID, req.Status)
			}
			if _, err = tx.ExecContext(ctx, `UPDATE host_deployment_rollout SET status='failed',error_message=?,finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND task_key=?`, reason, runID, taskKey); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE host_deployment_command SET status='cancelled',stop_reason=?,finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND task_key=? AND status IN ('blocked','pending')`, reason, runID, taskKey); err != nil {
				return err
			}
		} else if req.Status == StatusSuccess {
			if err = r.advanceRollout(ctx, tx, runID, taskKey, commandBatch, currentBatch); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func isTerminalCommandStatus(status string) bool {
	return status == StatusSuccess || status == StatusFailed || status == StatusRolledBack
}

const (
	rolloutWait = iota
	rolloutFail
	rolloutRelease
	rolloutComplete
)

func decideRolloutTransition(reportStatus string, commandBatch, currentBatch, remaining, nextBatch int) int {
	if reportStatus == StatusFailed || reportStatus == StatusRolledBack {
		return rolloutFail
	}
	if reportStatus != StatusSuccess || commandBatch != currentBatch || remaining > 0 {
		return rolloutWait
	}
	if nextBatch > 0 {
		return rolloutRelease
	}
	return rolloutComplete
}

func (r *Repository) advanceRollout(ctx context.Context, tx *sql.Tx, runID int64, taskKey string, commandBatch, currentBatch int) error {
	if commandBatch != currentBatch {
		return nil
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM host_deployment_command WHERE workflow_run_id=? AND task_key=? AND batch_number=? AND status<>'success'`, runID, taskKey, currentBatch).Scan(&remaining); err != nil || remaining > 0 {
		return err
	}
	var nextBatch sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT MIN(batch_number) FROM host_deployment_command WHERE workflow_run_id=? AND task_key=? AND status='blocked'`, runID, taskKey).Scan(&nextBatch); err != nil {
		return err
	}
	next := 0
	if nextBatch.Valid {
		next = int(nextBatch.Int64)
	}
	if decideRolloutTransition(StatusSuccess, commandBatch, currentBatch, remaining, next) == rolloutRelease {
		var interval int
		if err := tx.QueryRowContext(ctx, `SELECT interval_seconds FROM host_deployment_rollout WHERE workflow_run_id=? AND task_key=?`, runID, taskKey).Scan(&interval); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE host_deployment_command SET status='pending',available_at=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL ? SECOND) WHERE workflow_run_id=? AND task_key=? AND batch_number=? AND status='blocked'`, interval, runID, taskKey, nextBatch.Int64); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE host_deployment_rollout SET current_batch=? WHERE workflow_run_id=? AND task_key=?`, nextBatch.Int64, runID, taskKey)
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE host_deployment_rollout SET status='success',finished_at=CURRENT_TIMESTAMP WHERE workflow_run_id=? AND task_key=?`, runID, taskKey)
	return err
}

func (r *Repository) List(ctx context.Context, projectID int64, scope authz.ScopeSet) ([]Agent, error) {
	predicate, args := scope.SQLPredicate("project_id", "", "")
	query := `SELECT id,project_id,host_group_ref,asset_ref,name,version,os,arch,CASE WHEN last_heartbeat<DATE_SUB(CURRENT_TIMESTAMP,INTERVAL 3 MINUTE) THEN 'offline' ELSE status END,last_heartbeat FROM host_agent WHERE ` + predicate
	if projectID > 0 {
		query += " AND project_id=?"
		args = append(args, projectID)
	}
	query += " ORDER BY id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []Agent
	for rows.Next() {
		var agent Agent
		if err := rows.Scan(&agent.ID, &agent.ProjectID, &agent.HostGroupRef, &agent.AssetRef, &agent.Name, &agent.Version, &agent.OS, &agent.Arch, &agent.Status, &agent.LastHeartbeat); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (r *Repository) Enqueue(ctx context.Context, commands []DeploymentCommand, policy RolloutPolicy) error {
	if len(commands) == 0 {
		return fmt.Errorf("no host deployment targets")
	}
	prepared, targetHash, totalBatches, err := prepareRollout(commands, policy)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	first := prepared[0].Command
	if _, err = tx.ExecContext(ctx, `INSERT IGNORE INTO host_deployment_rollout(workflow_run_id,task_key,project_id,service_id,service_revision_id,batch_size,interval_seconds,total_targets,total_batches,target_hash,status,current_batch) VALUES(?,?,?,?,?,?,?,?,?,?,'running',1)`, first.WorkflowRunID, first.TaskKey, first.ProjectID, first.ServiceID, first.ServiceRevisionID, policy.BatchSize, policy.IntervalSeconds, len(prepared), totalBatches, targetHash); err != nil {
		return err
	}
	var storedHash string
	var storedBatchSize, storedInterval, storedTargets int
	if err = tx.QueryRowContext(ctx, `SELECT target_hash,batch_size,interval_seconds,total_targets FROM host_deployment_rollout WHERE workflow_run_id=? AND task_key=? FOR UPDATE`, first.WorkflowRunID, first.TaskKey).Scan(&storedHash, &storedBatchSize, &storedInterval, &storedTargets); err != nil {
		return err
	}
	if storedHash != targetHash || storedBatchSize != policy.BatchSize || storedInterval != policy.IntervalSeconds || storedTargets != len(prepared) {
		return fmt.Errorf("host rollout target set changed for an existing workflow task")
	}
	for _, item := range prepared {
		command := item.Command
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO host_deployment_command(workflow_run_id,task_key,project_id,service_id,service_revision_id,agent_id,artifact_url,artifact_sha256,deploy_script,health_script,rollback_script,timeout_seconds,batch_number,status) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, command.WorkflowRunID, command.TaskKey, command.ProjectID, command.ServiceID, command.ServiceRevisionID, command.AgentID, command.ArtifactURL, command.ArtifactSHA256, command.DeployScript, command.HealthScript, command.RollbackScript, command.TimeoutSeconds, item.BatchNumber, item.Status)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

type preparedDeployment struct {
	Command     DeploymentCommand
	BatchNumber int
	Status      string
}

func prepareRollout(commands []DeploymentCommand, policy RolloutPolicy) ([]preparedDeployment, string, int, error) {
	if policy.BatchSize <= 0 || policy.BatchSize > 1000 || policy.IntervalSeconds < 0 || policy.IntervalSeconds > 3600 {
		return nil, "", 0, fmt.Errorf("invalid host rollout policy")
	}
	ordered := append([]DeploymentCommand(nil), commands...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].AgentID < ordered[j].AgentID })
	identity := ordered[0]
	hashInput := strings.Builder{}
	prepared := make([]preparedDeployment, 0, len(ordered))
	seen := make(map[int64]struct{}, len(ordered))
	for index, command := range ordered {
		if command.AgentID <= 0 || command.WorkflowRunID != identity.WorkflowRunID || command.TaskKey != identity.TaskKey || command.ProjectID != identity.ProjectID || command.ServiceID != identity.ServiceID || command.ServiceRevisionID != identity.ServiceRevisionID {
			return nil, "", 0, fmt.Errorf("host deployment commands do not share one workflow identity")
		}
		if _, ok := seen[command.AgentID]; ok {
			return nil, "", 0, fmt.Errorf("duplicate host Agent target %d", command.AgentID)
		}
		seen[command.AgentID] = struct{}{}
		if index > 0 {
			hashInput.WriteByte(',')
		}
		hashInput.WriteString(strconv.FormatInt(command.AgentID, 10))
		batch := index/policy.BatchSize + 1
		status := StatusBlocked
		if batch == 1 {
			status = StatusPending
		}
		prepared = append(prepared, preparedDeployment{Command: command, BatchNumber: batch, Status: status})
	}
	digest := sha256.Sum256([]byte(hashInput.String()))
	totalBatches := (len(ordered) + policy.BatchSize - 1) / policy.BatchSize
	return prepared, hex.EncodeToString(digest[:]), totalBatches, nil
}

func (r *Repository) DeploymentState(ctx context.Context, runID int64, taskKey string) (DeploymentState, error) {
	var state DeploymentState
	err := r.db.QueryRowContext(ctx, `SELECT status,current_batch,total_batches,error_message FROM host_deployment_rollout WHERE workflow_run_id=? AND task_key=?`, runID, taskKey).Scan(&state.RolloutStatus, &state.CurrentBatch, &state.TotalBatches, &state.ErrorMessage)
	if err != nil {
		return state, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(status='blocked'),0),COALESCE(SUM(status IN ('pending','claimed')),0),COALESCE(SUM(status='running'),0),COALESCE(SUM(status='success'),0),COALESCE(SUM(status='failed'),0),COALESCE(SUM(status='rolled_back'),0),COALESCE(SUM(status='cancelled'),0) FROM host_deployment_command WHERE workflow_run_id=? AND task_key=?`, runID, taskKey).Scan(&state.Total, &state.Blocked, &state.Pending, &state.Running, &state.Succeeded, &state.Failed, &state.RolledBack, &state.Cancelled)
	return state, err
}

func tail(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	return value[len(value)-limit:], true
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(strings.ToLower(value))
	return err == nil
}
