package config

import (
	"github.com/ChenJustin666/axiops/common/observabilityaccess"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Webhook struct {
	Token string `json:"Token"`
}
type Config struct {
	rest.RestConf
	ResourceAccess observabilityaccess.Config `json:",optional"`
	Cache          redis.RedisConf
	ManagerRpc     zrpc.RpcClientConf
	PortalRpc      zrpc.RpcClientConf
	Webhook        Webhook `json:"Webhook"`
}
