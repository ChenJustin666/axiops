package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/assetdeploy"
	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/cicd/hostagent"
	"github.com/ChenJustin666/axiops/internal/cicd/imagepull"
	"github.com/ChenJustin666/axiops/internal/cicd/notification"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/config"
	"github.com/ChenJustin666/axiops/internal/shared/db"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/ChenJustin666/axiops/migrations"
	"github.com/zeromicro/go-zero/core/conf"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

func main() {
	configFile := flag.String("f", "configs/workflow-worker.yaml", "config file path")
	flag.Parse()

	var c config.WorkflowWorkerConfig
	conf.MustLoad(*configFile, &c)
	if err := inheritDatabaseConfig(&c); err != nil {
		log.Fatalf("load shared database configuration: %v", err)
	}
	c.CredentialEncryptionKey = config.CredentialEncryptionKey(c.CredentialEncryptionKey)
	mysqlDB, err := db.NewMySQL(&c.MySQL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer mysqlDB.Close()

	// Workers never apply DDL. Deployment must run axiops-migrate first.
	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = migrations.Ensure(migrationCtx, mysqlDB, false, 30)
	cancelMigration()
	if err != nil {
		log.Fatalf("database migration check failed: %v", err)
	}
	// Platform settings are managed from the CI resource page. The config file
	// remains a bootstrap fallback for installations that have not saved them.
	platformSettings, settingsErr := build.NewRepository(mysqlDB).FindPlatformBuildSettings(context.Background())
	if settingsErr != nil {
		log.Fatalf("load platform CI settings: %v", settingsErr)
	}
	// Disabling dispatch must not disable observation of already-running work.
	if platformSettings != nil && platformSettings.ClusterID > 0 {
		c.Build.Enabled = true
		c.Build.ClusterID = platformSettings.ClusterID
		c.Build.Namespace = platformSettings.Namespace
		if platformSettings.BuilderImage != "" {
			c.Build.Defaults.BuilderImage = platformSettings.BuilderImage
		}
		if platformSettings.BuildCommand != "" {
			c.Build.Defaults.BuildCommand = platformSettings.BuildCommand
		}
		if platformSettings.TimeoutSeconds > 0 {
			c.Build.Defaults.TimeoutSeconds = platformSettings.TimeoutSeconds
		}
		// A saved empty value explicitly removes the platform request/limit;
		// it must override even a nonempty bootstrap configuration.
		c.Build.Defaults.ResourceCPU = platformSettings.ResourceCPU
		c.Build.Defaults.ResourceMemory = platformSettings.ResourceMemory
		c.Build.Defaults.CachePVCName = platformSettings.CachePVCName
		if platformSettings.CacheDir != "" {
			c.Build.Defaults.CacheDir = platformSettings.CacheDir
		}
		c.Build.Defaults.JobRetentionSeconds = platformSettings.JobRetentionSeconds
	}
	if err := validateConfig(c); err != nil {
		log.Fatalf("workflow worker configuration is invalid: %v", err)
	}

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
	yamlProvider := workflow.NewKubernetesYAMLProvider(kubernetesFactory, c.Kubernetes.FieldManager, c.Kubernetes.ForceApply)
	helmProvider := workflow.NewOfficialHelmProvider(mysqlDB, kubernetesFactory)
	snapshotStore := workflow.NewSQLDeploymentSnapshotStore(mysqlDB)
	yamlProvider.WithHealthTargets(snapshotStore)
	buildLogic := build.NewLogic(mysqlDB, authz.NewSQLAuthorizer(mysqlDB))
	executors, err := workflow.NewExecutorRegistry(
		workflow.NewBuildExecutor(build.NewWorkflowAdapter(buildLogic)),
		workflow.NewIsolatedTestExecutor(build.NewWorkflowAdapter(buildLogic)),
		workflow.NewNotifyExecutor(),
		workflow.NewWorkloadRestartExecutor(mysqlDB, kubernetesFactory, authz.NewSQLAuthorizer(mysqlDB)),
		workflow.NewYAMLDeployExecutor(
			workflow.NewDefaultYAMLResolver(workflow.NewSQLYAMLSourceStore(mysqlDB)),
			yamlProvider,
			snapshotStore,
		).WithDeploymentPreparer(workflow.DeploymentPreparers{assetdeploy.NewManager(mysqlDB, box, kubernetesFactory), imagepull.NewManager(build.NewSQLRegistryAuthStore(mysqlDB, box), kubernetesFactory)}),
		workflow.NewYAMLRollbackExecutor(snapshotStore, yamlProvider).WithDeploymentPreparer(workflow.DeploymentPreparers{imagepull.NewManager(build.NewSQLRegistryAuthStore(mysqlDB, box), kubernetesFactory), assetdeploy.NewManager(mysqlDB, box, kubernetesFactory)}),
		workflow.NewHelmDeployExecutor(
			workflow.NewDefaultHelmResolver(workflow.NewSQLHelmSourceStore(mysqlDB), box),
			helmProvider,
			snapshotStore,
		).WithDeploymentPreparer(imagepull.NewManager(build.NewSQLRegistryAuthStore(mysqlDB, box), kubernetesFactory)),
		workflow.NewHelmRollbackExecutor(snapshotStore, helmProvider),
		workflow.NewHostDeployExecutor(
			workflow.NewDefaultHostResolver(workflow.NewSQLHostSourceStore(mysqlDB)),
			workflow.NewHostAgentProvider(hostagent.NewRepository(mysqlDB)),
		),
	)
	if err != nil {
		log.Fatalf("register workflow executors: %v", err)
	}
	var buildWorker buildCommandProcessor
	var buildObserver buildCommandProcessor
	if c.Build.Enabled {
		logArchive, err := build.NewLogArchiveFromEnvironment()
		if err != nil {
			log.Fatalf("initialize external build logs: %v", err)
		}
		outputVerifier, err := build.NewOutputVerifierFromEnvironment(mysqlDB, box)
		if err != nil {
			log.Fatalf("initialize build output verification: %v", err)
		}
		restConfig, err := kubernetesFactory.RESTConfig(context.Background(), c.Build.ClusterID)
		if err != nil {
			log.Fatalf("load build cluster: %v", err)
		}
		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			log.Fatalf("create build cluster client: %v", err)
		}
		jobs := clientset.BatchV1().Jobs(c.Build.Namespace)
		checkCtx, cancelCheck := context.WithTimeout(context.Background(), 30*time.Second)
		_, checkErr := clientset.CoreV1().Namespaces().Get(checkCtx, c.Build.Namespace, metav1.GetOptions{})
		if apierrors.IsNotFound(checkErr) {
			// The CI namespace is created on demand so a fresh cluster does not
			// crash the worker. Deployment target namespaces are managed by the
			// project environments, not here.
			_, createErr := clientset.CoreV1().Namespaces().Create(checkCtx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   c.Build.Namespace,
					Labels: map[string]string{"axiops.io/managed-by": "axiops-cicd"},
				},
			}, metav1.CreateOptions{})
			if createErr != nil {
				cancelCheck()
				log.Printf("CI namespace %q in cluster %d could not be created: %v", c.Build.Namespace, c.Build.ClusterID, createErr)
			} else {
				log.Printf("CI namespace %q created in cluster %d", c.Build.Namespace, c.Build.ClusterID)
			}
		} else if checkErr != nil {
			cancelCheck()
			log.Printf("CI namespace %q in cluster %d is unavailable; build commands will retry: %v", c.Build.Namespace, c.Build.ClusterID, checkErr)
		}
		cancelCheck()
		log.Printf("CI builds bound to cluster=%d namespace=%s; deployment targets come from project environments", c.Build.ClusterID, c.Build.Namespace)
		pods := clientset.CoreV1().Pods(c.Build.Namespace)
		secrets := clientset.CoreV1().Secrets(c.Build.Namespace)
		logLimit := int64(1 << 20)
		if logArchive != nil {
			logLimit = 8 << 20
		}
		kubeJobProvider := build.NewKubeJobProvider(c.Build.Namespace, jobs, jobs, pods).WithRuntimeData(
			build.NewKubernetesSecretStore(secrets), build.NewKubernetesPodLogReader(pods), logLimit,
		).WithDefaults(build.KubeJobDefaults{BuildCommand: c.Build.Defaults.BuildCommand, BuilderImage: c.Build.Defaults.BuilderImage, CachePVCName: c.Build.Defaults.CachePVCName, CacheDir: c.Build.Defaults.CacheDir, JobRetentionSeconds: int32(c.Build.Defaults.JobRetentionSeconds), TimeoutSeconds: c.Build.Defaults.TimeoutSeconds, ResourceCPU: c.Build.Defaults.ResourceCPU, ResourceMemory: c.Build.Defaults.ResourceMemory})
		if platformSettings != nil {
			kubeJobProvider.WithExecutionPolicy(platformSettings.ExecutionPolicy)
		}
		providers, err := build.NewProviderRegistry(kubeJobProvider)
		if err != nil {
			log.Fatalf("register build providers: %v", err)
		}
		buildRepository := build.NewRepository(mysqlDB).WithWorkerSettings(platformSettings)
		commandWorker := build.NewWorker(
			buildRepository, providers,
			build.NewSQLExecutionResolver(mysqlDB, c.Build.Namespace, box),
			c.Build.MaxAttempts, time.Duration(c.Build.RetryAfterSecs)*time.Second,
		).WithLogPersistence(buildRepository, logArchive).WithOperationTimeout(time.Duration(c.Build.StaleAfterSecs) * time.Second / 2)
		buildWorker = &buildProcessor{
			commands: commandWorker, stale: buildRepository,
			maxAttempts: c.Build.MaxAttempts, staleAfter: time.Duration(c.Build.StaleAfterSecs) * time.Second,
		}
		buildObserver = &buildObservationProcessor{reconciler: build.NewReconciler(buildRepository, providers).WithLogArchive(logArchive).WithOutputVerifier(outputVerifier)}
	}
	var notificationWorker buildCommandProcessor
	if c.Notification.Enabled {
		notificationWorker = notification.NewWorker(notification.NewRepository(mysqlDB), box, c.Notification.MaxAttempts, time.Duration(c.Notification.LeaseSeconds)*time.Second)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workerID := resolveWorkerID(c.Worker.ID)
	log.Printf("AxiOps workflow worker started: id=%s", workerID)
	var group sync.WaitGroup
	lanes := c.Worker.Concurrency
	if lanes <= 0 {
		lanes = 4
	}
	for lane := 0; lane < lanes; lane++ {
		laneID := fmt.Sprintf("%s:w%d", workerID, lane)
		worker := workflow.NewWorker(workflow.NewRepository(mysqlDB), executors, time.Duration(c.Worker.LeaseSeconds)*time.Second, c.Worker.BatchSize)
		group.Add(1)
		go func() {
			defer group.Done()
			runWorker(ctx, worker, laneID, time.Duration(c.Worker.PollIntervalMS)*time.Millisecond, func(err error) { log.Printf("workflow worker tick failed: %v", err) })
		}()
	}
	if buildWorker != nil {
		group.Add(1)
		go func() {
			defer group.Done()
			runBuildWorker(ctx, buildWorker, workerID+":build", time.Duration(c.Build.PollIntervalMS)*time.Millisecond, func(err error) {
				log.Printf("build worker tick failed: %v", err)
			})
		}()
	}
	if notificationWorker != nil {
		group.Add(1)
		go func() {
			defer group.Done()
			runBuildWorker(ctx, notificationWorker, workerID+":notify", time.Duration(c.Notification.PollIntervalMS)*time.Millisecond, func(err error) {
				log.Printf("notification worker delivery failed: %v", err)
			})
		}()
	}
	if buildObserver != nil {
		group.Add(1)
		go func() {
			defer group.Done()
			runBuildWorker(ctx, buildObserver, workerID+":observe", time.Duration(c.Build.PollIntervalMS)*time.Millisecond, func(err error) { log.Printf("build observation failed: %v", err) })
		}()
	}
	group.Wait()
	log.Printf("AxiOps workflow worker stopped: id=%s", workerID)
}

func inheritDatabaseConfig(c *config.WorkflowWorkerConfig) error {
	if c == nil || strings.TrimSpace(c.DatabaseConfigFile) == "" {
		return nil
	}
	var shared struct {
		MySQL config.MySQLConfig
	}
	if err := conf.Load(strings.TrimSpace(c.DatabaseConfigFile), &shared); err != nil {
		return err
	}
	if strings.TrimSpace(shared.MySQL.Database) == "" {
		return fmt.Errorf("MySQL.Database is required in %s", c.DatabaseConfigFile)
	}
	c.MySQL = shared.MySQL
	return nil
}

type workflowTicker interface {
	Tick(context.Context, string) error
}

type buildCommandProcessor interface {
	ProcessOne(context.Context, string) (bool, error)
}

type buildProcessor struct {
	commands    *build.Worker
	stale       staleCommandRequeuer
	maxAttempts int
	staleAfter  time.Duration
	lastReap    time.Time
}

type staleCommandRequeuer interface {
	RequeueStaleCommands(context.Context, time.Time, int) (int64, error)
	ExpireQueuedRuns(context.Context, int) (int64, error)
}

func (p *buildProcessor) ProcessOne(ctx context.Context, workerID string) (bool, error) {
	if p.lastReap.IsZero() || time.Since(p.lastReap) >= time.Minute {
		if _, err := p.stale.RequeueStaleCommands(ctx, time.Now().Add(-p.staleAfter), p.maxAttempts); err != nil {
			return false, err
		}
		if _, err := p.stale.ExpireQueuedRuns(ctx, 100); err != nil {
			return false, err
		}
		p.lastReap = time.Now()
	}
	return p.commands.ProcessOne(ctx, workerID)
}

type buildObservationProcessor struct{ reconciler *build.Reconciler }

func (p *buildObservationProcessor) ProcessOne(ctx context.Context, _ string) (bool, error) {
	processed, err := p.reconciler.ReconcileOne(ctx)
	cleaned, cleanErr := p.reconciler.CleanupOne(ctx)
	expired, expireErr := p.reconciler.ExpireLogOne(ctx)
	return processed || cleaned || expired, errors.Join(err, cleanErr, expireErr)
}

func runWorker(ctx context.Context, worker workflowTicker, workerID string, interval time.Duration, onError func(error)) {
	tick := func() {
		if err := worker.Tick(ctx, workerID); err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
	}
	tick()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

func runBuildWorker(ctx context.Context, worker buildCommandProcessor, workerID string, interval time.Duration, onError func(error)) {
	for {
		processed, err := worker.ProcessOne(ctx, workerID)
		if err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		if ctx.Err() != nil {
			return
		}
		// Observing a still-running Job counts as processed; always pace polling.
		_ = processed
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func validateConfig(c config.WorkflowWorkerConfig) error {
	if c.Worker.Concurrency < 0 || c.Worker.Concurrency > 32 {
		return fmt.Errorf("Worker.Concurrency must be between 1 and 32")
	}
	if strings.TrimSpace(c.CredentialEncryptionKey) == "" {
		return fmt.Errorf("CredentialEncryptionKey is required")
	}
	if strings.TrimSpace(c.MySQL.Database) == "" {
		return fmt.Errorf("MySQL.Database is required")
	}
	if c.Worker.PollIntervalMS < 100 || c.Worker.PollIntervalMS > 60000 {
		return fmt.Errorf("Worker.PollIntervalMS must be between 100 and 60000")
	}
	if c.Worker.LeaseSeconds < 5 || c.Worker.LeaseSeconds > 3600 {
		return fmt.Errorf("Worker.LeaseSeconds must be between 5 and 3600")
	}
	if c.Worker.BatchSize < 1 || c.Worker.BatchSize > 1000 {
		return fmt.Errorf("Worker.BatchSize must be between 1 and 1000")
	}
	if len(c.Worker.ID) > 120 {
		return fmt.Errorf("Worker.ID must be at most 120 characters")
	}
	if c.Build.Enabled {
		if c.Build.ClusterID <= 0 || strings.TrimSpace(c.Build.Namespace) == "" {
			return fmt.Errorf("Build.ClusterID and Build.Namespace are required when build worker is enabled")
		}
		if len(validation.IsDNS1123Label(c.Build.Namespace)) > 0 {
			return fmt.Errorf("invalid CI namespace")
		}
		defaults := c.Build.Defaults
		if defaults.TimeoutSeconds != 0 && (defaults.TimeoutSeconds < 60 || defaults.TimeoutSeconds > 86400) {
			return fmt.Errorf("CI timeout must be 60..86400 seconds")
		}
		if defaults.JobRetentionSeconds < 0 || defaults.JobRetentionSeconds > 604800 {
			return fmt.Errorf("CI Job retention must be 0..604800 seconds")
		}
		for _, raw := range []string{defaults.ResourceCPU, defaults.ResourceMemory} {
			if raw != "" {
				q, err := resource.ParseQuantity(raw)
				if err != nil || q.Sign() <= 0 {
					return fmt.Errorf("invalid CI default resources")
				}
			}
		}
		if defaults.CachePVCName != "" && len(validation.IsDNS1123Subdomain(defaults.CachePVCName)) > 0 {
			return fmt.Errorf("invalid CI cache PVC name")
		}
		if c.Build.PollIntervalMS < 100 || c.Build.PollIntervalMS > 60000 {
			return fmt.Errorf("Build.PollIntervalMS must be between 100 and 60000")
		}
		if c.Build.MaxAttempts < 1 || c.Build.MaxAttempts > 20 || c.Build.RetryAfterSecs < 1 || c.Build.RetryAfterSecs > 3600 || c.Build.StaleAfterSecs < 30 || c.Build.StaleAfterSecs > 86400 {
			return fmt.Errorf("Build retry settings are invalid")
		}
	}
	if c.Notification.Enabled {
		if c.Notification.PollIntervalMS < 100 || c.Notification.PollIntervalMS > 60000 {
			return fmt.Errorf("Notification.PollIntervalMS must be between 100 and 60000")
		}
		if c.Notification.MaxAttempts < 1 || c.Notification.MaxAttempts > 20 {
			return fmt.Errorf("Notification.MaxAttempts must be between 1 and 20")
		}
		if c.Notification.LeaseSeconds < 5 || c.Notification.LeaseSeconds > 3600 {
			return fmt.Errorf("Notification.LeaseSeconds must be between 5 and 3600")
		}
	}
	return nil
}

func resolveWorkerID(configured string) string {
	if configured = strings.TrimSpace(configured); configured != "" {
		return configured
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "workflow-worker"
	}
	value := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	if len(value) > 128 {
		return value[:128]
	}
	return value
}
