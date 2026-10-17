package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/deploypreview"
	"github.com/ChenJustin666/axiops/internal/cicd/envworkload"
	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
	"github.com/ChenJustin666/axiops/internal/cicd/hostagent"
	"github.com/ChenJustin666/axiops/internal/cicd/hostimport"
	"github.com/ChenJustin666/axiops/internal/cicd/notification"
	"github.com/ChenJustin666/axiops/internal/cicd/projectenv"
	"github.com/ChenJustin666/axiops/internal/cicd/release"
	"github.com/ChenJustin666/axiops/internal/cicd/templatelib"
	"github.com/ChenJustin666/axiops/internal/cicd/webhook"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/cicd/yamlimport"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/ChenJustin666/axiops/migrations"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

func main() {
	configFile := flag.String("f", "configs/cicd-api.yaml", "config file path")
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
	if strings.TrimSpace(c.CredentialEncryptionKey) == "" {
		log.Fatal("CredentialEncryptionKey must be configured for code source credentials")
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
	box, err := secretbox.New(c.CredentialEncryptionKey)
	if err != nil {
		log.Fatalf("initialize credential decryption: %v", err)
	}
	kubernetesFactory := workflow.NewSQLKubernetesClientFactory(
		workflow.NewSQLClusterConnectionStore(mysqlDB), box,
		workflow.KubernetesFactoryOptions{
			QPS: c.Kubernetes.QPS, Burst: c.Kubernetes.Burst,
			Timeout:  time.Duration(c.Kubernetes.TimeoutSeconds) * time.Second,
			CacheTTL: time.Duration(c.Kubernetes.ClientCacheSecs) * time.Second,
		},
	)

	// Logic
	authorizer := authz.NewSQLAuthorizer(mysqlDB)
	logArchive, err := build.NewLogArchiveFromEnvironment()
	if err != nil {
		log.Fatalf("initialize external build logs: %v", err)
	}
	appsvcLogic := appsvc.NewLogic(mysqlDB, authorizer)
	buildLogic := build.NewLogic(mysqlDB, authorizer).WithLogArchive(logArchive).WithCredentialDecryptor(box)
	releaseLogic := release.NewLogic(mysqlDB)
	projectEnvLogic := projectenv.NewLogic(mysqlDB, authorizer).WithDecryptor(box)
	deployPreviewLogic := deploypreview.NewLogic(mysqlDB, authorizer, box, kubernetesFactory, c.Kubernetes.FieldManager)
	envWorkloadLogic := envworkload.NewLogic(projectEnvLogic, kubernetesFactory)
	workflowLogic := workflow.NewLogic(mysqlDB, authorizer).WithRestartFactory(kubernetesFactory).WithHelmDecryptor(mysqlDB, box)
	envWorkloadLogic.WithRestartCapabilities(workflowLogic)
	codeSourceLogic := codesource.NewLogic(mysqlDB, authorizer, c.CredentialEncryptionKey)
	projectEnvLogic.WithCodeSources(codeSourceLogic)

	// Handlers
	appsvcHandler := appsvc.NewHandler(appsvcLogic)
	buildHandler := build.NewHandler(buildLogic)
	releaseHandler := release.NewHandler(releaseLogic)
	projectEnvHandler := projectenv.NewHandler(projectEnvLogic)
	deployPreviewHandler := deploypreview.NewHandler(deployPreviewLogic)
	envWorkloadHandler := envworkload.NewHandler(envWorkloadLogic)
	workflowHandler := workflow.NewHandler(workflowLogic)
	codeSourceHandler := codesource.NewHandler(codeSourceLogic)
	webhookHandler := webhook.NewHandler(mysqlDB, build.NewWorkflowAdapter(buildLogic), c.CredentialEncryptionKey)
	yamlImportHandler := yamlimport.NewHandler(yamlimport.NewLogic(appsvcLogic, codeSourceLogic))
	helmImportHandler := helmimport.NewHandler(helmimport.NewLogic(appsvcLogic, codeSourceLogic))
	hostImportHandler := hostimport.NewHandler(hostimport.NewLogic(appsvcLogic, codeSourceLogic))
	notificationHandler := notification.NewHandler(notification.NewLogic(mysqlDB, authorizer, c.CredentialEncryptionKey))
	hostAgentHandler := hostagent.NewHandler(hostagent.NewLogic(mysqlDB, authorizer, 2*time.Minute))

	// HTTP 服务
	server := rest.MustNewServer(rest.RestConf{
		Host: c.Host,
		Port: c.Port,
	})
	defer server.Stop()
	server.Use(middleware.SyncPathValues)

	authMw := middleware.NewAuthMiddleware(c.JWT.Secret).WithAudit(middleware.NewAuditMiddleware(mysqlDB))
	templateHandler := templatelib.NewHandler(mysqlDB, authorizer)
	templateCtx, cancelTemplates := context.WithTimeout(context.Background(), 30*time.Second)
	err = templateHandler.Seed(templateCtx)
	cancelTemplates()
	if err != nil {
		log.Fatalf("initialize template library: %v", err)
	}
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/delivery-templates", Handler: authMw.Handle(templateHandler.List)},
		{Method: http.MethodPost, Path: "/api/v1/delivery-templates", Handler: authMw.Handle(templateHandler.Save)},
		{Method: http.MethodPut, Path: "/api/v1/delivery-templates/:id", Handler: authMw.Handle(templateHandler.Save)},
		{Method: http.MethodDelete, Path: "/api/v1/delivery-templates/:id", Handler: authMw.Handle(templateHandler.Archive)},
		{Method: http.MethodPost, Path: "/api/v1/delivery-templates/preview", Handler: authMw.Handle(templateHandler.Preview)},
		{Method: http.MethodGet, Path: "/api/v1/delivery-templates/:id/versions", Handler: authMw.Handle(templateHandler.Versions)},
		{Method: http.MethodGet, Path: "/api/v1/delivery-templates/:id/versions/:revision", Handler: authMw.Handle(templateHandler.Version)},
		{Method: http.MethodGet, Path: "/api/v1/delivery-templates/:id/usages", Handler: authMw.Handle(templateHandler.Usages)},
		{Method: http.MethodGet, Path: "/api/v1/services/:id/helm-template", Handler: authMw.Handle(templateHandler.ServiceTemplate)},
	})

	// 路由注册
	registerPublicRoutes(server)
	registerAppServiceRoutes(server, appsvcHandler, authMw)
	registerBuildRoutes(server, buildHandler, authMw)
	registerReleaseRoutes(server, releaseHandler, authMw)
	registerProjectEnvironmentRoutes(server, projectEnvHandler, authMw)
	registerEnvWorkloadRoutes(server, envWorkloadHandler, authMw)
	registerDeployPreviewRoutes(server, deployPreviewHandler, authMw)
	registerWorkflowRoutes(server, workflowHandler, authMw)
	registerCodeSourceRoutes(server, codeSourceHandler, authMw)
	registerWebhookRoutes(server, webhookHandler)
	registerYAMLImportRoutes(server, yamlImportHandler, authMw)
	registerHelmImportRoutes(server, helmImportHandler, authMw)
	registerHostImportRoutes(server, hostImportHandler, authMw)
	registerNotificationRoutes(server, notificationHandler, authMw)
	registerHostAgentRoutes(server, hostAgentHandler, authMw)

	fmt.Printf("AxiOps CI/CD API started: %s:%d\n", c.Host, c.Port)
	server.Start()
}

func registerDeployPreviewRoutes(server *rest.Server, h *deploypreview.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{{
		Method: http.MethodGet, Path: "/api/v1/environments/:id/render-preview", Handler: mw.Handle(h.Preview),
	}})
}

func registerHostAgentRoutes(server *rest.Server, h *hostagent.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/host-agents", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/host-agent-registration-tokens", Handler: mw.Handle(h.CreateRegistrationToken)},
		{Method: http.MethodPost, Path: "/api/v1/agent/register", Handler: h.Register},
		{Method: http.MethodPost, Path: "/api/v1/agent/heartbeat", Handler: h.AgentAuth(h.Heartbeat)},
		{Method: http.MethodGet, Path: "/api/v1/agent/commands/poll", Handler: h.AgentAuth(h.Poll)},
		{Method: http.MethodPost, Path: "/api/v1/agent/commands/:id/report", Handler: h.AgentAuth(h.Report)},
	})
}

func registerNotificationRoutes(server *rest.Server, h *notification.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/notification-subscriptions", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/notification-subscriptions", Handler: mw.Handle(h.Create)},
		{Method: http.MethodDelete, Path: "/api/v1/notification-subscriptions/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodGet, Path: "/api/v1/notification-deliveries", Handler: mw.Handle(h.ListDeliveries)},
	})
}

func registerYAMLImportRoutes(server *rest.Server, h *yamlimport.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/services/:id/yaml-import/preview", Handler: mw.Handle(h.Preview)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/yaml-import", Handler: mw.Handle(h.Import)},
	})
}

func registerHelmImportRoutes(server *rest.Server, h *helmimport.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/services/:id/helm-import/preview", Handler: mw.Handle(h.Preview)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/helm-import", Handler: mw.Handle(h.Import)},
	}, rest.WithTimeout(2*time.Minute))
}

func registerHostImportRoutes(server *rest.Server, h *hostimport.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/v1/services/:id/host-import/preview", Handler: mw.Handle(h.Preview)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/host-import", Handler: mw.Handle(h.Import)},
	})
}

func registerWebhookRoutes(server *rest.Server, h *webhook.Handler) {
	// Provider signatures are the authentication boundary; JWT middleware would reject Git callbacks.
	server.AddRoutes([]rest.Route{{Method: http.MethodPost, Path: "/api/v1/webhooks/git", Handler: h.HandleGitWebhook}})
}

func registerCodeSourceRoutes(server *rest.Server, h *codesource.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/code-sources", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/code-sources", Handler: mw.Handle(h.CreateGlobal)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/code-sources", Handler: mw.Handle(h.Create)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id", Handler: mw.Handle(h.Get)},
		{Method: http.MethodPut, Path: "/api/v1/code-sources/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/code-sources/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodPost, Path: "/api/v1/code-sources/:id/check", Handler: mw.Handle(h.Check)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id/repositories", Handler: mw.Handle(h.ListRepositories)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id/repository-files", Handler: mw.Handle(h.ListRepoFiles)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id/branches", Handler: mw.Handle(h.ListBranches)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id/tags", Handler: mw.Handle(h.ListTags)},
		{Method: http.MethodGet, Path: "/api/v1/code-sources/:id/commits", Handler: mw.Handle(h.GetCommit)},
	})
}

func registerWorkflowRoutes(server *rest.Server, h *workflow.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/restart-preview", Handler: mw.Handle(h.PreviewRestart)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/restarts", Handler: mw.Handle(h.SubmitRestart)},
		{Method: http.MethodGet, Path: "/api/v1/projects/:project_id/workflows", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/workflows", Handler: mw.Handle(h.Create)},
		{Method: http.MethodGet, Path: "/api/v1/workflows/:id", Handler: mw.Handle(h.Get)},
		{Method: http.MethodPut, Path: "/api/v1/workflows/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodPost, Path: "/api/v1/workflows/:id/validate", Handler: mw.Handle(h.Validate)},
		{Method: http.MethodPost, Path: "/api/v1/workflows/:id/runs", Handler: mw.Handle(h.Run)},
		{Method: http.MethodGet, Path: "/api/v1/workflows/:id/runs", Handler: mw.Handle(h.ListRuns)},
		{Method: http.MethodGet, Path: "/api/v1/workflow-runs/:id", Handler: mw.Handle(h.GetRun)},
		{Method: http.MethodGet, Path: "/api/v1/workflow-runs/:id/rollback-options", Handler: mw.Handle(h.RollbackOptions)},
		{Method: http.MethodGet, Path: "/api/v1/workflow-runs/:id/verification-options", Handler: mw.Handle(h.VerificationOptions)},
		{Method: http.MethodPost, Path: "/api/v1/workflow-runs/:id/verification", Handler: mw.Handle(h.SaveVerification)},
		{Method: http.MethodGet, Path: "/api/v1/project-environments/:id/production-policy", Handler: mw.Handle(h.GetProductionPolicy)},
		{Method: http.MethodPut, Path: "/api/v1/project-environments/:id/production-policy", Handler: mw.Handle(h.SaveProductionPolicy)},
		{Method: http.MethodGet, Path: "/api/v1/workflows/:id/production-options", Handler: mw.Handle(h.ProductionOptions)},
		{Method: http.MethodGet, Path: "/api/v1/workflows/:id/production-preview", Handler: mw.Handle(h.ProductionPreview)},
		{Method: http.MethodPost, Path: "/api/v1/workflows/:id/production-runs", Handler: mw.Handle(h.SubmitProduction)},
		{Method: http.MethodGet, Path: "/api/v1/workflow-runs/:id/rollback-preview", Handler: mw.Handle(h.PreviewRollback)},
		{Method: http.MethodPost, Path: "/api/v1/workflow-runs/:id/cancel", Handler: mw.Handle(h.Cancel)},
		{Method: http.MethodPost, Path: "/api/v1/workflow-runs/:id/approve", Handler: mw.Handle(h.Approve)},
		{Method: http.MethodPost, Path: "/api/v1/deployment-snapshots/:id/rollback", Handler: mw.Handle(h.Rollback)},
	})
}

func registerProjectEnvironmentRoutes(server *rest.Server, h *projectenv.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/projects/:project_id/environments", Handler: mw.Handle(h.List)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/environments", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/helm-values/read", Handler: mw.Handle(h.ReadHelmValues)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id", Handler: mw.Handle(h.Get)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/helm-shared-values", Handler: mw.Handle(h.HelmSharedValues)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/helm-shared-values", Handler: mw.Handle(h.HelmSharedValues)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/services", Handler: mw.Handle(h.CreateConfiguredService)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/clone", Handler: mw.Handle(h.Clone)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/variable-set", Handler: mw.Handle(h.GetVariableSet)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/services/:service_id/variable-set", Handler: mw.Handle(h.GetServiceVariableSet)},
		{Method: http.MethodPut, Path: "/api/v1/variable-sets/:id", Handler: mw.Handle(h.UpdateVariableSet)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/services/:service_id/config", Handler: mw.Handle(h.GetServiceConfig)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/services/:service_id/assets", Handler: mw.Handle(h.SaveServiceAssets)},
		{Method: http.MethodPost, Path: "/api/v1/services/:service_id/environment-config", Handler: mw.Handle(h.SaveServiceConfig)},
		{Method: http.MethodPost, Path: "/api/v1/services/:service_id/helm-chart/validate", Handler: mw.Handle(h.ValidateHelmChart)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/services/:service_id/override", Handler: mw.Handle(h.GetServiceOverride)},
		{Method: http.MethodPut, Path: "/api/v1/environments/:id/services/:service_id/override", Handler: mw.Handle(h.SaveServiceOverride)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/resources", Handler: mw.Handle(h.ListResources)},
		{Method: http.MethodPost, Path: "/api/v1/environments/:id/resources", Handler: mw.Handle(h.SaveResource)},
		{Method: http.MethodPut, Path: "/api/v1/environments/:id/resources", Handler: mw.Handle(h.SaveResource)},
		{Method: http.MethodDelete, Path: "/api/v1/environments/:id/resources/:resource_id", Handler: mw.Handle(h.DeleteResource)},
	}, rest.WithTimeout(2*time.Minute), rest.WithMaxBytes(4<<20))
}

func registerEnvWorkloadRoutes(server *rest.Server, h *envworkload.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/workloads", Handler: mw.Handle(h.Workloads)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/pods", Handler: mw.Handle(h.Pods)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/services", Handler: mw.Handle(h.Services)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/ingresses", Handler: mw.Handle(h.Ingresses)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/pvcs", Handler: mw.Handle(h.PVCs)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/configs", Handler: mw.Handle(h.Configs)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/resource-detail", Handler: mw.Handle(h.ResourceDetail)},
		{Method: http.MethodGet, Path: "/api/v1/environments/:id/pod-logs", Handler: mw.Handle(h.PodLogs)},
	})
}

func registerPublicRoutes(server *rest.Server) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/system/health", Handler: healthHandler},
	})
}

func registerAppServiceRoutes(server *rest.Server, h *appsvc.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/services", Handler: mw.Handle(h.List)},
		{Method: http.MethodGet, Path: "/api/v1/services/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/services", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPost, Path: "/api/v1/projects/:project_id/services", Handler: mw.Handle(h.CreateForProject)},
		{Method: http.MethodPut, Path: "/api/v1/services/:id", Handler: mw.Handle(h.Update)},
		{Method: http.MethodDelete, Path: "/api/v1/services/:id", Handler: mw.Handle(h.Delete)},
		{Method: http.MethodGet, Path: "/api/v1/services/:id/deletion-check", Handler: mw.Handle(h.CheckDeletion)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/restore", Handler: mw.Handle(h.Restore)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/purge", Handler: mw.Handle(h.Purge)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/rename", Handler: mw.Handle(h.Rename)},
		{Method: http.MethodDelete, Path: "/api/v1/services/:id/environment-bindings/:environment_id", Handler: mw.Handle(h.DetachEnvironment)},
		{Method: http.MethodGet, Path: "/api/v1/services/:id/revisions", Handler: mw.Handle(h.ListRevisions)},
		{Method: http.MethodPost, Path: "/api/v1/services/:id/revisions", Handler: mw.Handle(h.CreateRevision)},
	}, rest.WithMaxBytes(8<<20))
}

func registerBuildRoutes(server *rest.Server, h *build.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		// 构建运行
		{Method: http.MethodGet, Path: "/api/v1/builds", Handler: mw.Handle(h.ListRuns)},
		{Method: http.MethodGet, Path: "/api/v1/build-registries", Handler: mw.Handle(h.ListImageRegistries)},
		{Method: http.MethodPost, Path: "/api/v1/build-registries/test", Handler: mw.Handle(h.TestImageRegistry)},
		{Method: http.MethodPost, Path: "/api/v1/build-registries", Handler: mw.Handle(h.SaveImageRegistry)},
		{Method: http.MethodGet, Path: "/api/v1/builds/:id", Handler: mw.Handle(h.GetDetail)},
		{Method: http.MethodPost, Path: "/api/v1/builds", Handler: mw.Handle(h.Trigger)},
		{Method: http.MethodPost, Path: "/api/v1/builds/:id/cancel", Handler: mw.Handle(h.Cancel)},
		{Method: http.MethodPost, Path: "/api/v1/builds/:id/retry", Handler: mw.Handle(h.Retry)},
		{Method: http.MethodGet, Path: "/api/v1/builds/:id/logs", Handler: mw.Handle(h.GetLogs)},
		{Method: http.MethodGet, Path: "/api/v1/builds/:id/logs/stream", Handler: mw.Handle(h.StreamLogs)},
		{Method: http.MethodGet, Path: "/api/v1/build-metrics", Handler: mw.Handle(h.Metrics)},
		{Method: http.MethodGet, Path: "/api/v1/build-concurrency", Handler: mw.Handle(h.GetConcurrencyPolicy)},
		{Method: http.MethodPut, Path: "/api/v1/build-concurrency", Handler: mw.Handle(h.SaveConcurrencyPolicy)},
		{Method: http.MethodGet, Path: "/api/v1/build-platform-settings", Handler: mw.Handle(h.GetPlatformBuildSettings)},
		{Method: http.MethodPut, Path: "/api/v1/build-platform-settings", Handler: mw.Handle(h.SavePlatformBuildSettings)},
		// 构建配置
		{Method: http.MethodGet, Path: "/api/v1/services/:id/build-config", Handler: mw.Handle(h.GetBuildConfig)},
		{Method: http.MethodPut, Path: "/api/v1/services/:id/build-config", Handler: mw.Handle(h.SaveBuildConfig)},
		// 构建模板
		{Method: http.MethodGet, Path: "/api/v1/build-templates", Handler: mw.Handle(h.ListTemplates)},
	})
}

func registerReleaseRoutes(server *rest.Server, h *release.Handler, mw *middleware.AuthMiddleware) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/v1/releases", Handler: mw.Handle(h.ListByService)},
		{Method: http.MethodGet, Path: "/api/v1/releases/:id", Handler: mw.Handle(h.GetByID)},
		{Method: http.MethodPost, Path: "/api/v1/releases", Handler: mw.Handle(h.Create)},
		{Method: http.MethodPost, Path: "/api/v1/releases/:id/approve", Handler: mw.Handle(h.Approve)},
		{Method: http.MethodPost, Path: "/api/v1/releases/rollback", Handler: mw.Handle(h.Rollback)},
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]string{
		"status":  "ok",
		"service": "axiops-cicd-api",
	})
}
