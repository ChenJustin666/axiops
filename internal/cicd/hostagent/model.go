package hostagent

import "time"

const (
	StatusPending    = "pending"
	StatusBlocked    = "blocked"
	StatusClaimed    = "claimed"
	StatusRunning    = "running"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusRolledBack = "rolled_back"
	StatusCancelled  = "cancelled"
)

type Agent struct {
	ID            int64     `json:"id"`
	ProjectID     int64     `json:"project_id"`
	HostGroupRef  string    `json:"host_group_ref"`
	AssetRef      string    `json:"asset_ref"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

type RegistrationTokenRequest struct {
	HostGroupRef   string `json:"host_group_ref"`
	ExpiresMinutes int    `json:"expires_minutes"`
	MaxUses        int    `json:"max_uses"`
}

type RegistrationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RegisterRequest struct {
	Token    string                 `json:"token"`
	AssetRef string                 `json:"asset_ref"`
	Name     string                 `json:"name"`
	Version  string                 `json:"version"`
	OS       string                 `json:"os"`
	Arch     string                 `json:"arch"`
	Metadata map[string]interface{} `json:"metadata"`
}

type RegisterResponse struct {
	AgentID    int64  `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

type HeartbeatRequest struct {
	Version string `json:"version"`
}

type Command struct {
	ID             int64  `json:"id"`
	ArtifactURL    string `json:"artifact_url"`
	ArtifactSHA256 string `json:"artifact_sha256"`
	DeployScript   string `json:"deploy_script"`
	HealthScript   string `json:"health_script"`
	RollbackScript string `json:"rollback_script"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type ReportRequest struct {
	Status       string `json:"status"`
	LogText      string `json:"log_text"`
	ErrorMessage string `json:"error_message"`
}

type DeploymentCommand struct {
	WorkflowRunID     int64
	TaskKey           string
	ProjectID         int64
	ServiceID         int64
	ServiceRevisionID int64
	AgentID           int64
	ArtifactURL       string
	ArtifactSHA256    string
	DeployScript      string
	HealthScript      string
	RollbackScript    string
	TimeoutSeconds    int
}

type RolloutPolicy struct {
	BatchSize       int
	IntervalSeconds int
}

type DeploymentState struct {
	Total, Blocked, Pending, Running, Succeeded, Failed, RolledBack, Cancelled int
	CurrentBatch, TotalBatches                                                 int
	RolloutStatus, ErrorMessage                                                string
}
