package sysauthservicelogic

import (
	"context"
	"strings"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/authz"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/portal-rpc/pb"
	"github.com/ChenJustin666/axiops/common/handler/errorx"
	"github.com/zeromicro/go-zero/core/logx"
)

type ApiAuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApiAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApiAuthLogic {
	return &ApiAuthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ApiAuthLogic) ApiAuth(in *pb.ApiAuthRequest) (*pb.ApiAuthResponse, error) {
	if l.svcCtx.Config.DemoMode && !allowDemoModeRequest(in.ApiPath, in.ApiMethod) {
		return nil, errorx.Msg("演示模式下仅允许 GET 请求!")
	}

	allowed, err := l.svcCtx.AuthzManager.Authorize(l.ctx, authz.NewAPIAuthorizeRequest(in.UserRoles, in.ApiPath, in.ApiMethod))
	if err != nil {
		l.Errorf("[RBAC] 检查权限失败: %v", err)
		return nil, errorx.Msg("检查权限失败")
	}
	if !allowed {
		l.Errorf("[RBAC] 用户无权限访问 %s %s", in.ApiMethod, in.ApiPath)
		return nil, errorx.Msg("无权限访问")
	}
	return &pb.ApiAuthResponse{
		IsAuth: true,
	}, nil
}

func allowDemoModeRequest(path, method string) bool {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	normalizedPath := strings.TrimSpace(path)
	return normalizedMethod == "GET" ||
		(normalizedPath == "/portal/v1/auth/logout" && normalizedMethod == "POST")
}
