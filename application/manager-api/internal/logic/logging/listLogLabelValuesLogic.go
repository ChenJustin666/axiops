package logging

import (
	"context"
	"github.com/ChenJustin666/axiops/common/observabilityaccess"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/types"
	logservice "github.com/ChenJustin666/axiops/application/manager-rpc/client/logservice"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogLabelValuesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogLabelValuesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogLabelValuesLogic {
	return &ListLogLabelValuesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListLogLabelValuesLogic) ListLogLabelValues(req *types.LogLabelValuesRequest) (*types.LogLabelValuesResponse, error) {
	rpcResp, err := l.svcCtx.LogRpc.ListLogLabelValues(l.ctx, &logservice.ListLogLabelValuesReq{
		SourceType:   req.SourceType,
		ClusterUuid:  req.ClusterUuid,
		QueryMode:    req.QueryMode,
		LabelKey:     req.LabelKey,
		WorkspaceId:  req.WorkspaceId,
		Namespace:    req.Namespace,
		Application:  req.Application,
		ResourceName: req.ResourceName,
		Hosts:        req.Hosts,
	})
	if err != nil {
		return nil, err
	}
	if err := observabilityaccess.Recheck(l.ctx); err != nil {
		return nil, err
	}
	return &types.LogLabelValuesResponse{Values: rpcResp.Values}, nil
}
