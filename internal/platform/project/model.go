package project

import "time"

const (
	DeliveryTypeUnclassified = "unclassified"
	DeliveryTypeK8sYAML      = "k8s_yaml"
	DeliveryTypeHelm         = "helm"
	DeliveryTypeHost         = "host"

	DeliveryStatusPendingConfirmation = "pending_confirmation"
	DeliveryStatusActive              = "active"

	InitializationStatusPending    = "pending"
	InitializationStatusInProgress = "in_progress"
	InitializationStatusCompleted  = "completed"
)

// IsValidDeliveryType reports whether a value can be selected for a new project.
func IsValidDeliveryType(deliveryType string) bool {
	switch deliveryType {
	case DeliveryTypeK8sYAML, DeliveryTypeHelm, DeliveryTypeHost:
		return true
	default:
		return false
	}
}

// CanConfirmDeliveryType restricts legacy projects to a single, explicit conversion.
func CanConfirmDeliveryType(currentType, currentStatus, nextType string) bool {
	return currentType == DeliveryTypeUnclassified &&
		currentStatus == DeliveryStatusPendingConfirmation &&
		IsValidDeliveryType(nextType)
}

// Project 项目模型
type Project struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	Code                 string     `json:"code"`
	Description          string     `json:"description"`
	Status               string     `json:"status"`
	DeliveryType         string     `json:"delivery_type"`
	DeliveryStatus       string     `json:"delivery_status"`
	OwnerUserID          int64      `json:"owner_user_id"`
	CreatedBy            int64      `json:"created_by"`
	InitializationStatus string     `json:"initialization_status"`
	InitializationStep   int        `json:"initialization_step"`
	InitializedAt        *time.Time `json:"initialized_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ProjectMember 项目成员
type ProjectMember struct {
	ID        int64     `json:"id"`
	ProjectID int64     `json:"project_id"`
	UserID    int64     `json:"user_id"`
	RoleID    int64     `json:"role_id"`
	JoinedAt  time.Time `json:"joined_at"`
	CreatedBy int64     `json:"created_by"`
}

// ProjectWithMembers 带成员的项目
type ProjectWithMembers struct {
	Project
	Members []ProjectMember `json:"members"`
}

// CreateProjectRequest 创建项目
type CreateProjectRequest struct {
	Name         string `json:"name" validate:"required,max=64"`
	Code         string `json:"code" validate:"required,max=64"`
	Description  string `json:"description"`
	OwnerUserID  int64  `json:"owner_user_id"`
	DeliveryType string `json:"delivery_type" validate:"required,oneof=k8s_yaml helm host"`
}

// UpdateProjectRequest 更新项目
type UpdateProjectRequest struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	Status               string  `json:"status"`
	OwnerUserID          *int64  `json:"owner_user_id"`
	InitializationStatus *string `json:"initialization_status,omitempty"`
	InitializationStep   *int    `json:"initialization_step,omitempty"`
}

// DeliveryProfile is the immutable delivery-form selection for a project.
type DeliveryProfile struct {
	ProjectID      int64  `json:"project_id"`
	DeliveryType   string `json:"delivery_type"`
	DeliveryStatus string `json:"delivery_status"`
}

// ConfirmDeliveryProfileRequest confirms the type of a legacy project once.
type ConfirmDeliveryProfileRequest struct {
	DeliveryType string `json:"delivery_type" validate:"required,oneof=k8s_yaml helm host"`
}

// AddMemberRequest 添加项目成员
type AddMemberRequest struct {
	UserID int64 `json:"user_id" validate:"required"`
	RoleID int64 `json:"role_id" validate:"required"`
}

// ListProjectsRequest 项目列表查询
type ListProjectsRequest struct {
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}
