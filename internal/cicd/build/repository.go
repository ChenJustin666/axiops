package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/notification"
	"github.com/ChenJustin666/axiops/internal/cicd/servicestate"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

type Repository struct {
	db                  *sql.DB
	checkWorkerSettings bool
	workerSettingsJSON  string
}

// Fence new dispatch when the saved executor settings change. Observation and
// recovery keep using the original connection until a controlled restart.
func (r *Repository) WithWorkerSettings(settings *PlatformBuildSettings) *Repository {
	r.checkWorkerSettings = true
	r.workerSettingsJSON = workerSettingsJSON(settings)
	return r
}

func workerSettingsJSON(settings *PlatformBuildSettings) string {
	if settings == nil {
		return "null"
	}
	copy := *settings
	copy.Enabled = true
	copy.UpdatedAt = time.Time{}
	copy.UpdatedBy = 0
	data, _ := json.Marshal(copy)
	return string(data)
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ==================== BuildRun ====================

func (r *Repository) FindRunByID(ctx context.Context, id int64) (*BuildRun, error) {
	query := `SELECT id, service_id, project_id, pipeline_run_id, build_number, config_snapshot, input_fingerprint,
		trigger_type, trigger_user_id, COALESCE(trigger_key,''), status, provider,
		git_branch, git_commit, git_commit_short, git_author, git_author_email, COALESCE(git_message,''),
		COALESCE(git_changed_files,'[]'), git_compare_url, git_commit_url, COALESCE(git_changes,''),
		image_url, image_digest, image_size, artifact_url, COALESCE(artifact_sha256,''), artifact_size,
		image_verification_status, artifact_verification_status, output_verification_error, outputs_verified_at,
		job_name, pod_name, node_name, COALESCE(stages,'[]'),
		queued_at, started_at, finished_at, duration_seconds, COALESCE(error_message,''), created_at
		FROM build_run WHERE id = ?`
	var b BuildRun
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.ServiceID, &b.ProjectID, &b.PipelineRunID, &b.BuildNumber, &b.ConfigSnapshot, &b.InputFingerprint,
		&b.TriggerType, &b.TriggerUserID, &b.TriggerKey, &b.Status, &b.Provider,
		&b.GitBranch, &b.GitCommit, &b.GitCommitShort, &b.GitAuthor, &b.GitAuthorEmail, &b.GitMessage,
		&b.GitChangedFiles, &b.GitCompareURL, &b.GitCommitURL, &b.GitChangesJSON,
		&b.ImageURL, &b.ImageDigest, &b.ImageSize, &b.ArtifactURL, &b.ArtifactSHA256, &b.ArtifactSize,
		&b.ImageVerificationStatus, &b.ArtifactVerificationStatus, &b.OutputVerificationError, &b.OutputsVerifiedAt,
		&b.JobName, &b.PodName, &b.NodeName, &b.Stages,
		&b.QueuedAt, &b.StartedAt, &b.FinishedAt, &b.DurationSeconds, &b.ErrorMessage, &b.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *Repository) FindNextRunningRun(ctx context.Context) (*BuildRun, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM build_run WHERE status='running' AND job_name<>'' ORDER BY started_at, id LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindRunByID(ctx, id)
}

// Rotate observations so a long build cannot starve logs/results of other runs.
func (r *Repository) FindNextRunningRunAfter(ctx context.Context, after int64) (*BuildRun, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM build_run WHERE status='running' AND job_name<>'' ORDER BY (id>?) DESC,id LIMIT 1`, after).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindRunByID(ctx, id)
}

func (r *Repository) UpdateRunWaitReason(ctx context.Context, id int64, reason string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET error_message=? WHERE id=? AND status='running'`, reason, id)
	return err
}

func (r *Repository) ListRuns(ctx context.Context, scope authz.ScopeSet, serviceID, projectID int64, status string, page, pageSize int) ([]BuildRun, int64, error) {
	var conditions []string
	var args []interface{}

	scopePredicate, scopeArgs := scope.SQLPredicate("project_id", "service_id", "id")
	conditions = append(conditions, scopePredicate)
	args = append(args, scopeArgs...)

	if serviceID > 0 {
		conditions = append(conditions, "service_id = ?")
		args = append(args, serviceID)
	}
	if projectID > 0 {
		conditions = append(conditions, "project_id = ?")
		args = append(args, projectID)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + join(conditions, " AND ")
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM build_run"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	dataQuery := `SELECT id, service_id, project_id, pipeline_run_id, build_number, config_snapshot, input_fingerprint,
		trigger_type, trigger_user_id, COALESCE(trigger_key,''), status, provider,
		git_branch, git_commit, git_commit_short, git_author, git_author_email, COALESCE(git_message,''),
		COALESCE(git_changed_files,'[]'), git_compare_url, git_commit_url,
		image_url, image_digest, image_size, artifact_url, COALESCE(artifact_sha256,''), artifact_size,
		image_verification_status, artifact_verification_status, output_verification_error, outputs_verified_at,
		job_name, pod_name, node_name, COALESCE(stages,'[]'),
		queued_at, started_at, finished_at, duration_seconds, COALESCE(error_message,''), created_at
		FROM build_run` + where + ` ORDER BY build_number DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []BuildRun
	for rows.Next() {
		var b BuildRun
		if err := rows.Scan(
			&b.ID, &b.ServiceID, &b.ProjectID, &b.PipelineRunID, &b.BuildNumber, &b.ConfigSnapshot, &b.InputFingerprint,
			&b.TriggerType, &b.TriggerUserID, &b.TriggerKey, &b.Status, &b.Provider,
			&b.GitBranch, &b.GitCommit, &b.GitCommitShort, &b.GitAuthor, &b.GitAuthorEmail, &b.GitMessage,
			&b.GitChangedFiles, &b.GitCompareURL, &b.GitCommitURL,
			&b.ImageURL, &b.ImageDigest, &b.ImageSize, &b.ArtifactURL, &b.ArtifactSHA256, &b.ArtifactSize,
			&b.ImageVerificationStatus, &b.ArtifactVerificationStatus, &b.OutputVerificationError, &b.OutputsVerifiedAt,
			&b.JobName, &b.PodName, &b.NodeName, &b.Stages,
			&b.QueuedAt, &b.StartedAt, &b.FinishedAt, &b.DurationSeconds, &b.ErrorMessage, &b.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		runs = append(runs, b)
	}
	return runs, total, rows.Err()
}

func (r *Repository) AllocateBuildNumber(ctx context.Context, serviceID int64) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"INSERT IGNORE INTO build_number_sequence (service_id, current_number) VALUES (?, 0)", serviceID,
	); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE build_number_sequence SET current_number = current_number + 1 WHERE service_id = ?", serviceID,
	); err != nil {
		return 0, err
	}
	var number int
	if err := tx.QueryRowContext(ctx,
		"SELECT current_number FROM build_number_sequence WHERE service_id = ?", serviceID,
	).Scan(&number); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return number, nil
}

func (r *Repository) CreateRunWithSubmitCommand(ctx context.Context, b *BuildRun) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := projectstate.LockWritable(ctx, tx, b.ProjectID); err != nil {
		return 0, err
	}
	// Serialize durable submission against workflow cancellation. An expired
	// worker must not enqueue work after cancellation already confirmed no build.
	if b.WorkflowRunID > 0 {
		var state string
		var projectID int64
		if err := tx.QueryRowContext(ctx, `SELECT status,project_id FROM workflow_run WHERE id=? FOR UPDATE`, b.WorkflowRunID).Scan(&state, &projectID); err != nil {
			return 0, err
		}
		if state != "running" || projectID != b.ProjectID || b.WorkflowNodeKey == "" || b.TriggerKey == "" {
			return 0, fmt.Errorf("workflow no longer accepts build submissions")
		}
		var owned int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_task_run WHERE workflow_run_id=? AND node_key=? AND status='running' AND locked_by<>'' AND lease_until>=CURRENT_TIMESTAMP`, b.WorkflowRunID, b.WorkflowNodeKey).Scan(&owned); err != nil {
			return 0, err
		}
		if owned != 1 {
			return 0, fmt.Errorf("workflow submission lease expired")
		}
	}
	if err := servicestate.LockActive(ctx, tx, b.ProjectID, b.ServiceID); err != nil {
		return 0, err
	}
	if err := freezeGitBaseline(ctx, tx, b); err != nil {
		return 0, err
	}

	now := time.Now()
	query := `INSERT INTO build_run (service_id, project_id, pipeline_run_id, build_number, config_snapshot, input_fingerprint,
		trigger_type, trigger_user_id, trigger_key, status, provider,
		git_branch, git_commit, git_commit_short, git_author, git_author_email, git_message,
		git_changed_files, git_compare_url, git_commit_url, image_url, job_name, queued_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)`
	result, err := tx.ExecContext(ctx, query,
		b.ServiceID, b.ProjectID, b.PipelineRunID, b.BuildNumber, b.ConfigSnapshot, b.InputFingerprint,
		b.TriggerType, b.TriggerUserID, b.TriggerKey, b.Status, b.Provider,
		b.GitBranch, b.GitCommit, b.GitCommitShort, b.GitAuthor, b.GitAuthorEmail, b.GitMessage,
		b.GitChangedFiles, b.GitCompareURL, b.GitCommitURL, b.ImageURL, b.JobName, now,
	)
	if err != nil {
		return 0, err
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	payload, err := json.Marshal(map[string]int64{"build_run_id": runID})
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO build_command
		(build_run_id, command_type, provider, payload, status, idempotency_key)
		VALUES (?, 'submit', ?, ?, 'pending', ?)`,
		runID, b.Provider, payload, fmt.Sprintf("build:%d:submit", runID)); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

func (r *Repository) FindRunByTriggerKey(ctx context.Context, serviceID int64, key string) (*BuildRun, error) {
	if serviceID <= 0 || strings.TrimSpace(key) == "" {
		return nil, nil
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM build_run WHERE service_id=? AND trigger_key=?`, serviceID, key).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindRunByID(ctx, id)
}

// FindReusableRun returns an active or successful run for the exact immutable
// source/config input. Failed runs are intentionally excluded so users can
// retry after fixing a transient registry or runner failure.
func (r *Repository) FindReusableRun(ctx context.Context, serviceID int64, fingerprint string) (*BuildRun, error) {
	if serviceID <= 0 || strings.TrimSpace(fingerprint) == "" {
		return nil, nil
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM build_run
		WHERE service_id=? AND input_fingerprint=? AND status='success'
		AND ((image_digest<>'' AND image_verification_status='verified')
		OR (artifact_sha256<>'' AND artifact_verification_status='verified'))
		ORDER BY CASE status WHEN 'success' THEN 0 ELSE 1 END, id DESC LIMIT 1`, serviceID, fingerprint).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.FindRunByID(ctx, id)
}

func (r *Repository) TransitionRunStatus(ctx context.Context, id int64, status string, errMsg string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var current string
	var projectID, triggerUserID int64
	var buildNumber int
	if err := tx.QueryRowContext(ctx, "SELECT status, project_id, trigger_user_id, build_number FROM build_run WHERE id = ? FOR UPDATE", id).Scan(&current, &projectID, &triggerUserID, &buildNumber); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}
	if err := ValidateTransition(current, status); err != nil {
		return err
	}
	if current == status {
		return tx.Commit()
	}

	now := time.Now()
	switch status {
	case StatusRunning:
		_, err = tx.ExecContext(ctx, "UPDATE build_run SET status=?, started_at=? WHERE id=?", status, now, id)
	case StatusSuccess, StatusFailed, StatusTimeout, StatusCancelled:
		_, err = tx.ExecContext(ctx,
			"UPDATE build_run SET status=?, finished_at=?, duration_seconds=TIMESTAMPDIFF(SECOND, COALESCE(started_at, queued_at), ?), error_message=?, cleanup_pending=(job_name<>'' OR provider='kube_job'),cleanup_retry_at=CURRENT_TIMESTAMP WHERE id=?",
			status, now, now, errMsg, id)
	default:
		_, err = tx.ExecContext(ctx, "UPDATE build_run SET status=? WHERE id=?", status, id)
	}
	if err != nil {
		return err
	}
	if status == StatusSuccess || status == StatusFailed || status == StatusTimeout || status == StatusCancelled {
		message := notification.Message{EventType: "build.terminal", Title: fmt.Sprintf("Build #%d %s", buildNumber, status), Content: errMsg, Status: status, ProjectID: projectID, ResourceID: id, URL: fmt.Sprintf("/cicd/builds/%d", id)}
		if err := notification.EnqueueTerminal(ctx, tx, projectID, triggerUserID, "build.terminal", id, status, message); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) RequestCancel(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, provider string
	if err := tx.QueryRowContext(ctx,
		"SELECT status, provider FROM build_run WHERE id = ? FOR UPDATE", id,
	).Scan(&status, &provider); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRunNotFound
		}
		return err
	}
	if status != StatusCancelling {
		if err := ValidateTransition(status, StatusCancelling); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE build_run SET status = ?, error_message = ? WHERE id = ?", StatusCancelling, "用户请求取消", id,
		); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(map[string]int64{"build_run_id": id})
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO build_command
		(build_run_id, command_type, provider, payload, status, idempotency_key)
		VALUES (?, 'cancel', ?, ?, 'pending', ?)`,
		id, provider, payload, fmt.Sprintf("build:%d:cancel", id)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) UpdateRunExecInfo(ctx context.Context, id int64, jobName, podName, nodeName string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE build_run SET job_name=?, pod_name=?, node_name=? WHERE id=?", jobName, podName, nodeName, id)
	return err
}

func (r *Repository) UpdateRunArtifact(ctx context.Context, id int64, imageURL, imageDigest string, imageSize int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE build_run SET image_url=?, image_digest=?, image_size=? WHERE id=?", imageURL, imageDigest, imageSize, id)
	return err
}

func (r *Repository) UpdateRunOutputs(ctx context.Context, id int64, imageURL, imageDigest string, imageSize int64, artifactURL, artifactSHA256 string, artifactSize int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET image_url=?, image_digest=?, image_size=?, artifact_url=?, artifact_sha256=?, artifact_size=? WHERE id=?`,
		imageURL, imageDigest, imageSize, artifactURL, artifactSHA256, artifactSize, id)
	return err
}

func (r *Repository) UpdateRunVerification(ctx context.Context, id int64, verification OutputVerification) error {
	_, err := r.db.ExecContext(ctx, `UPDATE build_run SET image_verification_status=?, artifact_verification_status=?, output_verification_error=?, outputs_verified_at=? WHERE id=?`,
		verification.ImageStatus, verification.ArtifactStatus, verification.Error, verification.VerifiedAt, id)
	return err
}

func (r *Repository) UpdateRunStages(ctx context.Context, id int64, stages []BuildStage) error {
	data, _ := json.Marshal(stages)
	_, err := r.db.ExecContext(ctx, "UPDATE build_run SET stages=? WHERE id=?", string(data), id)
	return err
}

// ==================== BuildConfig ====================

func (r *Repository) FindConfigByServiceID(ctx context.Context, serviceID int64) (*BuildConfig, error) {
	query := `SELECT id, service_id, template_type, build_mode, build_image, COALESCE(build_command,''), dockerfile_path, build_context,
		image_repo, cache_enabled, cache_dir, cache_strategy, cache_pvc_name, timeout_seconds, branch_rules, env_vars,
		resource_cpu, resource_memory, provider, jenkins_job_name, host_agent_id,
		created_by, created_at, updated_at,dockerfile_source,COALESCE(dockerfile_content,'') FROM build_config WHERE service_id = ?`
	var c BuildConfig
	err := r.db.QueryRowContext(ctx, query, serviceID).Scan(
		&c.ID, &c.ServiceID, &c.TemplateType, &c.BuildMode, &c.BuildImage, &c.BuildCommand, &c.DockerfilePath, &c.BuildContext,
		&c.ImageRepo, &c.CacheEnabled, &c.CacheDir, &c.CacheStrategy, &c.CachePVCName, &c.TimeoutSeconds, &c.BranchRules, &c.EnvVars,
		&c.ResourceCPU, &c.ResourceMemory, &c.Provider, &c.JenkinsJobName, &c.HostAgentID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt, &c.DockerfileSource, &c.DockerfileContent,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UpsertConfig(ctx context.Context, c *BuildConfig) error {
	return upsertConfig(ctx, r.db, c)
}

func (r *Repository) SaveConfigWithBinding(ctx context.Context, c *BuildConfig, binding *BuildServiceBinding) error {
	if binding == nil {
		return r.UpsertConfig(ctx, c)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM app_service WHERE id=? AND status='active' FOR UPDATE`, c.ServiceID).Scan(&id); err != nil {
		return err
	}
	var imageName *string
	if binding.RegistryID != nil {
		imageName = &c.ImageRepo
	}
	_, err = tx.ExecContext(ctx, `UPDATE app_service SET code_source_id=COALESCE(?,code_source_id),
		repo_url=COALESCE(?,repo_url), repo_branch=COALESCE(?,repo_branch),
		registry_id=COALESCE(?,registry_id), image_name=COALESCE(?,image_name) WHERE id=?`,
		binding.CodeSourceID, binding.RepoURL, binding.RepoBranch, binding.RegistryID, imageName, c.ServiceID)
	if err != nil {
		return err
	}
	if err = upsertConfig(ctx, tx, c); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertConfig(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}, c *BuildConfig) error {
	query := `INSERT INTO build_config (service_id, template_type, build_mode, build_image, build_command, dockerfile_path,
		build_context, image_repo, cache_enabled, cache_dir, cache_strategy, cache_pvc_name, timeout_seconds, branch_rules, env_vars,
		resource_cpu, resource_memory, provider, jenkins_job_name, host_agent_id, created_by,dockerfile_source,dockerfile_content)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?,?)
		ON DUPLICATE KEY UPDATE
		template_type=VALUES(template_type), build_mode=VALUES(build_mode), build_image=VALUES(build_image), build_command=VALUES(build_command),
		dockerfile_path=VALUES(dockerfile_path), build_context=VALUES(build_context), image_repo=VALUES(image_repo),
		cache_enabled=VALUES(cache_enabled), cache_dir=VALUES(cache_dir), cache_strategy=VALUES(cache_strategy), cache_pvc_name=VALUES(cache_pvc_name), timeout_seconds=VALUES(timeout_seconds),
		branch_rules=VALUES(branch_rules), env_vars=VALUES(env_vars), resource_cpu=VALUES(resource_cpu),
		resource_memory=VALUES(resource_memory), provider=VALUES(provider), jenkins_job_name=VALUES(jenkins_job_name),
		host_agent_id=VALUES(host_agent_id),dockerfile_source=VALUES(dockerfile_source),dockerfile_content=VALUES(dockerfile_content)`
	_, err := db.ExecContext(ctx, query,
		c.ServiceID, c.TemplateType, c.BuildMode, c.BuildImage, c.BuildCommand, c.DockerfilePath,
		c.BuildContext, c.ImageRepo, c.CacheEnabled, c.CacheDir, c.CacheStrategy, c.CachePVCName, c.TimeoutSeconds, c.BranchRules, c.EnvVars,
		c.ResourceCPU, c.ResourceMemory, c.Provider, c.JenkinsJobName, c.HostAgentID, c.CreatedBy, defaultString(c.DockerfileSource, "repository"), c.DockerfileContent,
	)
	return err
}

// ==================== BuildLog ====================

func (r *Repository) CreateLog(ctx context.Context, log *BuildLog) (int64, error) {
	query := `INSERT INTO build_log (build_run_id, stage, log_url, log_text, line_count, truncated, log_key, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE log_url=VALUES(log_url), log_text=VALUES(log_text), line_count=VALUES(line_count),
		truncated=VALUES(truncated), started_at=COALESCE(started_at, VALUES(started_at)), finished_at=VALUES(finished_at)`
	logKey := fmt.Sprintf("build:%d:%s", log.BuildRunID, log.Stage)
	result, err := r.db.ExecContext(ctx, query, log.BuildRunID, log.Stage, log.LogURL, log.LogText, log.LineCount, log.Truncated, logKey, log.StartedAt, log.FinishedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *Repository) ListLogsByRunID(ctx context.Context, buildRunID int64) ([]BuildLog, error) {
	query := `SELECT id, build_run_id, stage, COALESCE(log_url, ''), COALESCE(log_text, ''), line_count, truncated, started_at, finished_at, created_at
		FROM build_log WHERE build_run_id = ? ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query, buildRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]BuildLog, 0)
	for rows.Next() {
		var l BuildLog
		if err := rows.Scan(&l.ID, &l.BuildRunID, &l.Stage, &l.LogURL, &l.LogText, &l.LineCount, &l.Truncated, &l.StartedAt, &l.FinishedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ==================== BuildTemplate ====================

func (r *Repository) ListTemplates(ctx context.Context) ([]BuildTemplate, error) {
	query := `SELECT id, name, type, description, build_image, build_command, cache_dir, detect_files, is_builtin, created_by, created_at, updated_at
		FROM build_template ORDER BY is_builtin DESC, id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []BuildTemplate
	for rows.Next() {
		var t BuildTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Type, &t.Description, &t.BuildImage, &t.BuildCommand, &t.CacheDir, &t.DetectFiles, &t.IsBuiltin, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

func (r *Repository) FindTemplateByType(ctx context.Context, templateType string) (*BuildTemplate, error) {
	query := `SELECT id, name, type, description, build_image, build_command, cache_dir, detect_files, is_builtin, created_by, created_at, updated_at
		FROM build_template WHERE type = ? AND is_builtin = 1 LIMIT 1`
	var t BuildTemplate
	err := r.db.QueryRowContext(ctx, query, templateType).Scan(&t.ID, &t.Name, &t.Type, &t.Description, &t.BuildImage, &t.BuildCommand, &t.CacheDir, &t.DetectFiles, &t.IsBuiltin, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// helper
func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
