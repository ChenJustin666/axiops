package billing

import (
	"context"
	"fmt"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/types"
	"github.com/ChenJustin666/axiops/application/manager-rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnecBillingPriceConfigDelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOnecBillingPriceConfigDelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnecBillingPriceConfigDelLogic {
	return &OnecBillingPriceConfigDelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnecBillingPriceConfigDelLogic) OnecBillingPriceConfigDel(req *types.OnecBillingPriceConfigDelRequest) (resp *types.OnecBillingPriceConfigDelResponse, err error) {
	// 调用RPC服务删除收费配置
	_, err = l.svcCtx.ManagerRpc.OnecBillingPriceConfigDel(l.ctx, &pb.OnecBillingPriceConfigDelReq{
		Id: req.Id,
	})
	if err != nil {
		l.Errorf("删除收费配置失败: %v", err)
		return nil, fmt.Errorf("删除收费配置失败: %v", err)
	}

	return &types.OnecBillingPriceConfigDelResponse{}, nil
}
