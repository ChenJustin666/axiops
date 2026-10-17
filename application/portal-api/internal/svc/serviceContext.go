package svc

import (
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/portalprojectservice"
	"github.com/ChenJustin666/axiops/application/portal-api/internal/config"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/alertservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/portalservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sitemessagesservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/storageservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
	"github.com/ChenJustin666/axiops/common/interceptors"
	"github.com/ChenJustin666/axiops/common/middleware"
	"github.com/ChenJustin666/axiops/common/verify"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config            config.Config
	Cache             *redis.Redis
	PortalRpc         portalservice.PortalService
	SysAuthRpc        sysauthservice.SysAuthService
	StoreRpc          storageservice.StorageService
	Validator         *verify.ValidatorInstance
	AlertPortalRpc    alertservice.AlertService
	JWTAuthMiddleware rest.Middleware
	SiteMessagesRpc   sitemessagesservice.SiteMessagesService
	ProjectRpc        portalprojectservice.PortalProjectService
	SiteMessageHub    *SiteMessageHub
}

func NewServiceContext(c config.Config) *ServiceContext {
	validator, err := verify.InitValidator(verify.LocaleZH)
	if err != nil {
		panic(err)
	}

	rdb := redis.MustNewRedis(c.Cache)

	// 自定义拦截器
	protalRpc := zrpc.MustNewClient(c.PortalRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	alertRpc := zrpc.MustNewClient(c.PortalRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	authRpc := zrpc.MustNewClient(c.PortalRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	sysAuthRpc := sysauthservice.NewSysAuthService(authRpc)

	// 初始化并启动 SiteMessageHub
	messageHub := NewSiteMessageHub(rdb)
	messageHub.Start()

	logx.Info("站内消息 WebSocket Hub 已启动")

	return &ServiceContext{
		Config:         c,
		Cache:          rdb,
		Validator:      validator,
		PortalRpc:      portalservice.NewPortalService(protalRpc),
		AlertPortalRpc: alertservice.NewAlertService(alertRpc),
		SysAuthRpc:     sysAuthRpc,
		StoreRpc:       storageservice.NewStorageService(protalRpc),
		JWTAuthMiddleware: middleware.NewJWTAuthMiddleware(
			sysAuthRpc,
		).Handle,
		ProjectRpc:        portalprojectservice.NewPortalProjectService(protalRpc),
		SiteMessagesRpc: sitemessagesservice.NewSiteMessagesService(protalRpc),
		SiteMessageHub:  messageHub,
	}
}
