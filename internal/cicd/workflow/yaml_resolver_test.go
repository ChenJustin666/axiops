package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeYAMLSourceStore struct {
	source   *YAMLSource
	artifact *BuildArtifact
}

func (s fakeYAMLSourceStore) LoadYAMLSource(context.Context, int64, int64, int64) (*YAMLSource, error) {
	return s.source, nil
}
func (s fakeYAMLSourceStore) LoadBuildArtifact(context.Context, int64) (*BuildArtifact, error) {
	return s.artifact, nil
}

func TestYAMLConfigValidationDefersOnlyBuildImage(t *testing.T) {
	source := &YAMLSource{ProjectID: 10, ServiceID: 9, ServiceRevisionID: 12,
		DeliveryType: DeliveryTypeK8sYAML, DefinitionFormat: DeliveryTypeK8sYAML,
		EnvironmentID: 7, EnvironmentType: "k8s_namespace", BindingStatus: "validated",
		EnvironmentStatus: "active", ClusterID: 2, Namespace: "test", ServiceName: "api",
		Manifest:  []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\ndata:\n  image: {{.Image}}\n  alias: $api-image$\n  port: '{{.port}}'\n"),
		Variables: map[string]string{"port": "8080"}}
	resolver := NewDefaultYAMLResolver(fakeYAMLSourceStore{source: source})
	run := WorkflowRun{ProjectID: 10, EnvironmentID: 7}
	cfg := YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12, FrozenYAML: source}
	target, err := resolver.RenderYAMLConfig(context.Background(), run, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ValidateYAMLManifest(target.Manifest, target.Namespace); err != nil {
		t.Fatal(err)
	}
	if source.ImageName != "" || source.ImageRef != "" {
		t.Fatal("validation changed the source or frozen input")
	}
	if _, err = resolver.RenderYAML(context.Background(), run, cfg); err == nil || !strings.Contains(err.Error(), "Image") {
		t.Fatalf("execution accepted a missing image: %v", err)
	}
	delete(source.Variables, "port")
	if _, err = resolver.RenderYAMLConfig(context.Background(), run, cfg); err == nil || !strings.Contains(err.Error(), "undefined variables: port") {
		t.Fatalf("configuration accepted a missing variable: %v", err)
	}
}

func TestDefaultYAMLResolverEnforcesProjectAndEnvironment(t *testing.T) {
	resolver := NewDefaultYAMLResolver(fakeYAMLSourceStore{source: &YAMLSource{
		ProjectID: 10, ServiceID: 9, ServiceRevisionID: 12, DeliveryType: DeliveryTypeK8sYAML,
		DefinitionFormat: DeliveryTypeK8sYAML, EnvironmentID: 7, EnvironmentType: "k8s_namespace",
		BindingStatus: "validated", EnvironmentStatus: "active", ClusterID: 2, Namespace: "test", Manifest: []byte("manifest"),
	}})
	target, err := resolver.RenderYAML(context.Background(), WorkflowRun{ProjectID: 10, EnvironmentID: 7}, YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12})
	if err != nil || target.ClusterID != 2 || target.Namespace != "test" {
		t.Fatalf("target=%+v err=%v", target, err)
	}
	if _, err := resolver.RenderYAML(context.Background(), WorkflowRun{ProjectID: 11, EnvironmentID: 7}, YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12}); err == nil {
		t.Fatal("expected project isolation error")
	}
}

func TestYAMLResolverInjectsBuildDigest(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: registry.example/team/api:old\n")
	resolver := NewDefaultYAMLResolver(fakeYAMLSourceStore{source: &YAMLSource{ProjectID: 10, ServiceID: 9, ServiceRevisionID: 12, DeliveryType: DeliveryTypeK8sYAML, DefinitionFormat: DeliveryTypeK8sYAML, EnvironmentID: 7, EnvironmentType: "k8s_namespace", BindingStatus: "validated", EnvironmentStatus: "active", ClusterID: 2, Namespace: "test", Manifest: manifest, ImageName: "registry.example/team/api"}, artifact: &BuildArtifact{ImageURL: "registry.example/team/api:new", ImageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}})
	node, _ := json.Marshal(Node{Type: NodeBuild, Config: mustJSON(BuildNodeConfig{ServiceID: 9})})
	target, err := resolver.RenderYAML(context.Background(), WorkflowRun{ProjectID: 10, EnvironmentID: 7, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, NodeSnapshot: node, ProviderRunID: "build:31"}}}, YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12})
	if err != nil || !strings.Contains(string(target.Manifest), "registry.example/team/api:new@sha256:aaaaaaaa") {
		t.Fatalf("manifest=%s err=%v", target.Manifest, err)
	}
}

func TestEnvironmentImageOverrideDoesNotNeedBuild(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: registry.example/team/api:old\n")
	resolver := NewDefaultYAMLResolver(fakeYAMLSourceStore{source: &YAMLSource{ProjectID: 10, ServiceID: 9, ServiceRevisionID: 12, DeliveryType: DeliveryTypeK8sYAML, DefinitionFormat: DeliveryTypeK8sYAML, EnvironmentID: 7, EnvironmentType: "k8s_namespace", BindingStatus: "validated", EnvironmentStatus: "active", ClusterID: 2, Namespace: "test", Manifest: manifest, ImageName: "registry.example/team/api", ImageRef: "registry.example/team/api@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}})
	target, err := resolver.RenderYAML(context.Background(), WorkflowRun{ProjectID: 10, EnvironmentID: 7}, YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12})
	if err != nil || !strings.Contains(string(target.Manifest), "sha256:bbbbbbbb") {
		t.Fatalf("manifest=%s err=%v", target.Manifest, err)
	}
}

func TestRenderYAMLDefinitionResolvesVariablesAndRemovesMetadata(t *testing.T) {
	replicas := 3
	source := &YAMLSource{
		ProductName: "pic03", ServiceName: "web-lib", EnvironmentName: "qa",
		ClusterName: "dev-cluster", Namespace: "pic03-qa", ImageName: "registry/team/web-lib",
		Replicas: &replicas,
	}
	definition := []byte("# AxiOps-Variables: [{\"key\":\"port\",\"value\":\"8080\"},{\"key\":\"cmd\",\"value\":\"server\"}]\n" +
		"# AxiOps-File: deployment.yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: {{.Service}}\nspec:\n  replicas: {{.replicas}}\n  template:\n    spec:\n      containers:\n      - name: {{.Service}}\n        image: {{.Image}}\n        command: [{{.cmd}}]\n        ports:\n        - containerPort: {{.port}}\n")
	rendered, err := renderYAMLDefinition(definition, source, "registry/team/web-lib@sha256:abc")
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	text := string(rendered)
	for _, expected := range []string{"name: web-lib", "replicas: 3", "registry/team/web-lib@sha256:abc", "containerPort: 8080", "server"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("rendered yaml missing %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "AxiOps-Variables") || strings.Contains(text, "{{.") {
		t.Fatalf("template metadata/placeholders leaked into rendered yaml: %s", text)
	}
}

func TestRenderYAMLDefinitionRejectsUndefinedVariable(t *testing.T) {
	_, err := renderYAMLDefinition([]byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Unknown}}\n"), &YAMLSource{}, "")
	if err == nil || !strings.Contains(err.Error(), "undefined variables") {
		t.Fatalf("expected undefined variable error, got %v", err)
	}
}

func TestRenderYAMLDefinitionUsesStarterDefaultsForLegacyManualService(t *testing.T) {
	source := &YAMLSource{SourceType: "manual", ServiceName: "api", ImageName: "registry/team/api"}
	definition := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  replicas: {{.replicas}}\n  template:\n    spec:\n      containers:\n      - name: api\n        image: {{.Image}}\n        ports:\n        - containerPort: {{.port}}\n")
	rendered, err := renderYAMLDefinition(definition, source, "registry/team/api:latest")
	if err != nil {
		t.Fatal(err)
	}
	text := string(rendered)
	for _, expected := range []string{"replicas: 1", "containerPort: 8080", "cpu: 100m", "memory: 128Mi", "cpu: 500m", "memory: 512Mi"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("legacy manual default %q missing from %s", expected, text)
		}
	}
}

func TestRenderYAMLDefinitionPreservesExplicitResources(t *testing.T) {
	source := &YAMLSource{ServiceName: "api", ImageName: "registry/team/api"}
	definition := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  replicas: 2\n  template:\n    spec:\n      containers:\n      - name: api\n        image: {{.Image}}\n        resources:\n          requests:\n            cpu: 250m\n            memory: 256Mi\n          limits:\n            cpu: 1\n            memory: 1Gi\n")
	rendered, err := renderYAMLDefinition(definition, source, "registry/team/api:latest")
	if err != nil {
		t.Fatal(err)
	}
	text := string(rendered)
	for _, expected := range []string{"replicas: 2", "cpu: 250m", "memory: 256Mi", "cpu: 1", "memory: 1Gi"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("explicit resource %q was not preserved: %s", expected, text)
		}
	}
}

func TestRenderYAMLDefinitionAppliesConfiguredProbes(t *testing.T) {
	definition := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: {{.Image}}\n        ports:\n        - containerPort: {{.port}}\n")
	for _, test := range []struct {
		name  string
		mode  string
		want  string
		avoid string
	}{
		{name: "disabled", mode: "none", avoid: "readinessProbe"},
		{name: "http", mode: "http", want: "path: /health", avoid: "tcpSocket"},
		{name: "tcp", mode: "tcp", want: "port: 8080", avoid: "httpGet"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := &YAMLSource{ServiceName: "api", ImageName: "registry/team/api", Variables: map[string]string{"port": "8080"}}
			values := map[string]string{"port": "8080", "Image": "registry/team/api:latest", "healthCheckType": test.mode, "healthPath": "/health", "healthPort": "0"}
			text := string(definition)
			renderedText, err := renderYAMLDefinition([]byte(text), source, values["Image"])
			if err != nil {
				t.Fatal(err)
			}
			text = string(renderedText)
			if test.mode == "none" {
				return
			}
			rendered, err := applyStandardProbes([]byte(text), values)
			if err != nil || !strings.Contains(string(rendered), test.want) || strings.Contains(string(rendered), test.avoid) {
				t.Fatalf("probe mode %s rendered incorrectly: %s err=%v", test.mode, rendered, err)
			}
		})
	}
}

func TestRenderYAMLDefinitionExpandsNestedVariables(t *testing.T) {
	source := &YAMLSource{ServiceName: "web-lib", Namespace: "pic03-dev"}
	definition := []byte("# AxiOps-Variables: [{\"key\":\"cmd\",\"value\":\"$Service$\"}]\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{.cmd}}\n")
	rendered, err := renderYAMLDefinition(definition, source, "")
	if err != nil || !strings.Contains(string(rendered), "name: web-lib") {
		t.Fatalf("nested variable was not expanded: %s (err=%v)", rendered, err)
	}
}

func TestYAMLVariableOverridesBeatRevisionDefaults(t *testing.T) {
	replicas := 0
	source := &YAMLSource{ServiceName: "risk-server", Namespace: "risk-qa", Replicas: &replicas,
		Variables: map[string]string{"port": "8001", "replicas": "2", "Namespace": "wrong"}}
	definition := []byte(`  # AxiOps-Variables: [{"key":"port","value":"8080"},{"key":"replicas","value":"1"},{"key":"Image","value":"old"}]
namespace: {{.Namespace}}
replicas: {{.replicas}}
port: {{.port}}
image: {{.Image}}
alias: $risk-server-image$
`)
	rendered, err := renderYAMLDefinition(definition, source, "registry/team/risk_server@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"namespace: risk-qa", "replicas: 0", "port: 8001", "image: registry/team/risk_server@sha256:abc", "alias: registry/team/risk_server@sha256:abc"} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("missing %q: %s", want, rendered)
		}
	}
	if strings.Contains(string(rendered), "AxiOps-Variables") {
		t.Fatalf("metadata leaked: %s", rendered)
	}
}

func TestYAMLResolverInjectsFullBuildImageForLegacyShortName(t *testing.T) {
	manifest := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: risk-server\nspec:\n  replicas: {{.replicas}}\n  template:\n    spec:\n      containers:\n      - name: risk-server\n        image: {{.Image}}\n        ports:\n        - containerPort: {{.port}}\n")
	source := &YAMLSource{ProjectID: 19, ServiceID: 17, ServiceRevisionID: 13, DeliveryType: DeliveryTypeK8sYAML, DefinitionFormat: DeliveryTypeK8sYAML, EnvironmentID: 17, EnvironmentType: "k8s_namespace", BindingStatus: "validated", EnvironmentStatus: "active", ClusterID: 2, Namespace: "risk-qa", Manifest: manifest, ImageName: "risk_server"}
	image := "registry/team/risk_server:test@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	resolver := NewDefaultYAMLResolver(fakeYAMLSourceStore{source: source, artifact: &BuildArtifact{ImageURL: strings.Split(image, "@")[0], ImageDigest: strings.Split(image, "@")[1]}})
	node, _ := json.Marshal(Node{Type: NodeBuild, Config: mustJSON(BuildNodeConfig{ServiceID: 17})})
	run := WorkflowRun{ProjectID: 19, EnvironmentID: 17, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, NodeSnapshot: node, ProviderRunID: "build:11"}}}
	cfg := YAMLDeployConfig{ServiceID: 17, ServiceRevisionID: 13}
	if _, err := resolver.RenderYAML(context.Background(), run, cfg); err == nil || !strings.Contains(err.Error(), "port, replicas") {
		t.Fatalf("expected stable missing variable list: %v", err)
	}
	source.Variables = map[string]string{"port": "8001", "replicas": "1"}
	target, err := resolver.RenderYAML(context.Background(), run, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(target.Manifest), image) || !strings.Contains(string(target.Manifest), "containerPort: 8001") {
		t.Fatalf("wrong manifest: %s", target.Manifest)
	}
	if _, err := ValidateYAMLManifest(target.Manifest, "risk-qa"); err != nil {
		t.Fatal(err)
	}
	// A different hardcoded image must still fail; no blanket replacement.
	source.Manifest = []byte(strings.ReplaceAll(string(manifest), "{{.Image}}", "other/team/api:old"))
	if _, err := resolver.RenderYAML(context.Background(), run, cfg); err == nil {
		t.Fatal("unrelated image should not be overwritten")
	}
}
