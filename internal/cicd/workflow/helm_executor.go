package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type HelmDeployConfig struct {
	StartOrder          int            `json:"start_order,omitempty"`
	ImageRef            string         `json:"image_ref,omitempty"`
	ServiceID           int64          `json:"service_id"`
	ServiceRevisionID   int64          `json:"service_revision_id"`
	ReleaseName         string         `json:"release_name"`
	FrozenHelm          *HelmSource    `json:"frozen_helm,omitempty"`
	FrozenHelmVariables []HelmVariable `json:"frozen_helm_variables,omitempty"`
}

type HelmTarget struct {
	ImageRef                          string
	ClusterID                         int64
	Namespace, ReleaseName, ChartPath string
	Chart                             []byte
	Values                            map[string]interface{}
	SnapshotValues                    map[string]interface{}
	TimeoutSeconds                    int
}
type HelmExecution struct {
	ExternalID, LogRef string
	Completed          bool
	Revision           int
}
type HelmState struct {
	Status, ErrorMessage string
	Revision             int
}
type HelmRenderResolver interface {
	RenderHelm(context.Context, WorkflowRun, HelmDeployConfig) (HelmTarget, error)
}
type HelmProvider interface {
	Render(context.Context, HelmRequest) ([]byte, error)
	UpgradeInstall(context.Context, HelmRequest) (HelmExecution, error)
	Status(context.Context, string) (HelmState, error)
	Rollback(context.Context, HelmRollbackRequest) (HelmExecution, error)
}
type HelmRequest struct {
	PrepareManifest                   func([]byte) ([]byte, error) `json:"-"`
	RunID                             int64
	NodeKey                           string
	ClusterID                         int64
	Namespace, ReleaseName, ChartPath string
	Chart                             []byte
	Values                            map[string]interface{}
	Atomic, Wait                      bool
	TimeoutSeconds                    int
	IdempotencyKey                    string
}
type HelmRollbackRequest struct {
	ExpectedCurrentRevision int    `json:"expected_current_revision,omitempty"`
	ExpectedInventoryDigest string `json:"expected_inventory_digest,omitempty"`
	RunID, ClusterID        int64
	NodeKey                 string
	Namespace, ReleaseName  string
	Revision                int
	Atomic, Wait            bool
	TimeoutSeconds          int
	IdempotencyKey          string
}

type HelmDeployExecutor struct {
	resolver  HelmRenderResolver
	provider  HelmProvider
	snapshots HelmDeploymentSnapshotStore
	preparer  HelmDeploymentPreparer
}

type HelmDeploymentPreparer interface {
	PrepareHelmDeployment(context.Context, WorkflowRun, HelmDeployConfig, HelmTarget, []byte) ([]byte, error)
}

func (e *HelmDeployExecutor) WithDeploymentPreparer(p HelmDeploymentPreparer) *HelmDeployExecutor {
	e.preparer = p
	return e
}

func NewHelmDeployExecutor(resolver HelmRenderResolver, provider HelmProvider, snapshots HelmDeploymentSnapshotStore) *HelmDeployExecutor {
	return &HelmDeployExecutor{resolver: resolver, provider: provider, snapshots: snapshots}
}
func (e *HelmDeployExecutor) NodeType() string { return NodeHelmDeploy }
func (e *HelmDeployExecutor) Execute(ctx context.Context, request ExecutionRequest) (result ExecutionResult, err error) {
	progress := newHelmTaskProgress(request.ReportProgress)
	defer func() { result.LogText = progress.text(result.LogText) }()
	if e == nil || e.resolver == nil || e.provider == nil || e.snapshots == nil {
		return ExecutionResult{}, fmt.Errorf("helm deploy executor is not configured")
	}
	config, err := helmDeployConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	if err := progress.report("[prepare] 正在读取目标环境的 Helm Chart 和 values"); err != nil {
		return ExecutionResult{}, err
	}
	target, err := e.resolver.RenderHelm(ctx, request.Run, config)
	if err != nil {
		return ExecutionResult{}, err
	}
	if validator, ok := e.snapshots.(interface {
		ValidateHelmProductionExecution(context.Context, WorkflowRun, TaskRun, HelmDeployConfig, HelmTarget) error
	}); ok {
		if err := validator.ValidateHelmProductionExecution(ctx, request.Run, request.Task, config, target); err != nil {
			return ExecutionResult{}, err
		}
	}
	if target.ClusterID <= 0 || target.Namespace == "" || target.ReleaseName == "" || target.ChartPath == "" {
		return ExecutionResult{}, fmt.Errorf("helm target is incomplete")
	}
	timeout := target.TimeoutSeconds
	if timeout <= 0 {
		timeout = 600
	}
	if err := progress.report(fmt.Sprintf("[target] 集群 #%d · 命名空间 %s · Helm Release %s", target.ClusterID, target.Namespace, target.ReleaseName)); err != nil {
		return ExecutionResult{}, err
	}
	providerRequest := HelmRequest{RunID: request.Run.ID, NodeKey: request.Task.NodeKey, ClusterID: target.ClusterID, Namespace: target.Namespace, ReleaseName: target.ReleaseName, ChartPath: target.ChartPath, Chart: target.Chart, Values: target.Values, Atomic: true, Wait: true, TimeoutSeconds: timeout, IdempotencyKey: fmt.Sprintf("workflow:%d:%s", request.Run.ID, request.Task.NodeKey)}
	manifest, err := e.provider.Render(ctx, providerRequest)
	if err != nil {
		return ExecutionResult{}, err
	}
	if err := validateHelmBuiltImage(manifest, target.Namespace, target.ImageRef); err != nil {
		return ExecutionResult{}, err
	}
	providerRequest.Atomic, err = helmAtomicPolicy(manifest, nil, target.Namespace)
	if err != nil {
		return ExecutionResult{}, err
	}
	snapshot, err := NewHelmDeploymentSnapshot(request.Run, request.Task, config, target, manifest)
	if err != nil {
		return ExecutionResult{}, err
	}
	if err := e.snapshots.Prepare(ctx, snapshot); err != nil {
		return ExecutionResult{}, err
	}
	policy := "常驻资源按 Helm 原子策略处理失败，以最终执行结果为准"
	if !providerRequest.Atomic {
		policy = "包含一次性 Job：等待任务完成；失败保留诊断资源，不自动回滚重放旧任务"
	}
	if err := progress.report(fmt.Sprintf("[prepare] Chart 渲染及资源范围校验通过，部署快照已保存\n[waiting] 正在安装或升级 Helm Release；等待资源就绪，超时 %d 秒；%s", timeout, policy)); err != nil {
		return ExecutionResult{}, err
	}
	providerRequest.PrepareManifest = func(manifest []byte) ([]byte, error) {
		if err := validateHelmBuiltImage(manifest, target.Namespace, target.ImageRef); err != nil {
			return nil, err
		}
		if e.preparer != nil {
			var err error
			manifest, err = e.preparer.PrepareHelmDeployment(ctx, request.Run, config, target, manifest)
			if err != nil {
				return nil, err
			}
		}
		// .Release.IsUpgrade may render different objects than the preliminary
		// install preview. Persist the actual post-rendered inventory before apply.
		actual, err := NewHelmDeploymentSnapshot(request.Run, request.Task, config, target, manifest)
		if err != nil {
			return nil, err
		}
		if err = e.snapshots.Prepare(ctx, actual); err != nil {
			return nil, err
		}
		return manifest, nil
	}
	execution, err := e.provider.UpgradeInstall(ctx, providerRequest)
	if err != nil {
		return ExecutionResult{}, err
	}
	if execution.ExternalID == "" {
		return ExecutionResult{}, fmt.Errorf("helm provider returned empty external id")
	}
	if execution.Completed {
		if execution.Revision <= 0 {
			return ExecutionResult{}, fmt.Errorf("helm provider returned an invalid release revision")
		}
		if err := e.snapshots.MarkHelmApplied(ctx, request.Run.ID, request.Task.NodeKey, target.ReleaseName, execution.Revision); err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{Status: RunSucceeded, ProviderRunID: "helm:" + execution.ExternalID, LogRef: execution.LogRef, LogText: fmt.Sprintf("[success] Helm Release %s 已就绪 · revision %d", target.ReleaseName, execution.Revision)}, nil
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: "helm:" + execution.ExternalID, LogRef: execution.LogRef, LogText: "[recover] 已找到原 Helm 执行记录，继续核对结果，不重复安装或升级"}, nil
}
func (e *HelmDeployExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	id, ok := strings.CutPrefix(request.Task.ProviderRunID, "helm:")
	if !ok || id == "" {
		return ExecutionResult{}, fmt.Errorf("invalid helm provider run id")
	}
	state, err := e.provider.Status(ctx, id)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case RunSucceeded:
		config, err := helmDeployConfig(request.Task.NodeSnapshot)
		if err != nil {
			return ExecutionResult{}, err
		}
		if state.Revision <= 0 {
			return ExecutionResult{}, fmt.Errorf("helm provider returned an invalid release revision")
		}
		if err := e.snapshots.MarkHelmApplied(ctx, request.Run.ID, request.Task.NodeKey, config.ReleaseName, state.Revision); err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{Status: RunSucceeded, LogText: fmt.Sprintf("[success] Helm Release %s 已就绪 · revision %d", config.ReleaseName, state.Revision)}, nil
	case RunFailed:
		return ExecutionResult{Status: RunFailed, ErrorMessage: state.ErrorMessage, LogText: "[failed] Helm 部署失败：" + safeDiagnostic(state.ErrorMessage)}, nil
	case RunCancelled:
		return ExecutionResult{Status: RunCancelled, LogText: "[cancel] Helm 执行已取消，请核对环境中的实际资源状态"}, nil
	case RunPending, RunRunning:
		return ExecutionResult{Status: RunRunning}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported helm provider status %q", state.Status)
	}
}
func helmDeployConfig(snapshot json.RawMessage) (HelmDeployConfig, error) {
	var node Node
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &node) != nil {
		return HelmDeployConfig{}, fmt.Errorf("invalid helm deploy node snapshot")
	}
	var config HelmDeployConfig
	if len(node.Config) == 0 || json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 || config.ServiceRevisionID <= 0 || strings.TrimSpace(config.ReleaseName) == "" {
		return HelmDeployConfig{}, fmt.Errorf("helm deploy requires service_id, service_revision_id, and release_name")
	}
	return config, nil
}

// MergeHelmValues implements the documented precedence without mutating inputs.
func MergeHelmValues(layers ...map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, layer := range layers {
		mergeHelmMap(result, layer)
	}
	return result
}
func mergeHelmMap(dst, src map[string]interface{}) {
	for key, value := range src {
		incoming, ok := value.(map[string]interface{})
		if !ok {
			dst[key] = deepCopyHelmValue(value)
			continue
		}
		current, ok := dst[key].(map[string]interface{})
		if !ok {
			current = map[string]interface{}{}
			dst[key] = current
		}
		mergeHelmMap(current, incoming)
	}
}
func deepCopyHelmValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		copied := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			copied[key] = deepCopyHelmValue(item)
		}
		return copied
	case []interface{}:
		copied := make([]interface{}, len(typed))
		for i, item := range typed {
			copied[i] = deepCopyHelmValue(item)
		}
		return copied
	default:
		return value
	}
}
