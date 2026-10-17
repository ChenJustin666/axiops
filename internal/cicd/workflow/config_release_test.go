package workflow

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"
)

func TestConfigurationOnlyPreservesEachContainerAndRejectsNewTargets(t *testing.T) {
	manifest := []byte(`apiVersion: apps/v1
kind: Deployment
metadata: {name: api}
spec:
  template:
    spec:
      containers:
      - name: app
        image: registry/new:latest
        env:
        - name: LOG_LEVEL
          value: info
      - name: sidecar
        image: registry/other:latest
`)
	a := "registry/app@sha256:" + strings.Repeat("a", 64)
	b := "registry/sidecar@sha256:" + strings.Repeat("b", 64)
	result, err := preserveDeploymentImages(manifest, "test", map[string]string{"Deployment/api/containers/app": a, "Deployment/api/containers/sidecar": b})
	if err != nil || !strings.Contains(string(result), a) || !strings.Contains(string(result), b) || strings.Contains(string(result), "latest") {
		t.Fatalf("images not preserved: %v", err)
	}
	if _, err = preserveDeploymentImages(manifest, "test", map[string]string{"Deployment/api/containers/app": a}); err == nil {
		t.Fatal("new sidecar silently adopted an image")
	}
	for _, fragment := range []string{"- name: LOG_LEVEL\n          value: 1", "- name: LOG_LEVEL\n          value: info\n        - name: LOG_LEVEL\n          value: debug", "- name: LOG_LEVEL\n          value: info\n          valueFrom: {fieldRef: {fieldPath: metadata.name}}"} {
		invalid := strings.Replace(string(manifest), "- name: LOG_LEVEL\n          value: info", fragment, 1)
		if _, err = ValidateYAMLManifest([]byte(invalid), "test"); err == nil {
			t.Fatal("invalid env accepted")
		}
	}
}

type configurationRepository struct {
	*promotionRepository
	current *CurrentDeployment
}

func (r *configurationRepository) LoadConfigurationDeployment(context.Context, int64, int64, int64) (*CurrentDeployment, error) {
	return r.current, nil
}
func TestConfigurationOnlyFreezesConfigAndPreservesApproval(t *testing.T) {
	logic, p, req := promotionFixture()
	current := &CurrentDeployment{SnapshotID: 5, Namespace: "app-prod", Images: map[string]string{"Deployment/api/containers/app": "registry/app@sha256:" + strings.Repeat("a", 64)}}
	repo := &configurationRepository{promotionRepository: p, current: current}
	logic.repo = repo
	frozen, _ := p.FreezeDeploymentConfigs(context.Background(), 7, p.item.Nodes, false)
	var cfg YAMLDeployConfig
	json.Unmarshal(frozen[2].Config, &cfg)
	req.ExecutionMode = "config_only"
	req.SourceBuildID = 0
	req.SourceSnapshotID = 5
	req.ExpectedConfigDigest = DeploymentConfigDigest(cfg.FrozenYAML)
	_, code, msg := logic.RunWithValidation(context.Background(), 7, 9, req)
	if code != 0 || len(p.tasks) != 2 || p.run.Status != RunPendingApproval {
		t.Fatalf("configuration run failed or skipped approval: %d %s", code, msg)
	}
	var n Node
	json.Unmarshal(p.tasks[1].NodeSnapshot, &n)
	json.Unmarshal(n.Config, &cfg)
	if cfg.FrozenYAML.Assets.Resources.MemoryLimit != "4Gi" || cfg.FrozenYAML.PreservedImages["Deployment/api/containers/app"] != current.Images["Deployment/api/containers/app"] {
		t.Fatal("configuration or images not frozen")
	}
	p.run = nil
	req.ExpectedConfigDigest = strings.Repeat("0", 64)
	_, code, _ = logic.RunWithValidation(context.Background(), 7, 9, req)
	if code == 0 || p.run != nil {
		t.Fatal("changed config accepted")
	}
	req.SourceSnapshotID = 99
	_, code, _ = logic.RunWithValidation(context.Background(), 7, 9, req)
	if code == 0 {
		t.Fatal("changed snapshot accepted")
	}
}

func TestManagedConfigMapChangeRollsOnlyConsumers(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata: {name: api}
spec:
  template:
    spec:
      containers:
      - name: app
        image: registry/app:1
        envFrom:
        - configMapRef: {name: app-config}
---
apiVersion: v1
kind: ConfigMap
metadata: {name: app-config}
data: {MODE: debug}
---
apiVersion: v1
kind: ConfigMap
metadata: {name: unrelated}
data: {FOO: abc}
`
	hash := func(text string) string {
		t.Helper()
		result, err := stampManagedConfigMaps([]byte(text), "test")
		if err != nil {
			t.Fatal(err)
		}
		objects, err := ValidateYAMLManifest(result, "test")
		if err != nil {
			t.Fatal(err)
		}
		value, _, _ := unstructured.NestedString(objects[0].Object, "spec", "template", "metadata", "annotations", "axiops.io/managed-configmap-hash")
		return value
	}
	original := hash(manifest)
	if original == "" || original == hash(strings.Replace(manifest, "MODE: debug", "MODE: info", 1)) {
		t.Fatal("consumed config change did not roll the workload")
	}
	if original != hash(strings.Replace(manifest, "FOO: abc", "FOO: def", 1)) {
		t.Fatal("unrelated config change rolled the workload")
	}
}
