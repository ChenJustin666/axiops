package config

import "github.com/zeromicro/go-zero/zrpc"

// PlatformAPIConfig platform-api 服务配置
type PlatformAPIConfig struct {
	Host string `json:",default=0.0.0.0"`
	Port int    `json:",default=8080"`

	// JWT
	JWT JWTConfig

	// CredentialEncryptionKey must be supplied through a secret manager or
	// environment-specific config. It is never persisted in the database.
	CredentialEncryptionKey string `json:",optional"`

	// MySQL
	MySQL      MySQLConfig
	Migrations MigrationConfig

	// Kubernetes is used by cicd-api for read-only server-side deployment
	// previews. platform-api may leave this section at its defaults.
	Kubernetes KubernetesWorkerConfig

	// Redis
	Redis RedisConfig

	// LegacyIdentity is disabled by default. When enabled, platform-api asks
	// the legacy portal-rpc service to validate old tokens before mapping them.
	LegacyIdentity LegacyIdentityConfig

	// Log
	Log LogConfig
}

type WorkflowWorkerConfig struct {
	Name string `json:",default=axiops-workflow-worker"`

	MySQL MySQLConfig
	// DatabaseConfigFile optionally reuses the MySQL section from a Platform
	// API config. It avoids duplicating local database credentials in two files.
	DatabaseConfigFile string `json:",optional"`

	CredentialEncryptionKey string
	Worker                  WorkerConfig
	Kubernetes              KubernetesWorkerConfig
	Build                   BuildWorkerConfig
	Notification            NotificationWorkerConfig
}

type WorkerConfig struct {
	Concurrency    int    `json:",default=4"`
	ID             string `json:",optional"`
	PollIntervalMS int    `json:",default=2000"`
	LeaseSeconds   int    `json:",default=120"`
	BatchSize      int    `json:",default=20"`
}

type KubernetesWorkerConfig struct {
	FieldManager    string  `json:",default=axiops-cicd"`
	ForceApply      bool    `json:",default=false"`
	QPS             float32 `json:",default=20"`
	Burst           int     `json:",default=40"`
	TimeoutSeconds  int     `json:",default=30"`
	ClientCacheSecs int     `json:",default=60"`
}

type BuildWorkerConfig struct {
	Defaults       BuildDefaultsConfig
	Enabled        bool   `json:",default=false"`
	ClusterID      int64  `json:",default=0"`
	Namespace      string `json:",default=axiops-cicd"`
	PollIntervalMS int    `json:",default=1000"`
	MaxAttempts    int    `json:",default=3"`
	RetryAfterSecs int    `json:",default=5"`
	StaleAfterSecs int    `json:",default=300"`
}

// Defaults are shared by services that select platform inheritance.
type BuildDefaultsConfig struct {
	BuildCommand        string `json:",optional"`
	TimeoutSeconds      int    `json:",default=1800"`
	ResourceCPU         string `json:",optional"`
	ResourceMemory      string `json:",optional"`
	BuilderImage        string `json:",optional"`
	CachePVCName        string `json:",optional"`
	CacheDir            string `json:",default=/workspace/.axiops-cache"`
	JobRetentionSeconds int    `json:",default=0"`
}

type NotificationWorkerConfig struct {
	Enabled        bool `json:",default=true"`
	PollIntervalMS int  `json:",default=2000"`
	MaxAttempts    int  `json:",default=5"`
	LeaseSeconds   int  `json:",default=60"`
}

type LegacyIdentityConfig struct {
	Enabled   bool   `json:",default=false"`
	Provider  string `json:",default=legacy_autoops"`
	TimeoutMS int    `json:",default=1500"`
	PortalRPC zrpc.RpcClientConf
}

type JWTConfig struct {
	Secret               string `json:",default=axiops-jwt-secret-change-me"`
	AccessExpireSeconds  int64  `json:",default=7200"`   // 2 小时
	RefreshExpireSeconds int64  `json:",default=604800"` // 7 天
}

type MySQLConfig struct {
	Host     string `json:",default=127.0.0.1"`
	Port     int    `json:",default=3306"`
	User     string `json:",default=root"`
	Password string `json:",default="`
	Database string `json:",default=axiops"`
	Charset  string `json:",default=utf8mb4"`
	MaxIdle  int    `json:",default=10"`
	MaxOpen  int    `json:",default=100"`
}

type MigrationConfig struct {
	AutoApply          bool `json:",default=true"`
	LockTimeoutSeconds int  `json:",default=30"`
}

type RedisConfig struct {
	Host     string `json:",default=127.0.0.1:6379"`
	Password string `json:",default="`
	DB       int    `json:",default=0"`
}

type LogConfig struct {
	Level string `json:",default=info"` // debug / info / warn / error
}

// DSN 生成 MySQL DSN
func (c *MySQLConfig) DSN() string {
	// MySQL timestamps are written with CURRENT_TIMESTAMP by the database and
	// with explicit time.Time values by the services. Use UTC at the driver
	// boundary AND on every database session. loc only controls how the driver
	// decodes timestamps; it does not change MySQL CURRENT_TIMESTAMP or the
	// session's TIMESTAMP conversion. Leaving the session at +08:00 made leases
	// appear eight hours in the future and prevented crashed tasks recovering.
	return c.User + ":" + c.Password + "@tcp(" + c.Host + ":" + itoa(c.Port) + ")/" + c.Database + "?charset=" + c.Charset + "&parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
