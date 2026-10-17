package repositoryservicelogic

import (
	"context"

	"github.com/ChenJustin666/axiops/application/console-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/console-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRepositoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteRepositoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRepositoryLogic {
	return &DeleteRepositoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteRepositoryLogic) DeleteRepository(in *pb.DeleteRepositoryReq) (*pb.DeleteRepositoryResp, error) {
	client, err := l.svcCtx.HarborManager.Get(in.RegistryUuid)
	if err != nil {
		return nil, errorx.Msg("获取仓库客户端失败")
	}

	err = client.Repository().Delete(in.ProjectName, in.RepoName)
	if err != nil {
		return nil, errorx.Msg("删除仓库失败")
	}

	return &pb.DeleteRepositoryResp{Message: "删除成功"}, nil
}
