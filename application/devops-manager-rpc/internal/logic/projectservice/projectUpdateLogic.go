package projectservicelogic

import (
	"context"

	"github.com/ChenJustin666/axiops/application/devops-manager-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/devops-manager-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProjectUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewProjectUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProjectUpdateLogic {
	return &ProjectUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ProjectUpdateLogic) ProjectUpdate(in *pb.UpdateProjectReq) (*pb.EmptyResp, error) {
	return nil, errorx.Msg("项目主数据请在门户项目管理中维护")
}
