// Package imagepull provisions namespace-scoped pull credentials for YAML
// deployments. Secrets are runtime dependencies, not user YAML or build artifacts.
package imagepull

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/yaml"
)

type RESTConfigFactory interface {
	RESTConfig(context.Context, int64) (*rest.Config, error)
}

type Manager struct {
	auth   build.RegistryAuthStore
	client func(context.Context, int64) (kubernetes.Interface, error)
}

func NewManager(auth build.RegistryAuthStore, factory RESTConfigFactory) *Manager {
	return &Manager{auth: auth, client: func(ctx context.Context, clusterID int64) (kubernetes.Interface, error) {
		cfg, err := factory.RESTConfig(ctx, clusterID)
		if err != nil {
			return nil, err
		}
		return kubernetes.NewForConfig(cfg)
	}}
}

func (m *Manager) PrepareYAMLDeployment(ctx context.Context, run workflow.WorkflowRun, config workflow.YAMLDeployConfig, target workflow.YAMLTarget) (workflow.YAMLTarget, error) {
	return m.prepareDeployment(ctx, run, config, target, false)
}

func (m *Manager) PrepareHelmDeployment(ctx context.Context, run workflow.WorkflowRun, config workflow.HelmDeployConfig, target workflow.HelmTarget, manifest []byte) ([]byte, error) {
	result, err := m.prepareDeployment(ctx, run, workflow.YAMLDeployConfig{ServiceID: config.ServiceID}, workflow.YAMLTarget{ClusterID: target.ClusterID, Namespace: target.Namespace, Manifest: manifest}, true)
	if err == nil {
		// Namespace provisioning must also work for public images; Helm cannot
		// persist its Release Secret into a namespace that does not exist yet.
		client, e := m.client(ctx, target.ClusterID)
		if e != nil {
			return nil, e
		}
		if e = ensureDeploymentNamespace(ctx, client, target.Namespace); e != nil {
			return nil, e
		}
	}
	return result.Manifest, err
}

func (m *Manager) prepareDeployment(ctx context.Context, run workflow.WorkflowRun, config workflow.YAMLDeployConfig, target workflow.YAMLTarget, helm bool) (workflow.YAMLTarget, error) {
	if m == nil || m.auth == nil || m.client == nil || run.ProjectID <= 0 || config.ServiceID <= 0 || target.ClusterID <= 0 || target.Namespace == "" {
		return target, fmt.Errorf("image pull credential target is incomplete")
	}
	validate := workflow.ValidateYAMLManifest
	if helm {
		validate = workflow.ValidateHelmManifest
	}
	resources, err := validate(target.Manifest, target.Namespace)
	if err != nil {
		return target, err
	}
	auth, err := m.auth.Load(ctx, run.ProjectID, config.ServiceID)
	if err != nil {
		return target, err
	}
	if auth == nil || (auth.Username == "" && auth.Password == "" && auth.Token == "") {
		return target, nil
	}
	base := strings.TrimSpace(auth.BaseURL)
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}
	registry, err := url.Parse(base)
	if err != nil || registry.Host == "" || registry.User != nil || registry.RawQuery != "" || registry.Fragment != "" || (registry.Scheme != "https" && registry.Scheme != "http") {
		return target, fmt.Errorf("image registry URL is invalid")
	}
	host := strings.ToLower(registry.Host)
	hash := sha256.Sum256([]byte(host))
	name := fmt.Sprintf("axiops-pull-p%d-s%d-%x", run.ProjectID, config.ServiceID, hash[:5])
	var dockerConfig string
	for i := range resources {
		path := podSpecPath(resources[i].GetKind())
		if path == nil {
			continue
		}
		pod, found, err := unstructured.NestedMap(resources[i].Object, path...)
		if err != nil {
			return target, fmt.Errorf("invalid pod template in %s/%s", resources[i].GetKind(), resources[i].GetName())
		}
		if !found {
			continue
		}
		matches := false
		for _, field := range []string{"containers", "initContainers"} {
			containers, _, err := unstructured.NestedSlice(pod, field)
			if err != nil {
				return target, fmt.Errorf("invalid %s in workload", field)
			}
			for _, raw := range containers {
				container, ok := raw.(map[string]interface{})
				if !ok {
					return target, fmt.Errorf("invalid container in workload")
				}
				image, _ := container["image"].(string)
				imageHost, _, explicit := strings.Cut(image, "/")
				if !explicit || !strings.EqualFold(imageHost, host) {
					continue
				}
				// Same host is not enough: the bound repository namespaces must match.
				dockerConfig, err = auth.DockerConfigForImage(image)
				if err != nil {
					return target, err
				}
				if auth.Username == "" || auth.Password == "" {
					return target, fmt.Errorf("Kubernetes image pull requires a registry username and password/access token in the password field")
				}
				matches = true
			}
		}
		if !matches {
			continue
		}
		refs, _, err := unstructured.NestedSlice(pod, "imagePullSecrets")
		if err != nil {
			return target, fmt.Errorf("invalid imagePullSecrets in workload")
		}
		alreadyReferenced := false
		for _, ref := range refs {
			entry, ok := ref.(map[string]interface{})
			if !ok {
				return target, fmt.Errorf("invalid imagePullSecrets entry")
			}
			if entry["name"] == name {
				alreadyReferenced = true
			}
		}
		if !alreadyReferenced {
			refs = append(refs, map[string]interface{}{"name": name})
		}
		if err := unstructured.SetNestedSlice(pod, refs, "imagePullSecrets"); err != nil {
			return target, err
		}
		if err := unstructured.SetNestedMap(resources[i].Object, pod, path...); err != nil {
			return target, err
		}
	}
	if dockerConfig == "" {
		return target, nil
	}
	// Finish validation/serialization before making any cluster changes.
	var manifest bytes.Buffer
	for i, resource := range resources {
		data, err := yaml.Marshal(resource.Object)
		if err != nil {
			return target, err
		}
		if i > 0 {
			manifest.WriteString("---\n")
		}
		manifest.Write(data)
	}
	client, err := m.client(ctx, target.ClusterID)
	if err != nil {
		return target, err
	}
	if err := ensureDeploymentNamespace(ctx, client, target.Namespace); err != nil {
		return target, err
	}
	labels := map[string]string{"app.kubernetes.io/managed-by": "axiops-image-pull", "axiops.io/project-id": strconv.FormatInt(run.ProjectID, 10), "axiops.io/service-id": strconv.FormatInt(config.ServiceID, 10)}
	if err := ensureSecret(ctx, client, target.Namespace, name, labels, []byte(dockerConfig)); err != nil {
		return target, err
	}
	target.Manifest = manifest.Bytes()
	return target, nil
}

func ensureDeploymentNamespace(ctx context.Context, client kubernetes.Interface, namespace string) error {
	if _, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		_, err = client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create deployment namespace: %s", apierrors.ReasonForError(err))
		}
	} else if err != nil {
		return fmt.Errorf("read deployment namespace: %s", apierrors.ReasonForError(err))
	}
	return nil
}

func podSpecPath(kind string) []string {
	switch kind {
	case "Pod":
		return []string{"spec"}
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		return []string{"spec", "template", "spec"}
	case "CronJob":
		return []string{"spec", "jobTemplate", "spec", "template", "spec"}
	default:
		return nil
	}
}

func ensureSecret(ctx context.Context, client kubernetes.Interface, namespace, name string, labels map[string]string, data []byte) error {
	secrets := client.CoreV1().Secrets(namespace)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := secrets.Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = secrets.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels}, Type: corev1.SecretTypeDockerConfigJson, Data: map[string][]byte{corev1.DockerConfigJsonKey: data}}, metav1.CreateOptions{})
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			current, err = secrets.Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			return err
		}
		for key, value := range labels {
			if current.Labels[key] != value {
				return fmt.Errorf("refusing to overwrite an unmanaged image pull Secret")
			}
		}
		if current.Type != corev1.SecretTypeDockerConfigJson {
			return fmt.Errorf("managed image pull Secret has an unexpected type")
		}
		if bytes.Equal(current.Data[corev1.DockerConfigJsonKey], data) {
			return nil
		}
		if current.Immutable != nil && *current.Immutable {
			return fmt.Errorf("managed image pull Secret is immutable")
		}
		updated := current.DeepCopy()
		updated.Data = map[string][]byte{corev1.DockerConfigJsonKey: data}
		_, err = secrets.Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		// Do not return API status details that may echo credential data.
		return fmt.Errorf("cannot provision image pull Secret %s/%s (%s); check namespace Secret permissions, ownership labels and immutability", namespace, name, apierrors.ReasonForError(err))
	}
	return nil
}
