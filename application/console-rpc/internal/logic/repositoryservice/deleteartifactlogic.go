package repositoryservicelogic

import (
	"context"

	"github.com/ChenJustin666/axiops/application/console-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/console-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteArtifactLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteArtifactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteArtifactLogic {
	return &DeleteArtifactLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteArtifactLogic) DeleteArtifact(in *pb.DeleteArtifactReq) (*pb.DeleteArtifactResp, error) {
	client, err := l.svcCtx.HarborManager.Get(in.RegistryUuid)
	if err != nil {
		return nil, errorx.Msg("获取仓库客户端失败")
	}

	err = client.Artifact().Delete(in.ProjectName, in.RepoName, in.Reference)
	if err != nil {
		return nil, errorx.Msg("删除制品失败")
	}

	return &pb.DeleteArtifactResp{Message: "删除成功"}, nil
}
