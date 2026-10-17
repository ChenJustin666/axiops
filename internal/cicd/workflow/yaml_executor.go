package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

type YAMLDeployConfig struct {
	StartOrder        int         `json:"start_order,omitempty"`
	ServiceID         int64       `json:"service_id"`
	ServiceRevisionID int64       `json:"service_revision_id"`
	FrozenYAML        *YAMLSource `json:"frozen_yaml,omitempty"`
}

type YAMLTarget struct {
	ClusterID    int64
	Namespace    string
	Manifest     []byte
	AssetSecrets []serviceassets.SecretRef
}

type YAMLRenderResolver interface {
	RenderYAML(context.Context, WorkflowRun, YAMLDeployConfig) (YAMLTarget, error)
}

// YAMLDeploymentPreparer provisions runtime dependencies and returns a manifest
// containing references only. Credentials must never enter deployment snapshots.
type YAMLDeploymentPreparer interface {
	PrepareYAMLDeployment(context.Context, WorkflowRun, YAMLDeployConfig, YAMLTarget) (YAMLTarget, error)
}

type DeploymentPreparers []YAMLDeploymentPreparer

func (p DeploymentPreparers) PrepareYAMLDeployment(ctx context.Context, run WorkflowRun, cfg YAMLDeployConfig, target YAMLTarget) (YAMLTarget, error) {
	var err error
	for _, item := range p {
		target, err = item.PrepareYAMLDeployment(ctx, run, cfg, target)
		if err != nil {
			return target, err
		}
	}
	return target, nil
}

type YAMLProviderRequest struct {
	RunID          int64
	NodeKey        string
	ClusterID      int64
	Namespace      string
	Manifest       []byte
	IdempotencyKey string
	ReportProgress func(string) error
}

type YAMLProviderExecution struct {
	ExternalID string
	LogRef     string
	LogText    string
	Completed  bool
}

type YAMLProviderState struct {
	Status       string
	ErrorMessage string
	LogText      string
}

type YAMLDeployProvider interface {
	Apply(context.Context, YAMLProviderRequest) (YAMLProviderExecution, error)
	Status(context.Context, string) (YAMLProviderState, error)
}

type YAMLDeployExecutor struct {
	resolver  YAMLRenderResolver
	provider  YAMLDeployProvider
	snapshots DeploymentSnapshotStore
	preparer  YAMLDeploymentPreparer
}

func (e *YAMLDeployExecutor) WithDeploymentPreparer(preparer YAMLDeploymentPreparer) *YAMLDeployExecutor {
	e.preparer = preparer
	return e
}

func NewYAMLDeployExecutor(resolver YAMLRenderResolver, provider YAMLDeployProvider, snapshots DeploymentSnapshotStore) *YAMLDeployExecutor {
	return &YAMLDeployExecutor{resolver: resolver, provider: provider, snapshots: snapshots}
}

func (e *YAMLDeployExecutor) NodeType() string { return NodeYAMLDeploy }

func (e *YAMLDeployExecutor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	if e == nil || e.resolver == nil || e.provider == nil || e.snapshots == nil {
		return ExecutionResult{}, fmt.Errorf("yaml deploy executor is not configured")
	}
	config, err := yamlDeployConfig(request.Task.NodeSnapshot)
	if err != nil {
		return ExecutionResult{}, err
	}
	if validator, ok := e.snapshots.(interface {
		ValidateProductionExecution(context.Context, WorkflowRun, TaskRun) error
	}); ok {
		if err := validator.ValidateProductionExecution(ctx, request.Run, request.Task); err != nil {
			return ExecutionResult{}, err
		}
	}
	progress := func(message string) error {
		if request.ReportProgress != nil {
			return request.ReportProgress(message)
		}
		return nil
	}
	if err := progress("[prepare] 正在渲染并校验部署配置"); err != nil {
		return ExecutionResult{}, err
	}
	target, err := e.resolver.RenderYAML(ctx, request.Run, config)
	if err != nil {
		return ExecutionResult{}, err
	}
	if target.ClusterID <= 0 || strings.TrimSpace(target.Namespace) == "" {
		return ExecutionResult{}, fmt.Errorf("yaml target cluster and namespace are required")
	}
	if _, err := ValidateYAMLManifest(target.Manifest, target.Namespace); err != nil {
		return ExecutionResult{}, err
	}
	if e.preparer != nil {
		if err := progress("[prepare] 配置校验通过，正在准备配套资源和镜像拉取凭据"); err != nil {
			return ExecutionResult{}, err
		}
		target, err = e.preparer.PrepareYAMLDeployment(ctx, request.Run, config, target)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("prepare deployment dependencies: %w", err)
		}
	}
	snapshot, err := NewYAMLDeploymentSnapshot(request.Run, request.Task, config, target)
	if err != nil {
		return ExecutionResult{}, err
	}
	if err := e.snapshots.Prepare(ctx, snapshot); err != nil {
		return ExecutionResult{}, err
	}
	if err := progress("[prepare] YAML rendered and validated; deployment snapshot saved"); err != nil {
		return ExecutionResult{}, err
	}
	// The key is stable across worker retries, so the provider must return the
	// same external operation instead of applying the manifest twice.
	execution, err := e.provider.Apply(ctx, YAMLProviderRequest{
		RunID: request.Run.ID, NodeKey: request.Task.NodeKey, ClusterID: target.ClusterID,
		Namespace: target.Namespace, Manifest: target.Manifest,
		IdempotencyKey: fmt.Sprintf("workflow:%d:%s", request.Run.ID, request.Task.NodeKey),
		ReportProgress: request.ReportProgress,
	})
	if request.ReportProgress == nil {
		execution.LogText = "[prepare] YAML rendered and validated; deployment snapshot saved\n" + execution.LogText
	}
	if err != nil {
		return ExecutionResult{LogText: execution.LogText}, err
	}
	if execution.ExternalID == "" {
		return ExecutionResult{}, fmt.Errorf("yaml provider returned empty external id")
	}
	if execution.Completed {
		if err := e.snapshots.MarkApplied(ctx, request.Run.ID, request.Task.NodeKey, NodeYAMLDeploy); err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{Status: RunSucceeded, ProviderRunID: "yaml:" + execution.ExternalID, LogRef: execution.LogRef, LogText: execution.LogText}, nil
	}
	return ExecutionResult{Status: RunRunning, ProviderRunID: "yaml:" + execution.ExternalID, LogRef: execution.LogRef, LogText: execution.LogText}, nil
}

func (e *YAMLDeployExecutor) Reconcile(ctx context.Context, request ExecutionRequest) (ExecutionResult, error) {
	externalID, ok := strings.CutPrefix(request.Task.ProviderRunID, "yaml:")
	if !ok || externalID == "" {
		return ExecutionResult{}, fmt.Errorf("invalid yaml provider run id %q", request.Task.ProviderRunID)
	}
	state, err := e.provider.Status(ctx, externalID)
	if err != nil {
		return ExecutionResult{}, err
	}
	switch state.Status {
	case RunSucceeded:
		if err := e.snapshots.MarkApplied(ctx, request.Run.ID, request.Task.NodeKey, NodeYAMLDeploy); err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{Status: RunSucceeded, LogText: state.LogText}, nil
	case RunFailed:
		return ExecutionResult{Status: RunFailed, ErrorMessage: state.ErrorMessage, LogText: state.LogText}, nil
	case RunCancelled:
		return ExecutionResult{Status: RunCancelled}, nil
	case RunPending, RunRunning:
		return ExecutionResult{Status: RunRunning, LogText: state.LogText}, nil
	default:
		return ExecutionResult{}, fmt.Errorf("unsupported yaml provider status %q", state.Status)
	}
}

func yamlDeployConfig(snapshot json.RawMessage) (YAMLDeployConfig, error) {
	var node Node
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &node) != nil {
		return YAMLDeployConfig{}, fmt.Errorf("invalid yaml deploy node snapshot")
	}
	var config YAMLDeployConfig
	if len(node.Config) == 0 || json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 || config.ServiceRevisionID <= 0 {
		return YAMLDeployConfig{}, fmt.Errorf("yaml deploy node requires service_id and service_revision_id")
	}
	return config, nil
}

var allowedYAMLKinds = map[string]struct{}{
	"ConfigMap": {}, "ServiceAccount": {}, "Role": {}, "RoleBinding": {},
	"Service": {}, "Deployment": {}, "StatefulSet": {}, "DaemonSet": {},
	"Job": {}, "CronJob": {}, "Ingress": {}, "PodDisruptionBudget": {},
	"HorizontalPodAutoscaler": {}, "NetworkPolicy": {}, "PersistentVolumeClaim": {},
}

func ValidateYAMLManifest(manifest []byte, namespace string) ([]unstructured.Unstructured, error) {
	resources, err := ParseYAMLManifest(manifest)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for i := range resources {
		resourceNamespace := resources[i].GetNamespace()
		if resourceNamespace != "" && resourceNamespace != namespace {
			return nil, fmt.Errorf("resource %s/%s targets namespace %q instead of %q", resources[i].GetKind(), resources[i].GetName(), resourceNamespace, namespace)
		}
		resources[i].SetNamespace(namespace)
		identity := resources[i].GroupVersionKind().GroupKind().String() + "/" + namespace + "/" + resources[i].GetName()
		if seen[identity] {
			return nil, fmt.Errorf("duplicate resource %s/%s in namespace %s", resources[i].GetKind(), resources[i].GetName(), namespace)
		}
		seen[identity] = true
	}
	return resources, nil
}

// ParseYAMLManifest validates repository YAML before an environment namespace
// has been selected. Environment-specific namespace enforcement happens at deploy time.
func ParseYAMLManifest(manifest []byte) ([]unstructured.Unstructured, error) {
	if len(bytes.TrimSpace(manifest)) == 0 {
		return nil, fmt.Errorf("rendered yaml manifest is empty")
	}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	var resources []unstructured.Unstructured
	seen := map[string]bool{}
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode yaml manifest: %w", err)
		}
		if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
			continue
		}
		var resource unstructured.Unstructured
		if err := json.Unmarshal(raw, &resource.Object); err != nil {
			return nil, fmt.Errorf("decode kubernetes resource: %w", err)
		}
		if resource.GetAPIVersion() == "" || resource.GetKind() == "" || resource.GetName() == "" {
			return nil, fmt.Errorf("resource apiVersion, kind, and metadata.name are required")
		}
		if resource.GetKind() == "Secret" {
			return nil, fmt.Errorf("Secret resources must use the credential injection path")
		}
		if _, allowed := allowedYAMLKinds[resource.GetKind()]; !allowed {
			return nil, fmt.Errorf("resource kind %q is not allowed for project deployment", resource.GetKind())
		}
		identity := resource.GetAPIVersion() + "/" + resource.GetKind() + "/" + resource.GetNamespace() + "/" + resource.GetName()
		if seen[identity] {
			return nil, fmt.Errorf("duplicate resource %s/%s", resource.GetKind(), resource.GetName())
		}
		seen[identity] = true
		if err := validateContainerEnvironment(resource); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("rendered yaml manifest contains no resources")
	}
	return resources, nil
}
