package svc

import (
	"github.com/ChenJustin666/axiops/application/manager-rpc/client/managerservice"
	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
	"github.com/ChenJustin666/axiops/application/workload-api/internal/config"
	"github.com/ChenJustin666/axiops/common/interceptors"
	"github.com/ChenJustin666/axiops/common/k8smanager/cluster"
	"github.com/ChenJustin666/axiops/common/middleware"
	"github.com/ChenJustin666/axiops/common/observabilityaccess"
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
}

func NewServiceContext(c config.Config) *ServiceContext {
	validator, err := verify.InitValidator(verify.LocaleZH)
	if err != nil {
		panic(err)
	}

	managerRpc := zrpc.MustNewClient(c.ManagerRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)

	manager := cluster.NewManager(managerservice.NewManagerService(managerRpc), redis.MustNewRedis(c.Cache))
	authRpc := zrpc.MustNewClient(c.PortalRpc,
		zrpc.WithUnaryClientInterceptor(interceptors.ClientMetadataInterceptor()),
		zrpc.WithUnaryClientInterceptor(interceptors.ClientErrorInterceptor()),
	)
	return &ServiceContext{
		Config: c,
		//Cache:             redis.MustNewRedis(c.Cache),
		Validator: validator,
		JWTAuthMiddleware: middleware.NewJWTAuthMiddleware(
			sysauthservice.NewSysAuthService(authRpc)).WithResourceAccess(observabilityaccess.NewSQL(c.ResourceAccess, observabilityaccess.Workspaces(managerservice.NewManagerService(managerRpc)))).Handle,
		ManagerRpc: managerservice.NewManagerService(managerRpc),
		K8sManager: manager,
	}
}
