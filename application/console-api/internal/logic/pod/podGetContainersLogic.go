package pod

import (
	"context"

	"github.com/ChenJustin666/axiops/application/console-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/console-api/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type PodGetContainersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取 Pod 所有容器
func NewPodGetContainersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PodGetContainersLogic {
	return &PodGetContainersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PodGetContainersLogic) PodGetContainers(req *types.PodGetContainersReq) (resp *types.PodGetContainersResp, err error) {
	// todo: add your logic here and delete this line

	return
}
