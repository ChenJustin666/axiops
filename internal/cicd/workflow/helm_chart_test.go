package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
)

func testHelmDefinition(template string) []byte {
	contents := map[string]string{
		"Chart.yaml":               "apiVersion: v2\nname: api\ntype: application\nversion: 1.0.0\n",
		"values.yaml":              "image:\n  tag: v1\n",
		"templates/configmap.yaml": template,
	}
	definition := helmimport.Definition{SchemaVersion: 1, ChartPath: "deploy/api"}
	for _, name := range []string{"Chart.yaml", "templates/configmap.yaml", "values.yaml"} {
		content := contents[name]
		digest := sha256.Sum256([]byte(content))
		definition.Files = append(definition.Files, helmimport.ChartFile{Path: name, Content: content, SHA256: hex.EncodeToString(digest[:])})
	}
	data, _ := json.Marshal(definition)
	return data
}

func TestOfficialHelmRendererUsesImmutableChartAndValues(t *testing.T) {
	definition := testHelmDefinition("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}\ndata:\n  tag: {{ .Values.image.tag | quote }}\n")
	manifest, err := renderHelmManifest(context.Background(), HelmRequest{
		RunID: 1, NodeKey: "deploy", ClusterID: 2, Namespace: "test", ReleaseName: "api",
		Chart: definition, Values: map[string]interface{}{"image": map[string]interface{}{"tag": "v2"}},
		TimeoutSeconds: 60, IdempotencyKey: "workflow:1:deploy",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "name: api") || !strings.Contains(string(manifest), `tag: "v2"`) {
		t.Fatalf("unexpected manifest:\n%s", manifest)
	}
	inventory, err := helmManifestInventory(manifest, "test")
	if err != nil || strings.Contains(string(inventory), "v2") || !strings.Contains(string(inventory), "ConfigMap") {
		t.Fatalf("inventory=%s err=%v", inventory, err)
	}
}

func TestHelmInventoryRejectsClusterScopedResources(t *testing.T) {
	manifest := []byte("apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\nmetadata:\n  name: unsafe\n")
	if _, err := helmManifestInventory(manifest, "test"); err == nil {
		t.Fatal("cluster-scoped Helm resource was accepted")
	}
}

func TestImmutableHelmChartRejectsChangedFile(t *testing.T) {
	definition := testHelmDefinition("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n")
	var parsed helmimport.Definition
	_ = json.Unmarshal(definition, &parsed)
	parsed.Files[0].Content += "changed"
	definition, _ = json.Marshal(parsed)
	if _, err := loadImmutableHelmChart(definition); err == nil {
		t.Fatal("changed immutable Chart file was accepted")
	}
}
