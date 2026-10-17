package project

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

// Logic 项目业务逻辑
type Logic struct {
	repo       *Repository
	authorizer authz.Checker
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer}
}

// List 项目列表
func (l *Logic) List(ctx context.Context, userID int64, keyword, status string, page, pageSize int) ([]Project, int64, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "project", "view")
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	projects, total, err := l.repo.List(ctx, scope, keyword, status, page, pageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	return projects, total, errcode.CodeSuccess
}

// ListByUser 用户参与的项目
func (l *Logic) ListByUser(ctx context.Context, userID int64) ([]Project, int) {
	projects, _, code := l.List(ctx, userID, "", "active", 1, 10000)
	return projects, code
}

// GetByID 项目详情
func (l *Logic) GetByID(ctx context.Context, userID, id int64) (*ProjectWithMembers, int) {
	p, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if p == nil {
		return nil, errcode.CodeProjectNotFound
	}
	if code := l.authorize(ctx, userID, "project", "view", id); code != errcode.CodeSuccess {
		return nil, code
	}

	members, err := l.repo.GetMembers(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}

	return &ProjectWithMembers{
		Project: *p,
		Members: members,
	}, errcode.CodeSuccess
}

// Create 创建项目
func (l *Logic) Create(ctx context.Context, req *CreateProjectRequest, createdBy int64) (int64, int) {
	if req == nil || createdBy <= 0 || !validProjectIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if code := l.authorize(ctx, createdBy, "project", "create", 0); code != errcode.CodeSuccess {
		return 0, code
	}
	if !IsValidDeliveryType(req.DeliveryType) {
		return 0, errcode.CodeParamInvalid
	}
	// 检查 code 唯一
	existing, err := l.repo.FindByCode(ctx, req.Code)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if existing != nil {
		return 0, errcode.CodeProjectNameExists
	}

	p := &Project{
		Name:                 req.Name,
		Code:                 req.Code,
		Description:          req.Description,
		Status:               "active",
		DeliveryType:         req.DeliveryType,
		DeliveryStatus:       DeliveryStatusActive,
		OwnerUserID:          req.OwnerUserID,
		CreatedBy:            createdBy,
		InitializationStatus: InitializationStatusPending,
		InitializationStep:   0,
	}
	if p.OwnerUserID == 0 {
		p.OwnerUserID = createdBy
	}
	// Role IDs are installation-specific; resolve the built-in role by code.
	projectAdminRoleID, roleErr := l.repo.RoleIDByCode(ctx, "project_admin")
	if roleErr != nil {
		return 0, errcode.CodeDBError
	}

	id, err := l.repo.CreateWithOwnerRole(ctx, p, projectAdminRoleID)
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			return 0, errcode.CodeProjectNameExists
		}
		return 0, errcode.CodeDBError
	}

	return id, errcode.CodeSuccess
}

// GetDeliveryProfile returns the project delivery type after project-view authorization.
func (l *Logic) GetDeliveryProfile(ctx context.Context, userID, id int64) (*DeliveryProfile, int) {
	p, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if p == nil {
		return nil, errcode.CodeProjectNotFound
	}
	if code := l.authorize(ctx, userID, "project", "view", id); code != errcode.CodeSuccess {
		return nil, code
	}
	return &DeliveryProfile{ProjectID: p.ID, DeliveryType: p.DeliveryType, DeliveryStatus: p.DeliveryStatus}, errcode.CodeSuccess
}

// ConfirmDeliveryProfile confirms a legacy project's delivery type once.
func (l *Logic) ConfirmDeliveryProfile(ctx context.Context, userID, id int64, req *ConfirmDeliveryProfileRequest) int {
	if req == nil || userID <= 0 || id <= 0 {
		return errcode.CodeParamInvalid
	}
	if !IsValidDeliveryType(req.DeliveryType) {
		return errcode.CodeParamInvalid
	}
	p, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if p == nil {
		return errcode.CodeProjectNotFound
	}
	if code := l.authorize(ctx, userID, "project", "update", id); code != errcode.CodeSuccess {
		return code
	}
	if !CanConfirmDeliveryType(p.DeliveryType, p.DeliveryStatus, req.DeliveryType) {
		return errcode.CodeParamInvalid
	}
	confirmed, err := l.repo.ConfirmDeliveryProfile(ctx, id, req.DeliveryType)
	if err != nil {
		return errcode.CodeDBError
	}
	if !confirmed {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeSuccess
}

// Update 更新项目
func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateProjectRequest) int {
	if req == nil || userID <= 0 || id <= 0 {
		return errcode.CodeParamInvalid
	}
	p, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if p == nil {
		return errcode.CodeProjectNotFound
	}
	if code := l.authorize(ctx, userID, "project", "update", id); code != errcode.CodeSuccess {
		return code
	}

	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Status != "" {
		if req.Status != "active" && req.Status != "disabled" {
			return errcode.CodeParamInvalid
		}
		p.Status = req.Status
	}
	if req.OwnerUserID != nil {
		p.OwnerUserID = *req.OwnerUserID
	}
	if !applyInitializationUpdate(p, req, time.Now()) {
		return errcode.CodeParamInvalid
	}

	if err := l.repo.Update(ctx, p); err != nil {
		if errors.Is(err, projectstate.ErrDeleted) {
			return errcode.CodeProjectDeleted
		}
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// Initialization is explicitly completed by the final confirmation. Older tabs
// may save progress, but must never reopen a completed project.
func applyInitializationUpdate(p *Project, req *UpdateProjectRequest, now time.Time) bool {
	if p.InitializationStatus == InitializationStatusCompleted &&
		((req.InitializationStatus != nil && *req.InitializationStatus != InitializationStatusCompleted) ||
			(req.InitializationStep != nil && *req.InitializationStep != 4)) {
		return false
	}
	if req.InitializationStatus != nil {
		if !isValidInitializationStatus(*req.InitializationStatus) {
			return false
		}
		p.InitializationStatus = *req.InitializationStatus
	}
	if req.InitializationStep != nil {
		if *req.InitializationStep < 0 || *req.InitializationStep > 4 {
			return false
		}
		p.InitializationStep = *req.InitializationStep
	}
	if p.InitializationStatus == InitializationStatusCompleted {
		if p.InitializationStep != 4 {
			return false
		}
		if p.InitializedAt == nil {
			p.InitializedAt = &now
		}
	} else if p.InitializationStatus != InitializationStatusCompleted {
		p.InitializedAt = nil
	}

	return true
}

func isValidInitializationStatus(status string) bool {
	switch status {
	case InitializationStatusPending, InitializationStatusInProgress, InitializationStatusCompleted:
		return true
	default:
		return false
	}
}

var projectCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func validProjectIdentity(name, code string) bool {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	return name != "" && len(name) <= 128 && projectCodePattern.MatchString(code)
}

// Delete 删除项目
func (l *Logic) Delete(ctx context.Context, userID, id int64, req *DeleteProjectRequest) (*DeletionCheck, int) {
	if req == nil || req.ConfirmName == "" {
		return nil, errcode.CodeParamInvalid
	}
	return l.deleteChecked(ctx, userID, id, req)
}

// AddMember 添加成员
func (l *Logic) AddMember(ctx context.Context, projectID int64, req *AddMemberRequest, operatorID int64) int {
	if code := l.authorize(ctx, operatorID, "project_member", "create", projectID); code != errcode.CodeSuccess {
		return code
	}
	validRole, err := l.repo.IsScopedRole(ctx, req.RoleID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !validRole {
		return errcode.CodeParamInvalid
	}
	if err := l.repo.AddMember(ctx, &ProjectMember{
		ProjectID: projectID,
		UserID:    req.UserID,
		RoleID:    req.RoleID,
		CreatedBy: operatorID,
	}); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

// RemoveMember 移除成员
func (l *Logic) RemoveMember(ctx context.Context, projectID, userID, operatorID int64) int {
	if code := l.authorize(ctx, operatorID, "project_member", "delete", projectID); code != errcode.CodeSuccess {
		return code
	}
	if err := l.repo.RemoveMember(ctx, projectID, userID); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func (l *Logic) authorize(ctx context.Context, userID int64, resource, action string, projectID int64) int {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: resource, Action: action, ProjectID: projectID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}
