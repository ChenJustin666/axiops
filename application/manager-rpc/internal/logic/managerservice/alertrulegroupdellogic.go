package managerservicelogic

import (
	"context"

	"github.com/ChenJustin666/axiops/application/manager-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AlertRuleGroupDelLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAlertRuleGroupDelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AlertRuleGroupDelLogic {
	return &AlertRuleGroupDelLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AlertRuleGroupDelLogic) AlertRuleGroupDel(in *pb.DelAlertRuleGroupReq) (*pb.DelAlertRuleGroupResp, error) {
	// todo: add your logic here and delete this line

	return &pb.DelAlertRuleGroupResp{}, nil
}
