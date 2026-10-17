package logging

import (
	"context"
	"github.com/ChenJustin666/axiops/common/observabilityaccess"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/types"
	logservice "github.com/ChenJustin666/axiops/application/manager-rpc/client/logservice"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetLogQueryModeOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLogQueryModeOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLogQueryModeOptionsLogic {
	return &GetLogQueryModeOptionsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetLogQueryModeOptionsLogic) GetLogQueryModeOptions(req *types.LogQueryModeOptionsRequest) (*types.LogQueryModeOptionsResponse, error) {
	rpcResp, err := l.svcCtx.LogRpc.GetLogQueryModeOptions(l.ctx, &logservice.LogQueryModeOptionsReq{ClusterUuid: req.ClusterUuid})
	if err != nil {
		return nil, err
	}

	return &types.LogQueryModeOptionsResponse{
		Loki:          rpcResp.Loki,
		Elasticsearch: rpcResp.Elasticsearch,
		RawQuery:      observabilityaccess.Can(l.ctx, "view", true),
		Export:        observabilityaccess.Can(l.ctx, "export", false),
	}, nil
}
