package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterConnection struct {
	EncryptedKubeconfig string
	Status              string
}

type ClusterConnectionStore interface {
	LoadClusterConnection(context.Context, int64) (*ClusterConnection, error)
}

type CredentialDecryptor interface {
	Decrypt(string) (string, error)
}

type SQLClusterConnectionStore struct{ db *sql.DB }

func NewSQLClusterConnectionStore(db *sql.DB) *SQLClusterConnectionStore {
	return &SQLClusterConnectionStore{db: db}
}

func (s *SQLClusterConnectionStore) LoadClusterConnection(ctx context.Context, clusterID int64) (*ClusterConnection, error) {
	if s == nil || s.db == nil || clusterID <= 0 {
		return nil, fmt.Errorf("cluster connection database and id are required")
	}
	var connection ClusterConnection
	err := s.db.QueryRowContext(ctx, `SELECT kubeconfig, status FROM cluster WHERE id=?`, clusterID).
		Scan(&connection.EncryptedKubeconfig, &connection.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &connection, err
}

type KubernetesFactoryOptions struct {
	QPS      float32
	Burst    int
	Timeout  time.Duration
	CacheTTL time.Duration
}

type cachedKubernetesClient struct {
	client    KubernetesResourceApplier
	expiresAt time.Time
}

type SQLKubernetesClientFactory struct {
	store     ClusterConnectionStore
	decryptor CredentialDecryptor
	options   KubernetesFactoryOptions
	now       func() time.Time
	mu        sync.Mutex
	clients   map[int64]cachedKubernetesClient
}

func NewSQLKubernetesClientFactory(store ClusterConnectionStore, decryptor CredentialDecryptor, options KubernetesFactoryOptions) *SQLKubernetesClientFactory {
	if options.QPS <= 0 {
		options.QPS = 20
	}
	if options.Burst <= 0 {
		options.Burst = 40
	}
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	if options.CacheTTL <= 0 {
		options.CacheTTL = time.Minute
	}
	return &SQLKubernetesClientFactory{store: store, decryptor: decryptor, options: options, now: time.Now, clients: make(map[int64]cachedKubernetesClient)}
}

func (f *SQLKubernetesClientFactory) Client(ctx context.Context, clusterID int64) (KubernetesResourceApplier, error) {
	if f == nil || f.store == nil || f.decryptor == nil || clusterID <= 0 {
		return nil, fmt.Errorf("cluster client factory is not configured")
	}
	f.mu.Lock()
	if cached, ok := f.clients[clusterID]; ok && f.now().Before(cached.expiresAt) {
		f.mu.Unlock()
		return cached.client, nil
	}
	f.mu.Unlock()

	restConfig, err := f.RESTConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create cluster %d dynamic client: %w", clusterID, err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create cluster %d discovery client: %w", clusterID, err)
	}
	client := &dynamicResourceApplier{
		client: dynamicClient,
		mapper: restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient)),
	}
	f.mu.Lock()
	f.clients[clusterID] = cachedKubernetesClient{client: client, expiresAt: f.now().Add(f.options.CacheTTL)}
	f.mu.Unlock()
	return client, nil
}

// RESTConfig returns a copy used by providers that need typed clients, such
// as the Kubernetes Job build provider. Credentials remain memory-only.
func (f *SQLKubernetesClientFactory) RESTConfig(ctx context.Context, clusterID int64) (*rest.Config, error) {
	if f == nil || f.store == nil || f.decryptor == nil || clusterID <= 0 {
		return nil, fmt.Errorf("cluster client factory is not configured")
	}
	connection, err := f.store.LoadClusterConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if connection == nil {
		return nil, fmt.Errorf("cluster %d was not found", clusterID)
	}
	if connection.Status != "connected" {
		return nil, fmt.Errorf("cluster %d is not connected", clusterID)
	}
	if !strings.HasPrefix(connection.EncryptedKubeconfig, "v1:") {
		return nil, fmt.Errorf("cluster %d kubeconfig is not encrypted", clusterID)
	}
	kubeconfig, err := f.decryptor.Decrypt(connection.EncryptedKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("decrypt cluster %d kubeconfig: credential authentication failed", clusterID)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("parse cluster %d kubeconfig: %w", clusterID, err)
	}
	restConfig.QPS = f.options.QPS
	restConfig.Burst = f.options.Burst
	restConfig.Timeout = f.options.Timeout
	return rest.CopyConfig(restConfig), nil
}

type dynamicResourceApplier struct {
	client dynamic.Interface
	mapper meta.RESTMapper
}

func (a *dynamicResourceApplier) EnsureNamespace(ctx context.Context, namespace string) error {
	namespace = strings.TrimSpace(namespace)
	if a == nil || a.client == nil || a.mapper == nil || namespace == "" {
		return fmt.Errorf("kubernetes client and namespace are required")
	}
	resourceClient, err := a.resourceClient(schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}, "")
	if err != nil {
		return err
	}
	obj := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1", "kind": "Namespace",
		"metadata": map[string]interface{}{"name": namespace},
	}}
	content, err := json.Marshal(obj.Object)
	if err != nil {
		return err
	}
	_, err = resourceClient.Patch(ctx, namespace, types.ApplyPatchType, content, metav1.PatchOptions{FieldManager: "axiops-cicd", Force: func() *bool { value := true; return &value }()})
	return err
}

func (a *dynamicResourceApplier) Apply(ctx context.Context, namespace string, resource unstructured.Unstructured, fieldManager string, force bool) (unstructured.Unstructured, error) {
	resourceClient, err := a.resourceClient(resource.GroupVersionKind(), namespace)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	content, err := json.Marshal(resource.Object)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	applied, err := resourceClient.Patch(ctx, resource.GetName(), types.ApplyPatchType, content, metav1.PatchOptions{FieldManager: fieldManager, Force: &force})
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return *applied, nil
}

func (a *dynamicResourceApplier) Preview(ctx context.Context, namespace string, resource unstructured.Unstructured, fieldManager string) (string, error) {
	resourceClient, err := a.resourceClient(resource.GroupVersionKind(), namespace)
	if err != nil {
		return "", err
	}
	action := "update"
	live, err := resourceClient.Get(ctx, resource.GetName(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		action = "create"
	} else if err != nil {
		return "", err
	}
	content, err := json.Marshal(resource.Object)
	if err != nil {
		return "", err
	}
	previewed, err := resourceClient.Patch(ctx, resource.GetName(), types.ApplyPatchType, content, metav1.PatchOptions{FieldManager: fieldManager, DryRun: []string{metav1.DryRunAll}})
	if err != nil {
		return "", err
	}
	if action == "update" && apiequality.Semantic.DeepEqual(comparableResource(live), comparableResource(previewed)) {
		action = "unchanged"
	}
	return action, nil
}

func comparableResource(resource *unstructured.Unstructured) map[string]interface{} {
	if resource == nil {
		return nil
	}
	copy := resource.DeepCopy()
	unstructured.RemoveNestedField(copy.Object, "status")
	for _, field := range []string{"creationTimestamp", "deletionGracePeriodSeconds", "deletionTimestamp", "generation", "managedFields", "resourceVersion", "selfLink", "uid"} {
		unstructured.RemoveNestedField(copy.Object, "metadata", field)
	}
	return copy.Object
}

func (a *dynamicResourceApplier) resourceClient(gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	if a == nil || a.client == nil || a.mapper == nil {
		return nil, fmt.Errorf("dynamic kubernetes client is not configured")
	}
	mapping, err := a.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	resource := a.client.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if strings.TrimSpace(namespace) == "" {
			return nil, fmt.Errorf("namespace is required for %s", gvk.Kind)
		}
		return resource.Namespace(namespace), nil
	}
	return resource, nil
}
