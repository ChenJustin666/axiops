package build

import (
	"encoding/json"
	"time"
)

// BuildRun 构建运行记录（完整版）
type BuildRun struct {
	WorkflowRunID    int64   `json:"-"`
	WorkflowNodeKey  string  `json:"-"`
	ID               int64   `json:"id"`
	ServiceID        int64   `json:"service_id"`
	ProjectID        int64   `json:"project_id"`
	PipelineRunID    int64   `json:"pipeline_run_id"`
	BuildNumber      int     `json:"build_number"`
	ConfigSnapshot   *string `json:"config_snapshot,omitempty"` // JSON string
	InputFingerprint string  `json:"input_fingerprint,omitempty"`
	TriggerType      string  `json:"trigger_type"`
	TriggerUserID    int64   `json:"trigger_user_id"`
	TriggerKey       string  `json:"trigger_key,omitempty"`
	Status           string  `json:"status"`
	Provider         string  `json:"provider"`

	// Git 信息
	GitBranch       string `json:"git_branch"`
	GitCommit       string `json:"git_commit"`
	GitCommitShort  string `json:"git_commit_short"`
	GitAuthor       string `json:"git_author"`
	GitAuthorEmail  string `json:"git_author_email"`
	GitMessage      string `json:"git_message"`
	GitChangedFiles string `json:"git_changed_files,omitempty"` // JSON
	GitCompareURL   string `json:"git_compare_url"`
	GitCommitURL    string `json:"git_commit_url"`
	GitChangesJSON  string `json:"-"`

	// 产物
	ImageURL                   string     `json:"image_url"`
	ImageDigest                string     `json:"image_digest"`
	ImageSize                  int64      `json:"image_size"`
	ArtifactURL                string     `json:"artifact_url"`
	ArtifactSHA256             string     `json:"artifact_sha256"`
	ArtifactSize               int64      `json:"artifact_size"`
	ImageVerificationStatus    string     `json:"image_verification_status"`
	ArtifactVerificationStatus string     `json:"artifact_verification_status"`
	OutputVerificationError    string     `json:"output_verification_error,omitempty"`
	OutputsVerifiedAt          *time.Time `json:"outputs_verified_at,omitempty"`

	// 执行信息
	JobName  string `json:"job_name"`
	PodName  string `json:"pod_name"`
	NodeName string `json:"node_name"`
	Stages   string `json:"stages,omitempty"` // JSON

	// 时间
	QueuedAt        *time.Time `json:"queued_at"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	DurationSeconds int        `json:"duration_seconds"`
	ErrorMessage    string     `json:"error_message"`
	CreatedAt       time.Time  `json:"created_at"`
}

// BuildConfig 构建配置（每个服务一份）
type BuildConfig struct {
	DockerfileSource  string `json:"dockerfile_source"`
	DockerfileContent string `json:"dockerfile_content"`
	ID                int64  `json:"id"`
	ServiceID         int64  `json:"service_id"`
	TemplateType      string `json:"template_type"`
	// BuildMode controls how the builder produces the immutable output:
	// dockerfile (default), jar_image, or artifact.
	BuildMode      string    `json:"build_mode"`
	BuildImage     string    `json:"build_image"`
	BuildCommand   string    `json:"build_command"`
	DockerfilePath string    `json:"dockerfile_path"`
	BuildContext   string    `json:"build_context"`
	ImageRepo      string    `json:"image_repo"`
	CacheEnabled   bool      `json:"cache_enabled"`
	CacheStrategy  string    `json:"cache_strategy"`
	CachePVCName   string    `json:"cache_pvc_name"`
	CacheDir       string    `json:"cache_dir"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	BranchRules    *string   `json:"branch_rules,omitempty"` // JSON
	EnvVars        *string   `json:"env_vars,omitempty"`     // JSON
	ResourceCPU    string    `json:"resource_cpu"`
	ResourceMemory string    `json:"resource_memory"`
	Provider       string    `json:"provider"`
	JenkinsJobName string    `json:"jenkins_job_name"`
	HostAgentID    int64     `json:"host_agent_id"`
	CreatedBy      int64     `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BuildExecutionSnapshot freezes non-secret source and build configuration at
// trigger time so a queued command cannot change when the service is edited.
type BuildExecutionSnapshot struct {
	ChangeBaseline *GitChangeBaseline `json:"change_baseline,omitempty"`
	CodeSourceType string             `json:"code_source_type,omitempty"`
	RepositoryURL  string             `json:"repository_url"`
	CodeSourceID   int64              `json:"code_source_id,omitempty"`
	// Fingerprint identifies the immutable build input. It intentionally excludes
	// the generated image tag so a retry or another environment can reuse the
	// same verified artifact.
	Fingerprint string       `json:"fingerprint,omitempty"`
	Config      *BuildConfig `json:"config,omitempty"`
}

// BuildLog 构建日志
type BuildLog struct {
	ID         int64      `json:"id"`
	BuildRunID int64      `json:"build_run_id"`
	Stage      string     `json:"stage"`
	LogURL     string     `json:"log_url"`
	LogText    string     `json:"log_text,omitempty"`
	LineCount  int        `json:"line_count"`
	Truncated  bool       `json:"truncated"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// BuildCommand is a durable instruction consumed by axiops-task-worker.
type BuildCommand struct {
	ID             int64      `json:"id"`
	BuildRunID     int64      `json:"build_run_id"`
	CommandType    string     `json:"command_type"`
	Provider       string     `json:"provider"`
	Payload        string     `json:"payload"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LockedBy       string     `json:"locked_by"`
	LockedAt       *time.Time `json:"locked_at,omitempty"`
	LastError      string     `json:"last_error"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// BuildTemplate 构建模板
type BuildTemplate struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Description  string    `json:"description"`
	BuildImage   string    `json:"build_image"`
	BuildCommand string    `json:"build_command"`
	CacheDir     string    `json:"cache_dir"`
	DetectFiles  string    `json:"detect_files"` // JSON
	IsBuiltin    bool      `json:"is_builtin"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BuildStage 构建阶段
type BuildStage struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // pending/running/success/failed/skipped
	DurationSec int    `json:"duration_sec"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
}

// BuildResult is the only result contract a builder is allowed to publish.
// Image digest is immutable; artifact SHA-256 is required when the output is
// a downloadable file (for example a legacy JAR that cannot be containerized).
type BuildResult struct {
	Schema         string `json:"schema"`
	ImageDigest    string `json:"image_digest,omitempty"`
	ImageSize      int64  `json:"image_size,omitempty"`
	ArtifactURL    string `json:"artifact_url,omitempty"`
	ArtifactSHA256 string `json:"artifact_sha256,omitempty"`
	ArtifactSize   int64  `json:"artifact_size,omitempty"`
}

// GitFileChange 变更文件
type GitFileChange struct {
	Action string `json:"action"` // A/M/D/R
	Path   string `json:"path"`
}

// BranchRules 分支触发规则
type BranchRules struct {
	Includes []string `json:"includes"` // 包含的分支模式
	Excludes []string `json:"excludes"` // 排除的分支模式
}

// EnvVar 环境变量
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// --- Request/Response Types ---

// TriggerBuildRequest 触发构建
type TriggerBuildRequest struct {
	WorkflowRunID   int64  `json:"-"`
	WorkflowNodeKey string `json:"-"`
	ServiceID       int64  `json:"service_id" validate:"required"`
	Branch          string `json:"branch"`
	CommitID        string `json:"commit_id"`
	TriggerType     string `json:"trigger_type"` // manual/webhook/approval/schedule
	TriggerKey      string `json:"-"`
	Author          string `json:"-"`
	AuthorEmail     string `json:"-"`
	Message         string `json:"-"`
	// 高级选项
	CustomCommand string `json:"custom_command"` // 自定义构建命令（覆盖默认）
	SkipTests     bool   `json:"skip_tests"`     // 跳过测试
	CustomTag     string `json:"custom_tag"`     // 自定义镜像 Tag
}

// WebhookBuildRequest is the machine-authenticated equivalent of a manual trigger.
// SourceID is checked against the service before a run is queued.
type WebhookBuildRequest struct {
	ServiceID   int64
	SourceID    int64
	Branch      string
	CommitID    string
	Author      string
	AuthorEmail string
	Message     string
	DeliveryID  string
}

// CreateBuildConfigRequest 创建/更新构建配置
type CreateBuildConfigRequest struct {
	DockerfileSource  string               `json:"dockerfile_source"`
	DockerfileContent string               `json:"dockerfile_content"`
	ServiceBinding    *BuildServiceBinding `json:"service_binding,omitempty"`
	ServiceID         int64                `json:"service_id" validate:"required"`
	TemplateType      string               `json:"template_type"`
	BuildMode         string               `json:"build_mode"`
	BuildImage        string               `json:"build_image"`
	BuildCommand      string               `json:"build_command"`
	DockerfilePath    string               `json:"dockerfile_path"`
	BuildContext      string               `json:"build_context"`
	ImageRepo         string               `json:"image_repo"`
	CacheEnabled      *bool                `json:"cache_enabled"`
	CacheDir          string               `json:"cache_dir"`
	CacheStrategy     string               `json:"cache_strategy"`
	CachePVCName      string               `json:"cache_pvc_name"`
	TimeoutSeconds    *int                 `json:"timeout_seconds"`
	BranchRules       *BranchRules         `json:"branch_rules"`
	EnvVars           []EnvVar             `json:"env_vars"`
	ResourceCPU       string               `json:"resource_cpu"`
	ResourceMemory    string               `json:"resource_memory"`
	Provider          string               `json:"provider"`
	JenkinsJobName    string               `json:"jenkins_job_name"`
	HostAgentID       *int64               `json:"host_agent_id"`
}

// BuildServiceBinding is saved in the same transaction as the build config.
// Omitted fields keep the current service binding; legacy clients can omit it entirely.
type BuildServiceBinding struct {
	CodeSourceID *int64  `json:"code_source_id,omitempty"`
	RepoURL      *string `json:"repo_url,omitempty"`
	RepoBranch   *string `json:"repo_branch,omitempty"`
	RegistryID   *int64  `json:"registry_id,omitempty"`
}

// BuildRunDetail 构建详情（包含 stages 和 git 变更文件解析）
type BuildRunDetail struct {
	BuildRun
	GitChanges            *GitChangeSet   `json:"git_changes,omitempty"`
	StagesParsed          []BuildStage    `json:"stages_parsed"`
	GitChangedFilesParsed []GitFileChange `json:"git_changed_files_parsed"`
	Logs                  []BuildLog      `json:"logs,omitempty"`
}

type ConcurrencyPolicy struct {
	ScopeType           string    `json:"scope_type"`
	ScopeID             int64     `json:"scope_id"`
	MaxRunning          int       `json:"max_running"`
	QueueTimeoutSeconds int       `json:"queue_timeout_seconds"`
	Inherited           bool      `json:"inherited"`
	UpdatedBy           int64     `json:"updated_by"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SaveConcurrencyPolicyRequest struct {
	ProjectID           int64 `json:"project_id"`
	MaxRunning          int   `json:"max_running"`
	QueueTimeoutSeconds int   `json:"queue_timeout_seconds"`
}

// PlatformBuildSettings are the single platform-level source of truth for
// CI runner binding and defaults. Secrets are referenced by other resources;
// this record intentionally contains no credentials.
type PlatformBuildSettings struct {
	ExecutionPolicy     ExecutionPolicy `json:"execution_policy"`
	Enabled             bool            `json:"enabled"`
	ClusterID           int64           `json:"cluster_id"`
	Namespace           string          `json:"namespace"`
	BuilderImage        string          `json:"builder_image"`
	BuildCommand        string          `json:"build_command"`
	TimeoutSeconds      int             `json:"timeout_seconds"`
	ResourceCPU         string          `json:"resource_cpu"`
	ResourceMemory      string          `json:"resource_memory"`
	CachePVCName        string          `json:"cache_pvc_name"`
	CacheDir            string          `json:"cache_dir"`
	JobRetentionSeconds int             `json:"job_retention_seconds"`
	UpdatedBy           int64           `json:"updated_by"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type SavePlatformBuildSettingsRequest struct {
	ExecutionPolicy     ExecutionPolicy `json:"execution_policy"`
	Enabled             bool            `json:"enabled"`
	ClusterID           int64           `json:"cluster_id"`
	Namespace           string          `json:"namespace"`
	BuilderImage        string          `json:"builder_image"`
	BuildCommand        string          `json:"build_command"`
	TimeoutSeconds      int             `json:"timeout_seconds"`
	ResourceCPU         string          `json:"resource_cpu"`
	ResourceMemory      string          `json:"resource_memory"`
	CachePVCName        string          `json:"cache_pvc_name"`
	CacheDir            string          `json:"cache_dir"`
	JobRetentionSeconds int             `json:"job_retention_seconds"`
}

type BuildMetricSummary struct {
	Total              int64   `json:"total"`
	Succeeded          int64   `json:"succeeded"`
	Failed             int64   `json:"failed"`
	Cancelled          int64   `json:"cancelled"`
	SuccessRate        float64 `json:"success_rate"`
	AverageDurationSec float64 `json:"average_duration_seconds"`
	MaxDurationSec     int     `json:"max_duration_seconds"`
}

type BuildMetricPoint struct {
	Date               string  `json:"date"`
	Total              int64   `json:"total"`
	Succeeded          int64   `json:"succeeded"`
	Failed             int64   `json:"failed"`
	Cancelled          int64   `json:"cancelled"`
	SuccessRate        float64 `json:"success_rate"`
	AverageDurationSec float64 `json:"average_duration_seconds"`
}

type BuildMetrics struct {
	From    time.Time          `json:"from"`
	To      time.Time          `json:"to"`
	Summary BuildMetricSummary `json:"summary"`
	Daily   []BuildMetricPoint `json:"daily"`
}

func (b *BuildRun) ParseStages() []BuildStage {
	if b.Stages == "" {
		return nil
	}
	var stages []BuildStage
	_ = json.Unmarshal([]byte(b.Stages), &stages)
	return stages
}

func (b *BuildRun) ParseGitChangedFiles() []GitFileChange {
	if b.GitChangedFiles == "" {
		return nil
	}
	var files []GitFileChange
	_ = json.Unmarshal([]byte(b.GitChangedFiles), &files)
	return files
}
