package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func (r *Repository) FindConcurrencyPolicy(ctx context.Context, projectID int64) (*ConcurrencyPolicy, error) {
	scopeType := "project"
	scopeID := projectID
	if projectID <= 0 {
		scopeType, scopeID = "platform", 0
	}
	var policy ConcurrencyPolicy
	err := r.db.QueryRowContext(ctx, `SELECT scope_type,scope_id,max_running,queue_timeout_seconds,updated_by,updated_at
		FROM build_concurrency_policy WHERE scope_type=? AND scope_id=?`, scopeType, scopeID).Scan(
		&policy.ScopeType, &policy.ScopeID, &policy.MaxRunning, &policy.QueueTimeoutSeconds, &policy.UpdatedBy, &policy.UpdatedAt)
	if err == nil {
		return &policy, nil
	}
	if err != sql.ErrNoRows || projectID <= 0 {
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT scope_type,scope_id,max_running,queue_timeout_seconds,updated_by,updated_at
		FROM build_concurrency_policy WHERE scope_type='platform' AND scope_id=0`).Scan(
		&policy.ScopeType, &policy.ScopeID, &policy.MaxRunning, &policy.QueueTimeoutSeconds, &policy.UpdatedBy, &policy.UpdatedAt)
	if err != nil {
		return nil, err
	}
	policy.Inherited = true
	return &policy, nil
}

func (r *Repository) FindPlatformBuildSettings(ctx context.Context) (*PlatformBuildSettings, error) {
	return findPlatformBuildSettings(ctx, r.db)
}

func findPlatformBuildSettings(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}) (*PlatformBuildSettings, error) {
	var settings PlatformBuildSettings
	var policy string
	err := db.QueryRowContext(ctx, `SELECT enabled,cluster_id,namespace,builder_image,build_command,
		timeout_seconds,resource_cpu,resource_memory,cache_pvc_name,cache_dir,job_retention_seconds,updated_by,updated_at,COALESCE(execution_policy,'{}')
		FROM build_platform_settings WHERE id=1`).Scan(
		&settings.Enabled, &settings.ClusterID, &settings.Namespace, &settings.BuilderImage,
		&settings.BuildCommand, &settings.TimeoutSeconds, &settings.ResourceCPU, &settings.ResourceMemory,
		&settings.CachePVCName, &settings.CacheDir, &settings.JobRetentionSeconds, &settings.UpdatedBy, &settings.UpdatedAt, &policy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(policy), &settings.ExecutionPolicy); err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *Repository) SavePlatformBuildSettings(ctx context.Context, request SavePlatformBuildSettingsRequest, userID int64) error {
	policy, err := json.Marshal(request.ExecutionPolicy)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock in the same order as new command claims; do not move a running
	// execution to another cluster when the worker reconnects after restart.
	var concurrency int
	if err = tx.QueryRowContext(ctx, `SELECT max_running FROM build_concurrency_policy WHERE scope_type='platform' AND scope_id=0 FOR UPDATE`).Scan(&concurrency); err != nil {
		return err
	}
	var oldCluster int64
	var oldNamespace, oldPolicy string
	err = tx.QueryRowContext(ctx, `SELECT cluster_id,namespace,COALESCE(execution_policy,'{}') FROM build_platform_settings WHERE id=1 FOR UPDATE`).Scan(&oldCluster, &oldNamespace, &oldPolicy)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == sql.ErrNoRows {
		var active int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_run WHERE status IN ('dispatching','running','cancelling') OR cleanup_pending=1`).Scan(&active); err != nil {
			return err
		}
		if active > 0 {
			return ErrBuildPoolBusy
		}
		err = sql.ErrNoRows
	}
	var previous ExecutionPolicy
	if err == nil {
		if err = json.Unmarshal([]byte(oldPolicy), &previous); err != nil {
			return err
		}
		previousJSON, _ := json.Marshal(previous)
		if oldCluster != request.ClusterID || oldNamespace != request.Namespace || string(previousJSON) != string(policy) {
			var active int
			if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM build_run WHERE status IN ('dispatching','running','cancelling') OR cleanup_pending=1`).Scan(&active); err != nil {
				return err
			}
			if active > 0 {
				return ErrBuildPoolBusy
			}
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO build_platform_settings
		(id,enabled,cluster_id,namespace,builder_image,build_command,timeout_seconds,resource_cpu,resource_memory,cache_pvc_name,cache_dir,job_retention_seconds,updated_by,execution_policy)
		VALUES (1,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE enabled=VALUES(enabled),cluster_id=VALUES(cluster_id),namespace=VALUES(namespace),
		builder_image=VALUES(builder_image),build_command=VALUES(build_command),timeout_seconds=VALUES(timeout_seconds),
		resource_cpu=VALUES(resource_cpu),resource_memory=VALUES(resource_memory),cache_pvc_name=VALUES(cache_pvc_name),
		cache_dir=VALUES(cache_dir),job_retention_seconds=VALUES(job_retention_seconds),updated_by=VALUES(updated_by),execution_policy=VALUES(execution_policy)`,
		request.Enabled, request.ClusterID, request.Namespace, request.BuilderImage, request.BuildCommand,
		request.TimeoutSeconds, request.ResourceCPU, request.ResourceMemory, request.CachePVCName, request.CacheDir,
		request.JobRetentionSeconds, userID, string(policy))
	if err != nil {
		return err
	}
	return tx.Commit()
}

var ErrBuildPoolBusy = errors.New("build pool has active executions")

func (l *Logic) GetPlatformBuildSettings(ctx context.Context, userID int64) (*PlatformBuildSettings, int) {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_governance", Action: "view"})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	settings, err := l.repo.FindPlatformBuildSettings(ctx)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return settings, errcode.CodeSuccess
}

func (l *Logic) SavePlatformBuildSettings(ctx context.Context, userID int64, request SavePlatformBuildSettingsRequest) (*PlatformBuildSettings, int) {
	if request.ExecutionPolicy.Validate() != nil {
		return nil, errcode.CodeParamInvalid
	}
	request.Namespace = strings.TrimSpace(request.Namespace)
	request.BuilderImage = strings.TrimSpace(request.BuilderImage)
	request.BuildCommand = strings.TrimSpace(request.BuildCommand)
	if request.BuildCommand == "" {
		request.BuildCommand = DefaultPlatformBuildCommand
	}
	request.ResourceCPU = strings.TrimSpace(request.ResourceCPU)
	request.ResourceMemory = strings.TrimSpace(request.ResourceMemory)
	request.CachePVCName = strings.TrimSpace(request.CachePVCName)
	request.CacheDir = strings.TrimSpace(request.CacheDir)
	if request.Enabled {
		if request.ClusterID <= 0 || request.Namespace == "" || request.BuilderImage == "" || request.BuildCommand == "" {
			return nil, errcode.CodeParamInvalid
		}
	}
	if request.TimeoutSeconds < 60 || request.TimeoutSeconds > 86400 ||
		request.JobRetentionSeconds < 0 || request.JobRetentionSeconds > 604800 {
		return nil, errcode.CodeParamInvalid
	}
	if _, err := buildResources(request.ResourceCPU, request.ResourceMemory); err != nil {
		return nil, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_governance", Action: "update"})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	if err := l.repo.SavePlatformBuildSettings(ctx, request, userID); err != nil {
		if errors.Is(err, ErrBuildPoolBusy) {
			return nil, errcode.CodeBuildPoolBusy
		}
		return nil, errcode.CodeDBError
	}
	return l.GetPlatformBuildSettings(ctx, userID)
}

func (r *Repository) SaveConcurrencyPolicy(ctx context.Context, projectID int64, maxRunning, queueTimeout int, userID int64) error {
	scopeType := "project"
	scopeID := projectID
	if projectID <= 0 {
		scopeType, scopeID = "platform", 0
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO build_concurrency_policy(scope_type,scope_id,max_running,queue_timeout_seconds,updated_by)
		VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE max_running=VALUES(max_running),queue_timeout_seconds=VALUES(queue_timeout_seconds),updated_by=VALUES(updated_by)`,
		scopeType, scopeID, maxRunning, queueTimeout, userID)
	return err
}

func (r *Repository) QueryBuildMetrics(ctx context.Context, scope authz.ScopeSet, projectID, serviceID int64, from, to time.Time) (*BuildMetrics, error) {
	predicate, args := scope.SQLPredicate("project_id", "service_id", "id")
	where := predicate + ` AND created_at>=? AND created_at<? AND status IN ('success','failed','timeout','cancelled')`
	args = append(args, from, to)
	if projectID > 0 {
		where += " AND project_id=?"
		args = append(args, projectID)
	}
	if serviceID > 0 {
		where += " AND service_id=?"
		args = append(args, serviceID)
	}
	metrics := &BuildMetrics{From: from, To: to, Daily: []BuildMetricPoint{}}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(status='success'),0),COALESCE(SUM(status IN ('failed','timeout')),0),COALESCE(SUM(status='cancelled'),0),COALESCE(AVG(duration_seconds),0),COALESCE(MAX(duration_seconds),0) FROM build_run WHERE `+where, args...).Scan(
		&metrics.Summary.Total, &metrics.Summary.Succeeded, &metrics.Summary.Failed, &metrics.Summary.Cancelled,
		&metrics.Summary.AverageDurationSec, &metrics.Summary.MaxDurationSec); err != nil {
		return nil, err
	}
	metrics.Summary.SuccessRate = metricRate(metrics.Summary.Succeeded, metrics.Summary.Total)
	rows, err := r.db.QueryContext(ctx, `SELECT DATE_FORMAT(created_at,'%Y-%m-%d'),COUNT(*),COALESCE(SUM(status='success'),0),COALESCE(SUM(status IN ('failed','timeout')),0),COALESCE(SUM(status='cancelled'),0),COALESCE(AVG(duration_seconds),0) FROM build_run WHERE `+where+` GROUP BY DATE(created_at) ORDER BY DATE(created_at)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var point BuildMetricPoint
		if err := rows.Scan(&point.Date, &point.Total, &point.Succeeded, &point.Failed, &point.Cancelled, &point.AverageDurationSec); err != nil {
			return nil, err
		}
		point.SuccessRate = metricRate(point.Succeeded, point.Total)
		metrics.Daily = append(metrics.Daily, point)
	}
	return metrics, rows.Err()
}

func metricRate(success, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total)
}

func (l *Logic) GetConcurrencyPolicy(ctx context.Context, userID, projectID int64) (*ConcurrencyPolicy, int) {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_governance", Action: "view", ProjectID: projectID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	policy, err := l.repo.FindConcurrencyPolicy(ctx, projectID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return policy, errcode.CodeSuccess
}

func (l *Logic) SaveConcurrencyPolicy(ctx context.Context, userID int64, request SaveConcurrencyPolicyRequest) (*ConcurrencyPolicy, int) {
	if request.ProjectID < 0 || request.MaxRunning < 1 || request.MaxRunning > 1000 || request.QueueTimeoutSeconds < 60 || request.QueueTimeoutSeconds > 604800 {
		return nil, errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_governance", Action: "update", ProjectID: request.ProjectID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	if err := l.repo.SaveConcurrencyPolicy(ctx, request.ProjectID, request.MaxRunning, request.QueueTimeoutSeconds, userID); err != nil {
		return nil, errcode.CodeDBError
	}
	return l.GetConcurrencyPolicy(ctx, userID, request.ProjectID)
}

func (l *Logic) Metrics(ctx context.Context, userID, projectID, serviceID int64, days int) (*BuildMetrics, int) {
	if projectID < 0 || serviceID < 0 || days < 1 || days > 365 {
		return nil, errcode.CodeParamInvalid
	}
	scope, err := l.authorizer.Scopes(ctx, userID, "build", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !scope.HasAnyAllow() {
		return nil, errcode.CodeNoPermission
	}
	to := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	from := to.AddDate(0, 0, -days)
	metrics, err := l.repo.QueryBuildMetrics(ctx, scope, projectID, serviceID, from, to)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return metrics, errcode.CodeSuccess
}

func validateConcurrencyLimit(active, limit int) error {
	if limit <= 0 {
		return fmt.Errorf("build concurrency limit must be positive")
	}
	if active < 0 || active >= limit {
		return fmt.Errorf("build concurrency limit reached")
	}
	return nil
}
