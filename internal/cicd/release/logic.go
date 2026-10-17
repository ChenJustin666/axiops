package release

import (
	"context"
	"database/sql"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type Logic struct {
	repo    *Repository
	svcRepo *appsvc.Repository
}

func NewLogic(db *sql.DB) *Logic {
	return &Logic{
		repo:    NewRepository(db),
		svcRepo: appsvc.NewRepository(db),
	}
}

func (l *Logic) ListByService(ctx context.Context, serviceID int64, page, pageSize int) ([]ReleaseRecord, int64, int) {
	records, total, err := l.repo.ListByService(ctx, serviceID, page, pageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	return records, total, errcode.CodeSuccess
}

func (l *Logic) GetByID(ctx context.Context, id int64) (*ReleaseRecord, int) {
	rec, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if rec == nil {
		return nil, errcode.CodeReleaseNotFound
	}
	return rec, errcode.CodeSuccess
}

func (l *Logic) Create(ctx context.Context, req *CreateReleaseRequest, deployUserID int64) (int64, int) {
	svc, err := l.svcRepo.FindByID(ctx, req.ServiceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if svc == nil {
		return 0, errcode.CodeServiceNotFound
	}

	releaseNum, err := l.repo.GetNextReleaseNumber(ctx, req.ServiceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	strategy := req.Strategy
	if strategy == "" {
		strategy = "rolling"
	}

	imageFull := req.ImageFull
	if imageFull == "" && svc.ImageName != "" {
		imageFull = svc.ImageName + ":" + req.ImageTag
	}

	rec := &ReleaseRecord{
		ServiceID:      req.ServiceID,
		ProjectID:      svc.ProjectID,
		EnvironmentID:  req.EnvironmentID,
		ReleaseNumber:  releaseNum,
		ImageTag:       req.ImageTag,
		ImageFull:      imageFull,
		Strategy:       strategy,
		Status:         "pending",
		ApprovalStatus: "none", // TODO: 根据环境配置决定是否需要审批
		DeployUserID:   deployUserID,
	}

	id, err := l.repo.Create(ctx, rec)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	// TODO: 如果不需要审批，直接触发部署
	// go l.executeDeploy(id)

	return id, errcode.CodeSuccess
}

// Approve 审批发布
func (l *Logic) Approve(ctx context.Context, id int64, approved bool, approverID int64) int {
	rec, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if rec == nil {
		return errcode.CodeReleaseNotFound
	}
	if rec.ApprovalStatus != "pending" {
		return errcode.CodeParamInvalid
	}

	status := "approved"
	if !approved {
		status = "rejected"
	}

	if err := l.repo.UpdateApproval(ctx, id, status, approverID); err != nil {
		return errcode.CodeDBError
	}

	if approved {
		// 审批通过，更新发布状态为 deploying
		_ = l.repo.UpdateStatus(ctx, id, "deploying", 0)
		// TODO: 触发实际部署
	} else {
		_ = l.repo.UpdateStatus(ctx, id, "failed", 0)
	}

	return errcode.CodeSuccess
}

// Rollback 回滚
func (l *Logic) Rollback(ctx context.Context, req *RollbackRequest, deployUserID int64) (int64, int) {
	// 获取要回滚到的发布记录
	target, err := l.repo.FindByID(ctx, req.ReleaseID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if target == nil {
		return 0, errcode.CodeReleaseNotFound
	}

	// 创建新的发布记录（标记为回滚）
	createReq := &CreateReleaseRequest{
		ServiceID:     target.ServiceID,
		EnvironmentID: target.EnvironmentID,
		ImageTag:      target.ImageTag,
		ImageFull:     target.ImageFull,
		Strategy:      target.Strategy,
	}

	releaseNum, err := l.repo.GetNextReleaseNumber(ctx, target.ServiceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	rec := &ReleaseRecord{
		ServiceID:      createReq.ServiceID,
		ProjectID:      target.ProjectID,
		EnvironmentID:  createReq.EnvironmentID,
		ReleaseNumber:  releaseNum,
		ImageTag:       createReq.ImageTag,
		ImageFull:      createReq.ImageFull,
		Strategy:       createReq.Strategy,
		Status:         "pending",
		ApprovalStatus: "none",
		DeployUserID:   deployUserID,
		RollbackFrom:   req.ReleaseID,
	}

	id, err := l.repo.Create(ctx, rec)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	return id, errcode.CodeSuccess
}
