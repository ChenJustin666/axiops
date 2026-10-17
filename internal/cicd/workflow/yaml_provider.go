package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type KubernetesResourceApplier interface {
	Apply(context.Context, string, unstructured.Unstructured, string, bool) (unstructured.Unstructured, error)
	Preview(context.Context, string, unstructured.Unstructured, string) (string, error)
}

// NamespaceEnsurer is implemented by the real Kubernetes client. Test and
// preview clients may omit it; deployment still works against pre-created
// namespaces in that case.
type NamespaceEnsurer interface {
	EnsureNamespace(context.Context, string) error
}

type KubernetesClientFactory interface {
	Client(context.Context, int64) (KubernetesResourceApplier, error)
}

type YAMLResourceChange struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Action     string `json:"action"`
}

type KubernetesYAMLProvider struct {
	targets      YAMLHealthTargetStore
	factory      KubernetesClientFactory
	fieldManager string
	force        bool
	mu           sync.RWMutex
	completed    map[string]YAMLProviderState
}

func (p *KubernetesYAMLProvider) WithHealthTargets(store YAMLHealthTargetStore) *KubernetesYAMLProvider {
	p.targets = store
	return p
}

func NewKubernetesYAMLProvider(factory KubernetesClientFactory, fieldManager string, force bool) *KubernetesYAMLProvider {
	if fieldManager == "" {
		fieldManager = "axiops-cicd"
	}
	return &KubernetesYAMLProvider{factory: factory, fieldManager: fieldManager, force: force, completed: make(map[string]YAMLProviderState)}
}

func (p *KubernetesYAMLProvider) Apply(ctx context.Context, request YAMLProviderRequest) (YAMLProviderExecution, error) {
	if p == nil || p.factory == nil {
		return YAMLProviderExecution{}, fmt.Errorf("kubernetes client factory is not configured")
	}
	resources, err := ValidateYAMLManifest(request.Manifest, request.Namespace)
	if err != nil {
		return YAMLProviderExecution{}, err
	}
	lines := []string{}
	progress := func(message string) error {
		if request.ReportProgress != nil {
			return request.ReportProgress(message)
		}
		lines = append(lines, message)
		return nil
	}
	if err := progress(fmt.Sprintf("[apply] cluster=%d namespace=%s; manifest validated", request.ClusterID, request.Namespace)); err != nil {
		return YAMLProviderExecution{}, err
	}
	client, err := p.factory.Client(ctx, request.ClusterID)
	if err != nil {
		return YAMLProviderExecution{}, err
	}
	if ensurer, ok := client.(NamespaceEnsurer); ok {
		if err := ensurer.EnsureNamespace(ctx, request.Namespace); err != nil {
			return YAMLProviderExecution{}, fmt.Errorf("ensure namespace %q: %w", request.Namespace, err)
		}
	}
	for _, resource := range resources {
		if _, err := client.Apply(ctx, request.Namespace, resource, p.fieldManager, p.force); err != nil {
			return YAMLProviderExecution{LogText: strings.Join(lines, "\n")}, fmt.Errorf("apply %s/%s: %s", resource.GetKind(), resource.GetName(), safeDiagnostic(err.Error()))
		}
		if err := progress(fmt.Sprintf("[apply] %s/%s submitted", resource.GetKind(), resource.GetName())); err != nil {
			return YAMLProviderExecution{}, err
		}
	}
	if p.targets != nil {
		if err := progress("[apply] resources submitted; waiting for Kubernetes rollout checks"); err != nil {
			return YAMLProviderExecution{}, err
		}
		return YAMLProviderExecution{ExternalID: request.IdempotencyKey, Completed: false, LogText: strings.Join(lines, "\n")}, nil
	}
	p.mu.Lock()
	p.completed[request.IdempotencyKey] = YAMLProviderState{Status: RunSucceeded}
	p.mu.Unlock()
	if err := progress("[apply] resources submitted; readiness checks are not configured"); err != nil {
		return YAMLProviderExecution{}, err
	}
	return YAMLProviderExecution{ExternalID: request.IdempotencyKey, Completed: true, LogText: strings.Join(lines, "\n")}, nil
}

func (p *KubernetesYAMLProvider) Status(ctx context.Context, externalID string) (YAMLProviderState, error) {
	if p.targets != nil {
		target, err := p.targets.HealthTarget(ctx, externalID)
		if err != nil {
			return YAMLProviderState{}, err
		}
		return p.checkTarget(ctx, target)
	}
	p.mu.RLock()
	state, ok := p.completed[externalID]
	p.mu.RUnlock()
	if !ok {
		return YAMLProviderState{Status: RunRunning}, nil
	}
	return state, nil
}

func (p *KubernetesYAMLProvider) Preview(ctx context.Context, request YAMLProviderRequest) ([]YAMLResourceChange, error) {
	if p == nil || p.factory == nil {
		return nil, fmt.Errorf("kubernetes client factory is not configured")
	}
	resources, err := ValidateYAMLManifest(request.Manifest, request.Namespace)
	if err != nil {
		return nil, err
	}
	return p.previewResources(ctx, request.ClusterID, request.Namespace, resources)
}

// PreviewHelm performs server-side dry-run for the complete rendered Chart.
// Callers only receive resource identities and actions; Secret bodies never
// leave this package.
func (p *KubernetesYAMLProvider) PreviewHelm(ctx context.Context, clusterID int64, namespace string, manifest []byte) ([]YAMLResourceChange, error) {
	if p == nil || p.factory == nil {
		return nil, fmt.Errorf("kubernetes client factory is not configured")
	}
	resources, err := parseHelmManifestResources(manifest, namespace)
	if err != nil {
		return nil, err
	}
	return p.previewResources(ctx, clusterID, namespace, resources)
}

func (p *KubernetesYAMLProvider) previewResources(ctx context.Context, clusterID int64, namespace string, resources []unstructured.Unstructured) ([]YAMLResourceChange, error) {
	client, err := p.factory.Client(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	changes := make([]YAMLResourceChange, 0, len(resources))
	for _, resource := range resources {
		action, err := client.Preview(ctx, namespace, resource, p.fieldManager)
		if err != nil {
			return nil, fmt.Errorf("preview %s/%s: %w", resource.GetKind(), resource.GetName(), err)
		}
		changes = append(changes, YAMLResourceChange{
			APIVersion: resource.GetAPIVersion(), Kind: resource.GetKind(), Namespace: namespace,
			Name: resource.GetName(), Action: action,
		})
	}
	return changes, nil
}
