package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type HelmRESTConfigFactory interface {
	RESTConfig(context.Context, int64) (*rest.Config, error)
}

type OfficialHelmProvider struct {
	db      *sql.DB
	factory HelmRESTConfigFactory
}

func NewOfficialHelmProvider(db *sql.DB, factory HelmRESTConfigFactory) *OfficialHelmProvider {
	return &OfficialHelmProvider{db: db, factory: factory}
}

func (p *OfficialHelmProvider) Render(ctx context.Context, request HelmRequest) ([]byte, error) {
	if err := validateHelmRequest(request); err != nil {
		return nil, err
	}
	return renderHelmManifest(ctx, request, false)
}

// RenderPreview keeps Secret manifests in memory so Kubernetes can validate
// the real release. The preview API exposes only identities and redacted values.
func (p *OfficialHelmProvider) RenderPreview(ctx context.Context, request HelmRequest) ([]byte, error) {
	if err := validateHelmRequest(request); err != nil {
		return nil, err
	}
	return renderHelmManifest(ctx, request, false)
}

func (p *OfficialHelmProvider) UpgradeInstall(ctx context.Context, request HelmRequest) (HelmExecution, error) {
	if p == nil || p.db == nil || p.factory == nil {
		return HelmExecution{}, fmt.Errorf("official Helm provider is not configured")
	}
	if err := validateHelmRequest(request); err != nil {
		return HelmExecution{}, err
	}
	chart, err := loadImmutableHelmChart(request.Chart)
	if err != nil {
		return HelmExecution{}, err
	}
	preview, err := renderHelmManifest(ctx, request, false)
	if err != nil {
		return HelmExecution{}, err
	}
	atomic, err := helmAtomicPolicy(preview, nil, request.Namespace)
	if err != nil {
		return HelmExecution{}, err
	}
	requestHash, err := helmRequestHash("upgrade", request, 0)
	if err != nil {
		return HelmExecution{}, err
	}
	marker := "AxiOps " + request.IdempotencyKey
	record, created, err := p.beginExecution(ctx, helmExecutionRecord{
		IdempotencyKey: request.IdempotencyKey, RequestHash: requestHash, WorkflowRunID: request.RunID,
		NodeKey: request.NodeKey, Operation: "upgrade", ClusterID: request.ClusterID, Namespace: request.Namespace,
		ReleaseName: request.ReleaseName, Marker: marker, TimeoutSeconds: request.TimeoutSeconds,
	})
	if err != nil {
		return HelmExecution{}, err
	}
	if !created {
		return existingHelmExecution(record)
	}
	configuration, err := p.configuration(ctx, request.ClusterID, request.Namespace)
	if err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, err)
	}
	timeout := time.Duration(request.TimeoutSeconds) * time.Second
	labels, err := p.releaseOwnershipLabels(ctx, request.RunID, request.NodeKey)
	if err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, err)
	}
	var deployed *release.Release
	current, getErr := action.NewGet(configuration).Run(request.ReleaseName)
	switch {
	case getErr == nil && current != nil:
		if err := p.validateReleaseOwnership(ctx, current, request.ClusterID, request.Namespace, labels); err != nil {
			return HelmExecution{}, p.failExecution(ctx, record.ID, err)
		}
		atomic, err = helmAtomicPolicy(preview, []byte(current.Manifest), request.Namespace)
		if err != nil {
			return HelmExecution{}, p.failExecution(ctx, record.ID, err)
		}
		upgrade := action.NewUpgrade(configuration)
		upgrade.Namespace = request.Namespace
		upgrade.Atomic = atomic
		upgrade.Wait = true
		upgrade.WaitForJobs = true
		upgrade.CleanupOnFail = atomic
		upgrade.SkipCRDs = true
		upgrade.Timeout = timeout
		upgrade.MaxHistory = 20
		upgrade.Description = marker
		upgrade.Labels = labels
		upgrade.PostRenderer = helmPostRenderer{namespace: request.Namespace, prepare: request.PrepareManifest, rejectJobs: atomic}
		deployed, err = upgrade.RunWithContext(ctx, request.ReleaseName, chart, request.Values)
	case errors.Is(getErr, driver.ErrReleaseNotFound):
		install := action.NewInstall(configuration)
		install.ReleaseName = request.ReleaseName
		install.Namespace = request.Namespace
		install.Atomic = atomic
		install.Wait = true
		install.WaitForJobs = true
		install.SkipCRDs = true
		install.Timeout = timeout
		install.Description = marker
		install.Labels = labels
		install.PostRenderer = helmPostRenderer{namespace: request.Namespace, prepare: request.PrepareManifest, rejectJobs: atomic}
		deployed, err = install.RunWithContext(ctx, chart, request.Values)
	default:
		err = getErr
	}
	if err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm upgrade --install failed: %w", err))
	}
	if deployed == nil || deployed.Version <= 0 || deployed.Info == nil || deployed.Info.Status != release.StatusDeployed {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm release did not reach deployed state"))
	}
	if err := p.succeedExecution(ctx, record.ID, deployed.Version); err != nil {
		return HelmExecution{}, err
	}
	return HelmExecution{ExternalID: strconv.FormatInt(record.ID, 10), Completed: true, Revision: deployed.Version}, nil
}

func (p *OfficialHelmProvider) Rollback(ctx context.Context, request HelmRollbackRequest) (HelmExecution, error) {
	if p == nil || p.db == nil || p.factory == nil || request.RunID <= 0 || request.ClusterID <= 0 || request.NodeKey == "" || request.Namespace == "" || request.ReleaseName == "" || request.Revision <= 0 || request.TimeoutSeconds <= 0 || request.IdempotencyKey == "" {
		return HelmExecution{}, fmt.Errorf("invalid Helm rollback request")
	}
	configuration, err := p.configuration(ctx, request.ClusterID, request.Namespace)
	if err != nil {
		return HelmExecution{}, err
	}
	current, err := action.NewGet(configuration).Run(request.ReleaseName)
	if err != nil || current == nil || current.Version <= request.Revision {
		return HelmExecution{}, fmt.Errorf("Helm rollback source or target revision is invalid")
	}
	labels, err := p.releaseOwnershipLabels(ctx, request.RunID, request.NodeKey)
	if err != nil {
		return HelmExecution{}, err
	}
	if err := p.validateReleaseOwnership(ctx, current, request.ClusterID, request.Namespace, labels); err != nil {
		return HelmExecution{}, err
	}
	if hasJob, err := helmManifestHasJob([]byte(current.Manifest), request.Namespace); err != nil {
		return HelmExecution{}, err
	} else if hasJob {
		return HelmExecution{}, fmt.Errorf("当前 Release 包含一次性 Job，禁止通过回滚重复执行任务")
	}
	requestHash, err := helmRollbackHash(request)
	if err != nil {
		return HelmExecution{}, err
	}
	record, created, err := p.beginExecution(ctx, helmExecutionRecord{
		IdempotencyKey: request.IdempotencyKey, RequestHash: requestHash, Operation: "rollback",
		WorkflowRunID: request.RunID, NodeKey: request.NodeKey,
		ClusterID: request.ClusterID, Namespace: request.Namespace, ReleaseName: request.ReleaseName,
		TargetRevision: request.Revision, SourceRevision: current.Version, TimeoutSeconds: request.TimeoutSeconds,
	})
	if err != nil {
		return HelmExecution{}, err
	}
	if !created {
		return existingHelmExecution(record)
	}
	if request.ExpectedCurrentRevision > 0 && current.Version != request.ExpectedCurrentRevision {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm 当前 revision 在预览后已变化，已停止回滚"))
	}
	history := action.NewGet(configuration)
	history.Version = request.Revision
	target, err := history.Run(request.ReleaseName)
	if err != nil || target == nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm 历史 revision 已不存在，未执行回滚"))
	}
	if err := p.validateReleaseOwnership(ctx, target, request.ClusterID, request.Namespace, labels); err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, err)
	}
	if err := validateHelmRollbackHistory(request, []byte(target.Manifest)); err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, err)
	}
	rollback := action.NewRollback(configuration)
	rollback.Version = request.Revision
	rollback.Wait = true
	rollback.WaitForJobs = true
	rollback.CleanupOnFail = true
	rollback.Timeout = time.Duration(request.TimeoutSeconds) * time.Second
	rollback.MaxHistory = 20
	if err := rollback.Run(request.ReleaseName); err != nil {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm rollback failed: %w", err))
	}
	rolledBack, err := action.NewGet(configuration).Run(request.ReleaseName)
	if err != nil || rolledBack == nil || rolledBack.Version <= current.Version || rolledBack.Info == nil || rolledBack.Info.Status != release.StatusDeployed {
		return HelmExecution{}, p.failExecution(ctx, record.ID, fmt.Errorf("Helm rollback did not reach deployed state"))
	}
	if err := p.succeedExecution(ctx, record.ID, rolledBack.Version); err != nil {
		return HelmExecution{}, err
	}
	return HelmExecution{ExternalID: strconv.FormatInt(record.ID, 10), Completed: true, Revision: rolledBack.Version}, nil
}

func (p *OfficialHelmProvider) Status(ctx context.Context, externalID string) (HelmState, error) {
	id, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil || id <= 0 {
		return HelmState{}, fmt.Errorf("invalid Helm execution id")
	}
	record, err := p.findExecution(ctx, id)
	if err != nil {
		return HelmState{}, err
	}
	if record.Status == RunSucceeded {
		return HelmState{Status: RunSucceeded, Revision: record.HelmRevision}, nil
	}
	if record.Status == RunFailed {
		return HelmState{Status: RunFailed, ErrorMessage: record.ErrorMessage}, nil
	}
	configuration, err := p.configuration(ctx, record.ClusterID, record.Namespace)
	if err != nil {
		return HelmState{}, err
	}
	current, getErr := action.NewGet(configuration).Run(record.ReleaseName)
	matched := false
	if getErr == nil && current != nil && current.Info != nil {
		if record.Operation == "upgrade" {
			matched = current.Info.Description == record.Marker
		} else {
			matched = current.Version == record.SourceRevision+1 && current.Info.Description == fmt.Sprintf("Rollback to %d", record.TargetRevision)
		}
		if matched && current.Info.Status == release.StatusDeployed {
			if err := p.succeedExecution(ctx, record.ID, current.Version); err != nil {
				return HelmState{}, err
			}
			return HelmState{Status: RunSucceeded, Revision: current.Version}, nil
		}
		if matched && current.Info.Status == release.StatusFailed {
			failure := fmt.Errorf("Helm release reported failed state")
			_ = p.failExecution(ctx, record.ID, failure)
			return HelmState{Status: RunFailed, ErrorMessage: failure.Error()}, nil
		}
	}
	if time.Since(record.StartedAt) > time.Duration(record.TimeoutSeconds+60)*time.Second {
		failure := fmt.Errorf("Helm execution recovery timed out")
		_ = p.failExecution(ctx, record.ID, failure)
		return HelmState{Status: RunFailed, ErrorMessage: failure.Error()}, nil
	}
	return HelmState{Status: RunRunning}, nil
}

func validateHelmRequest(request HelmRequest) error {
	if request.ClusterID <= 0 || request.RunID <= 0 || strings.TrimSpace(request.NodeKey) == "" || strings.TrimSpace(request.Namespace) == "" || strings.TrimSpace(request.ReleaseName) == "" || len(request.Chart) == 0 || request.TimeoutSeconds <= 0 || request.TimeoutSeconds > 86400 || strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf("invalid Helm provider request")
	}
	return nil
}

type helmExecutionRecord struct {
	ID, WorkflowRunID, ClusterID                         int64
	IdempotencyKey, RequestHash, NodeKey, Operation      string
	Namespace, ReleaseName, Marker, Status, ErrorMessage string
	TargetRevision, SourceRevision, HelmRevision         int
	TimeoutSeconds                                       int
	StartedAt                                            time.Time
}

func (p *OfficialHelmProvider) beginExecution(ctx context.Context, item helmExecutionRecord) (helmExecutionRecord, bool, error) {
	expectedHash := item.RequestHash
	result, err := p.db.ExecContext(ctx, `INSERT IGNORE INTO helm_execution(idempotency_key,request_hash,workflow_run_id,node_key,operation,cluster_id,namespace,release_name,target_revision,source_revision,marker,status,timeout_seconds,started_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,'running',?,CURRENT_TIMESTAMP)`, item.IdempotencyKey, item.RequestHash, item.WorkflowRunID, item.NodeKey, item.Operation, item.ClusterID, item.Namespace, item.ReleaseName, item.TargetRevision, item.SourceRevision, item.Marker, item.TimeoutSeconds)
	if err != nil {
		return item, false, err
	}
	created, _ := result.RowsAffected()
	if err := p.db.QueryRowContext(ctx, `SELECT id,request_hash,status,helm_revision,error_message,started_at FROM helm_execution WHERE idempotency_key=?`, item.IdempotencyKey).Scan(&item.ID, &item.RequestHash, &item.Status, &item.HelmRevision, &item.ErrorMessage, &item.StartedAt); err != nil {
		return item, false, err
	}
	if item.RequestHash != expectedHash {
		return item, false, fmt.Errorf("Helm idempotency key was reused with a different request")
	}
	return item, created == 1, nil
}

func existingHelmExecution(record helmExecutionRecord) (HelmExecution, error) {
	switch record.Status {
	case RunSucceeded:
		return HelmExecution{ExternalID: strconv.FormatInt(record.ID, 10), Completed: true, Revision: record.HelmRevision}, nil
	case RunRunning, RunPending:
		return HelmExecution{ExternalID: strconv.FormatInt(record.ID, 10)}, nil
	case RunFailed:
		return HelmExecution{}, fmt.Errorf("previous Helm execution failed: %s", record.ErrorMessage)
	default:
		return HelmExecution{}, fmt.Errorf("unsupported Helm execution status %q", record.Status)
	}
}

func (p *OfficialHelmProvider) findExecution(ctx context.Context, id int64) (helmExecutionRecord, error) {
	var item helmExecutionRecord
	err := p.db.QueryRowContext(ctx, `SELECT id,idempotency_key,request_hash,workflow_run_id,node_key,operation,cluster_id,namespace,release_name,target_revision,source_revision,marker,status,helm_revision,error_message,timeout_seconds,started_at FROM helm_execution WHERE id=?`, id).Scan(
		&item.ID, &item.IdempotencyKey, &item.RequestHash, &item.WorkflowRunID, &item.NodeKey, &item.Operation, &item.ClusterID, &item.Namespace, &item.ReleaseName, &item.TargetRevision, &item.SourceRevision, &item.Marker, &item.Status, &item.HelmRevision, &item.ErrorMessage, &item.TimeoutSeconds, &item.StartedAt)
	return item, err
}

func (p *OfficialHelmProvider) succeedExecution(ctx context.Context, id int64, revision int) error {
	_, err := p.db.ExecContext(ctx, `UPDATE helm_execution SET status='succeeded',helm_revision=?,error_message='',finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running'`, revision, id)
	return err
}

func (p *OfficialHelmProvider) failExecution(ctx context.Context, id int64, cause error) error {
	message := cause.Error()
	if len(message) > 1024 {
		message = message[:1024]
	}
	_, _ = p.db.ExecContext(ctx, `UPDATE helm_execution SET status='failed',error_message=?,finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running'`, message, id)
	return cause
}

func helmRequestHash(operation string, request HelmRequest, revision int) (string, error) {
	payload, err := json.Marshal(struct {
		Operation, Namespace, ReleaseName string
		ClusterID                         int64
		Chart                             []byte
		Values                            map[string]interface{}
		Revision, Timeout                 int
	}{operation, request.Namespace, request.ReleaseName, request.ClusterID, request.Chart, request.Values, revision, request.TimeoutSeconds})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func helmRollbackHash(request HelmRollbackRequest) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func (p *OfficialHelmProvider) configuration(ctx context.Context, clusterID int64, namespace string) (*action.Configuration, error) {
	config, err := p.factory.RESTConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	getter := &staticHelmRESTClientGetter{config: config, namespace: namespace}
	configuration := new(action.Configuration)
	if err := configuration.Init(getter, namespace, "secret", func(string, ...interface{}) {}); err != nil {
		return nil, fmt.Errorf("initialize Helm action configuration: %w", err)
	}
	// Preserve Helm's optional interfaces: hooks need WaitForDelete and install
	// adoption needs three-way merge. Embedding only kube.Interface loses these.
	client, ok := configuration.KubeClient.(*kube.Client)
	if !ok {
		return nil, fmt.Errorf("unexpected Helm Kubernetes client")
	}
	configuration.KubeClient = helmScopedKubeClient{Interface: client, InterfaceExt: client, InterfaceThreeWayMerge: client, InterfaceLogs: client, InterfaceDeletionPropagation: client, InterfaceResources: client, namespace: namespace}
	return configuration, nil
}

// Helm also builds hook resources outside its post-renderer. Validate every
// Build so upgrade-only hooks and rollback history cannot escape the target.
type helmScopedKubeClient struct {
	kube.Interface
	kube.InterfaceExt
	kube.InterfaceThreeWayMerge
	kube.InterfaceLogs
	kube.InterfaceDeletionPropagation
	kube.InterfaceResources
	namespace string
}

func (c helmScopedKubeClient) Build(reader io.Reader, validate bool) (kube.ResourceList, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, 16<<20+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 16<<20 {
		return nil, fmt.Errorf("Helm rendered manifest exceeds 16 MiB")
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		if _, err := parseHelmManifestResources(raw, c.namespace); err != nil {
			return nil, err
		}
	}
	return c.Interface.Build(bytes.NewReader(raw), validate)
}

type staticHelmRESTClientGetter struct {
	config    *rest.Config
	namespace string
}

func (g *staticHelmRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return rest.CopyConfig(g.config), nil
}
func (g *staticHelmRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	client, err := discovery.NewDiscoveryClientForConfig(g.config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(client), nil
}
func (g *staticHelmRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient), nil
}
func (g *staticHelmRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return staticHelmClientConfig{config: g.config, namespace: g.namespace}
}

type staticHelmClientConfig struct {
	config    *rest.Config
	namespace string
}

func (c staticHelmClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}
func (c staticHelmClientConfig) ClientConfig() (*rest.Config, error) {
	return rest.CopyConfig(c.config), nil
}
func (c staticHelmClientConfig) Namespace() (string, bool, error) { return c.namespace, false, nil }
func (c staticHelmClientConfig) ConfigAccess() clientcmd.ConfigAccess {
	return clientcmd.NewDefaultPathOptions()
}
