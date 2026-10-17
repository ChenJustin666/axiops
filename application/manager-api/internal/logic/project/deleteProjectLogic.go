package project

import (
	"context"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/types"
	"github.com/ChenJustin666/axiops/common/handler/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteProjectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeleteProjectLogic 软删除项目
func NewDeleteProjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProjectLogic {
	return &DeleteProjectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProjectLogic) DeleteProject(req *types.DefaultIdRequest) (resp string, err error) {
	return "", errorx.Msg("项目主数据请在门户管理中维护")
}
