package webhook

import (
	"context"

	"github.com/ChenJustin666/axiops/application/manager-api/internal/svc"
	"github.com/ChenJustin666/axiops/application/manager-api/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveLogAlertWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReceiveLogAlertWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveLogAlertWebhookLogic {
	return &ReceiveLogAlertWebhookLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ReceiveLogAlertWebhookLogic) ReceiveLogAlertWebhook(req *types.AlertmanagerWebhook) (resp string, err error) {
	return NewReceiveAlertmanagerWebhookLogic(l.ctx, l.svcCtx).ReceiveAlertmanagerWebhook(req)
}
