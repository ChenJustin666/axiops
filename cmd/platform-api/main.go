package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
	"github.com/ChenJustin666/axiops/internal/platform/access"
	"github.com/ChenJustin666/axiops/internal/platform/auth"
	"github.com/ChenJustin666/axiops/internal/platform/cluster"
	"github.com/ChenJustin666/axiops/internal/platform/credential"
	"github.com/ChenJustin666/axiops/internal/platform/department"
	"github.com/ChenJustin666/axiops/internal/platform/identity"
	"github.com/ChenJustin666/axiops/internal/platform/project"
	"github.com/ChenJustin666/axiops/internal/platform/role"
	"github.com/ChenJustin666/axiops/internal/platform/user"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	"github.com/ChenJustin666/axiops/migrations"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

func main() {
	configFile := flag.String("f", "configs/platform-api.yaml", "config file path")
	listenPort := flag.Int("port", 0, "override HTTP port (0 uses config)")
	listenHost := flag.String("host", "", "override HTTP listen address (empty uses config)")
	flag.Parse()
	if *listenPort < 0 || *listenPort > 65535 {
		log.Fatal("port must be between 1 and 65535, or 0 to use config")
	}

	var c config.PlatformAPIConfig
	conf.MustLoad(*configFile, &c)
	if *listenPort != 0 {
		c.Port = *listenPort
	}
	if *listenHost != "" {
		c.Host = *listenHost
	}
	c.CredentialEncryptionKey = config.CredentialEncryptionKey(c.CredentialEncryptionKey)
	if c.CredentialEncryptionKey == "" {
		log.Fatal("CredentialEncryptionKey must be configured; refusing to start with plaintext credential storage")
	}

	mysqlDB, err := db.NewMySQL(&c.MySQL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer mysqlDB.Close()
	migrationCtx, cancelMigrations := context.WithTimeout(context.Background(), 5*time.Minute)
	migrationResult, err := migrations.Ensure(migrationCtx, mysqlDB, c.Migrations.AutoApply, c.Migrations.LockTimeoutSeconds)
	cancelMigrations()
	if err != nil {
		log.Fatalf("database migration check failed: %v", err)
	}
	log.Printf("database migrations ready: current=%s applied=%v", migrationResult.Current, migrationResult.Applied)

	// Logic 层
	authLogic := auth.NewLogic(mysqlDB, c.JWT)
	authorizer := authz.NewSQLAuthorizer(mysqlDB)
	userLogic := user.NewLogic(mysqlDB, authorizer)
	roleLogic := role.NewLogic(mysqlDB, authorizer)
	projectLogic := project.NewLogic(mysqlDB, authorizer)
	clusterLogic := cluster.NewLogic(mysqlDB, authorizer, c.CredentialEncryptionKey)
	credentialLogic := credential.NewLogic(mysqlDB, authorizer, c.CredentialEncryptionKey)
	departmentLogic := department.NewLogic(mysqlDB, authorizer)
	accessLogic := access.NewLogic(mysqlDB, authorizer)

	// Handler 层
	authHandler := auth.NewHandler(authLogic)
	userHandler := user.NewHandler(userLogic)
	roleHandler := role.NewHandler(roleLogic)
	projectHandler := project.NewHandler(projectLogic)
	clusterHandler := cluster.NewHandler(clusterLogic)
	credentialHandler := credential.NewHandler(credentialLogic)
	departmentHandler := department.NewHandler(departmentLogic)
	accessHandler := access.NewHandler(accessLogic)

	// HTTP 服务
	server := rest.MustNewServer(rest.RestConf{
		Host: c.Host,
		Port: c.Port,
	})
	defer server.Stop()
	server.Use(middleware.SyncPathValues)

	authMw := middleware.NewAuthMiddleware(c.JWT.Secret).WithAudit(middleware.NewAuditMiddleware(mysqlDB))
	if c.LegacyIdentity.Enabled {
		portalClient := sysauthservice.NewSysAuthService(zrpc.MustNewClient(c.LegacyIdentity.PortalRPC))
		legacyResolver := identity.NewResolver(
			c.LegacyIdentity.Provider,
			time.Duration(c.LegacyIdentity.TimeoutMS)*time.Millisecond,
			identity.NewPortalRPCVerifier(portalClient),
			identity.NewRepository(mysqlDB),
		)
		authMw.WithLegacyAuthenticator(legacyResolver)
	}

	// 注册所有路由
	registerPublicRoutes(server, authHandler)
	registerAuthRoutes(server, authHandler, authMw)
	registerUserRoutes(server, userHandler, authMw)
	registerRoleRoutes(server, roleHandler, authMw)
	registerProjectRoutes(server, projectHandler, authMw)
	registerClusterRoutes(server, clusterHandler, authMw)
	registerCredentialRoutes(server, credentialHandler, authMw)
	registerDepartmentRoutes(server, departmentHandler, authMw)
	registerAccessRoutes(server, accessHandler, authMw)

	fmt.Printf("AxiOps Platform API started: %s:%d\n", c.Host, c.Port)
	server.Start()
}

func registerDepartmentRoutes(server *rest.Server, h *department.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/departments", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/departments", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPut, Path: "/api/v1/departments/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/departments/:id", Handler: mw.Handle(h.Delete)},
	})
}

func registerAccessRoutes(server *rest.Server, h *access.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/auth/menus", Handler: mw.Handle(h.CurrentMenus)},
		{Method: http.MethodGet, Path: "/api/v1/auth/access", Handler: mw.Handle(h.CurrentAccess)},
		{Method: http.MethodGet, Path: "/api/v1/access/menus", Handler: mw.Handle(h.ListMenus)},
		{Method: http.MethodGet, Path: "/api/v1/access/bindings", Handler: mw.Handle(h.ListBindings)},
		{Method: http.MethodPost, Path: "/api/v1/access/bindings", Handler: mw.Handle(h.CreateBinding)},
		{Method: http.MethodPut, Path: "/api/v1/access/bindings/:id", Handler: mw.Handle(h.UpdateBinding)},
		{Method: http.MethodDelete, Path: "/api/v1/access/bindings/:id", Handler: mw.Handle(h.DeleteBinding)},
	})
}

func registerPublicRoutes(server *rest.Server, h *auth.Handler) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/auth/login", Handler: h.Login},
		{Method: http.MethodPost, Path: "/api/v1/auth/refresh", Handler: h.RefreshToken},
		{Method: http.MethodGet, Path: "/api/v1/system/health", Handler: healthHandler},
	})
}

func registerAuthRoutes(server *rest.Server, h *auth.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/auth/current-user", Handler: mw.Handle(h.GetCurrentUser)},
		{Method: http.MethodPost, Path: "/api/v1/auth/logout", Handler: mw.Handle(h.Logout)},
	})
}

func registerUserRoutes(server *rest.Server, h *user.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/users", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/users/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/users", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPut, Path: "/api/v1/users/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/users/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodPost, Path: "/api/v1/users/:id/reset-password", Handler: mw.Handle(h.ResetPassword)},
		{Method: http.MethodPost, Path: "/api/v1/users/:id/roles", Handler: mw.Handle(h.SetRoles)},
		{Method: http.MethodPut, Path: "/api/v1/users/password", Handler: mw.Handle(h.UpdatePassword)},
	})
}

func registerRoleRoutes(server *rest.Server, h *role.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/roles", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/roles/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/roles", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPut, Path: "/api/v1/roles/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/roles/:id", Handler: mw.Handle(h.Delete)},
	})
}

func registerProjectRoutes(server *rest.Server, h *project.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/projects", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/projects/my", Handler: mw.Handle(h.MyProjects)},
		{Method: http.MethodGet, Path: "/api/v1/projects/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodGet, Path: "/api/v1/projects/:id/delivery-profile", Handler: mw.Handle(h.GetDeliveryProfile)},
		{Method: http.MethodPost, Path: "/api/v1/projects", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPut, Path: "/api/v1/projects/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodPut, Path: "/api/v1/projects/:id/delivery-profile", Handler: mw.Handle(h.ConfirmDeliveryProfile)},
		{Method: http.MethodDelete, Path: "/api/v1/projects/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodGet, Path: "/api/v1/projects/:id/deletion-check", Handler: mw.Handle(h.CheckDeletion)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:id/members", Handler: mw.Handle(h.AddMember)},
		{Method: http.MethodDelete, Path: "/api/v1/projects/:id/members", Handler: mw.Handle(h.RemoveMember)},
	})
}

func registerClusterRoutes(server *rest.Server, h *cluster.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/clusters", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/clusters/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/clusters", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPut, Path: "/api/v1/clusters/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/clusters/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodPost, Path: "/api/v1/clusters/:id/test-connect", Handler: mw.Handle(h.TestConnect)},
	})
}

func registerCredentialRoutes(server *rest.Server, h *credential.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/credentials", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/credentials/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/credentials", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPost, Path: "/api/v1/credentials/:id/derive-secret", Handler: mw.Handle(h.DeriveSecret)},
		{Method: http.MethodPut, Path: "/api/v1/credentials/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/credentials/:id", Handler: mw.Handle(h.Delete)},
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]string{
		"status":  "ok",
		"service": "axiops-platform-api",
	})
}
