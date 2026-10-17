package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func loadImmutableHelmChart(definitionBytes []byte) (*chart.Chart, error) {
	return helmconfig.LoadChart(definitionBytes)
}

func renderHelmManifest(ctx context.Context, request HelmRequest, hideSecret bool) ([]byte, error) {
	loaded, err := loadImmutableHelmChart(request.Chart)
	if err != nil {
		return nil, err
	}
	configuration := new(action.Configuration)
	install := action.NewInstall(configuration)
	install.ClientOnly = true
	install.DryRun = true
	install.Replace = true
	install.ReleaseName = request.ReleaseName
	install.Namespace = request.Namespace
	install.SkipCRDs = true
	install.IncludeCRDs = false
	install.HideSecret = hideSecret
	release, err := install.RunWithContext(ctx, loaded, request.Values)
	if err != nil {
		return nil, fmt.Errorf("render Helm Chart: %w", err)
	}
	if strings.TrimSpace(release.Manifest) == "" {
		return nil, fmt.Errorf("rendered Helm manifest is empty")
	}
	for _, hook := range release.Hooks {
		if _, err := parseHelmManifestResources([]byte(hook.Manifest), request.Namespace); err != nil {
			return nil, fmt.Errorf("Helm hook 资源校验失败: %w", err)
		}
	}
	return []byte(release.Manifest), nil
}

func ValidateHelmManifest(manifest []byte, namespace string) ([]unstructured.Unstructured, error) {
	return parseHelmManifestResources(manifest, namespace)
}

type helmResourceIdentity struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
}

func helmManifestInventory(manifest []byte, namespace string) ([]byte, error) {
	resources, err := parseHelmManifestResources(manifest, namespace)
	if err != nil {
		return nil, err
	}
	inventory := make([]helmResourceIdentity, 0, len(resources))
	for _, resource := range resources {
		inventory = append(inventory, helmResourceIdentity{
			APIVersion: resource.GetAPIVersion(), Kind: resource.GetKind(),
			Namespace: resource.GetNamespace(), Name: resource.GetName(),
		})
	}
	sort.Slice(inventory, func(i, j int) bool {
		left := inventory[i].APIVersion + "/" + inventory[i].Kind + "/" + inventory[i].Namespace + "/" + inventory[i].Name
		right := inventory[j].APIVersion + "/" + inventory[j].Kind + "/" + inventory[j].Namespace + "/" + inventory[j].Name
		return left < right
	})
	return json.Marshal(inventory)
}

func parseHelmManifestResources(manifest []byte, namespace string) ([]unstructured.Unstructured, error) {
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	var resources []unstructured.Unstructured
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode rendered Helm manifest: %w", err)
		}
		if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
			continue
		}
		var resource unstructured.Unstructured
		if json.Unmarshal(raw, &resource) != nil || resource.GetAPIVersion() == "" || resource.GetKind() == "" || resource.GetName() == "" {
			return nil, fmt.Errorf("rendered Helm resource identity is invalid")
		}
		if _, allowed := allowedHelmKinds[resource.GetKind()]; !allowed {
			return nil, fmt.Errorf("Helm resource kind %q is not allowed for project delivery", resource.GetKind())
		}
		if _, _, err := helmStrictDecoder.Decode(raw, nil, nil); err != nil {
			// Decoder errors may contain supplied values, including credentials.
			return nil, fmt.Errorf("Helm resource %s/%s has invalid Kubernetes fields or types", resource.GetKind(), resource.GetName())
		}
		resourceNamespace := resource.GetNamespace()
		if resourceNamespace != "" && resourceNamespace != namespace {
			return nil, fmt.Errorf("Helm resource %s/%s targets namespace %q instead of %q", resource.GetKind(), resource.GetName(), resourceNamespace, namespace)
		}
		resource.SetNamespace(namespace)
		resources = append(resources, resource)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("rendered Helm manifest contains no resources")
	}
	return resources, nil
}

var helmStrictDecoder = serializer.NewCodecFactory(scheme.Scheme, serializer.EnableStrict).UniversalDeserializer()

var allowedHelmKinds = map[string]struct{}{
	"ConfigMap": {}, "Secret": {}, "ServiceAccount": {}, "Role": {}, "RoleBinding": {},
	"Service": {}, "Deployment": {}, "StatefulSet": {}, "DaemonSet": {},
	"Pod": {}, "Job": {}, "CronJob": {}, "Ingress": {}, "PodDisruptionBudget": {},
	"HorizontalPodAutoscaler": {}, "NetworkPolicy": {},
	"PersistentVolumeClaim": {},
}
