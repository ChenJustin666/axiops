package svc

import (
	"github.com/ChenJustin666/axiops/application/console-api/internal/common/uploadcore"
	"github.com/ChenJustin666/axiops/application/console-api/internal/config"
	"github.com/ChenJustin666/axiops/application/console-rpc/client/repositoryservice"
	"github.com/ChenJustin666/axiops/application/manager-rpc/client/managerservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
	"github.com/ChenJustin666/axiops/common/interceptors"
	"github.com/ChenJustin666/axiops/common/k8smanager/cluster"
	"github.com/ChenJustin666/axiops/common/middleware"
	"github.com/ChenJustin666/axiops/common/observabilityaccess"
	cluster2 "github.com/ChenJustin666/axiops/common/prometheusmanager/cluster"
	"github.com/ChenJustin666/axiops/common/verify"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config            config.Config
	Cache             *redis.Redis
	Validator         *verify.ValidatorInstance
	JWTAuthMiddleware rest.Middleware
	ManagerRpc        managerservice.ManagerService
	K8sManager        cluster.Manager
	UploadCoreClient  *uploadcore.Service
	RepositoryRpc     repositoryservice.RepositoryService
	PrometheusManager *cluster2.PrometheusManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	validator, err := verify.InitValidator(verify.LocaleZH)
	if err != nil {
		panic(err)
	}

	cacheClient := redis.MustNewRedis(c.Cache)

	managerRpc := zrpc.MustNewClient(c.ManagerRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	managerService := managerservice.NewManagerService(managerRpc)

	manager := cluster.NewManager(managerService, cacheClient)

	uploadServer := uploadcore.InitService(c.LocalCacheDir)
	repositoryRpc := zrpc.MustNewClient(c.ConsoleRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	authRpc := zrpc.MustNewClient(c.PortalRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)

	return &ServiceContext{
		Config:    c,
		Cache:     cacheClient,
		Validator: validator,
		JWTAuthMiddleware: middleware.NewJWTAuthMiddleware(
			sysauthservice.NewSysAuthService(authRpc)).WithResourceAccess(observabilityaccess.NewSQL(c.ResourceAccess, observabilityaccess.Workspaces(managerService))).Handle,
		ManagerRpc:        managerService,
		K8sManager:        manager,
		UploadCoreClient:  uploadServer,
		RepositoryRpc:     repositoryservice.NewRepositoryService(repositoryRpc),
		PrometheusManager: cluster2.NewPrometheusManager(managerService, cacheClient),
	}
}
