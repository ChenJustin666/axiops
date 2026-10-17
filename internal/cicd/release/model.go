package release

import "time"

// ReleaseRecord 发布记录模型
type ReleaseRecord struct {
	ID             int64      `json:"id"`
	ServiceID      int64      `json:"service_id"`
	ProjectID      int64      `json:"project_id"`
	EnvironmentID  int64      `json:"environment_id"`
	ClusterID      int64      `json:"cluster_id"`
	Namespace      string     `json:"namespace"`
	ReleaseNumber  int        `json:"release_number"`
	ImageTag       string     `json:"image_tag"`
	ImageFull      string     `json:"image_full"`
	Strategy       string     `json:"strategy"`
	Status         string     `json:"status"`
	ApprovalStatus string     `json:"approval_status"`
	ApprovalUserID int64      `json:"approval_user_id"`
	DeployUserID   int64      `json:"deploy_user_id"`
	RollbackFrom   int64      `json:"rollback_from"`
	DurationSec    int        `json:"duration_sec"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CreateReleaseRequest 创建发布
type CreateReleaseRequest struct {
	ServiceID     int64  `json:"service_id" validate:"required"`
	EnvironmentID int64  `json:"environment_id" validate:"required"`
	ImageTag      string `json:"image_tag" validate:"required"`
	ImageFull     string `json:"image_full"`
	Strategy      string `json:"strategy"` // rolling/bluegreen/canary
}

// RollbackRequest 回滚请求
type RollbackRequest struct {
	ReleaseID int64 `json:"release_id" validate:"required"`
}
