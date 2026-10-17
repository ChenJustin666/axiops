package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"helm.sh/helm/v3/pkg/kube"
)

func TestHelmNativeEnvironmentValuesAndCredentialIsolation(t *testing.T) {
	source := validHelmSource()
	settings := helmconfig.Default()
	settings.Values = "replicaCount: 5\ndb:\n  host: prod-db\nenv:\n- name: PROFILE\n  value: prod\n"
	source.Settings = &settings
	vars := []HelmVariable{{Key: "replicaCount", Value: "1"}, {Key: "db.password", Secret: true, CredentialID: 1, CredentialType: "secret", CredentialProjectID: 7}}
	resolver := NewDefaultHelmResolver(fakeHelmSourceStore{source: source, variables: vars}, fakeCredentialDecryptor{plaintext: `{"value":"a-private-password"}`})
	cfg := HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"}
	run := WorkflowRun{ProjectID: 7, EnvironmentID: 10}
	target, err := resolver.RenderHelmConfig(context.Background(), run, cfg)
	if err != nil || target.Values["replicaCount"] != 5 || target.Values["db"].(map[string]interface{})["host"] != "prod-db" {
		t.Fatalf("native override failed: %+v %v", target, err)
	}
	before, _ := json.Marshal(target.Values)
	redacted := redactedHelmValues(target.Values, target.SnapshotValues)
	after, _ := json.Marshal(target.Values)
	if !bytes.Equal(before, after) || redacted["db"].(map[string]interface{})["password"] == "a-private-password" {
		t.Fatal("redaction leaked or mutated runtime values")
	}
	for _, raw := range []string{"db: null", "db: replaced", "db:\n  password: plaintext"} {
		source.Settings.Values = raw
		if _, err := resolver.RenderHelmConfig(context.Background(), run, cfg); err == nil {
			t.Errorf("credential override allowed: %q", raw)
		}
	}
	source.Settings.Values = "{}"
	source.Settings.Image = helmconfig.ImageBinding{RefPath: "db"}
	if _, err := resolver.RenderHelmConfig(context.Background(), run, cfg); err == nil {
		t.Fatal("image mapping overwrote credential")
	}
}

func TestHelmProductionRequiresImmutableImageExceptConfigurationPreview(t *testing.T) {
	source := validHelmSource()
	source.IsProduction = true
	r := NewDefaultHelmResolver(fakeHelmSourceStore{source: source}, nil)
	cfg := HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"}
	run := WorkflowRun{ProjectID: 7, EnvironmentID: 10}
	if _, err := r.RenderHelm(context.Background(), run, cfg); err == nil {
		t.Fatal("production used mutable chart default")
	}
	if _, err := r.RenderHelmConfig(context.Background(), run, cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ImageRef = "registry.example.com/team/api:v1@sha256:" + strings.Repeat("a", 64)
	target, err := r.RenderHelm(context.Background(), run, cfg)
	if err != nil {
		t.Fatal(err)
	}
	image := target.Values["image"].(map[string]interface{})
	if image["repository"] != "registry.example.com/team/api" || image["tag"] != "v1@sha256:"+strings.Repeat("a", 64) {
		t.Fatalf("incorrect digest mapping: %+v", image)
	}
}

func TestHelmImageMappingMustBeUsedByRenderedChart(t *testing.T) {
	image := "registry.example.com/team/api:v1@sha256:" + strings.Repeat("b", 64)
	for _, binding := range []helmconfig.ImageBinding{{RepositoryPath: "image.repository", TagPath: "image.tag"}, {RefPath: "application.image"}} {
		values := map[string]interface{}{}
		if err := injectHelmImage(values, binding, image); err != nil {
			t.Fatal(err)
		}
		template := "{{ .Values.application.image }}"
		if binding.RefPath == "" {
			template = "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
		}
		definition := testHelmDefinition("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\nspec:\n  template:\n    spec:\n      containers:\n      - name: api\n        image: \"" + template + "\"\n")
		manifest, err := renderHelmManifest(context.Background(), HelmRequest{Namespace: "test", ReleaseName: "api", Chart: definition, Values: values}, false)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateHelmBuiltImage(manifest, "test", image); err != nil {
			t.Fatal(err)
		}
		if err := validateHelmBuiltImage(manifest, "test", strings.Replace(image, "v1@", "v2@", 1)); err == nil {
			t.Fatal("unused image mapping accepted")
		}
	}
	if err := injectHelmImage(map[string]interface{}{}, helmconfig.Default().Image, "registry.example.com/team/api:mutable"); err == nil {
		t.Fatal("mutable image accepted")
	}
}

type helmGuardClient struct {
	kube.Interface
	called bool
}

func (c *helmGuardClient) Build(r io.Reader, _ bool) (kube.ResourceList, error) {
	c.called = true
	_, err := io.ReadAll(r)
	return nil, err
}
func TestHelmActualRenderAndHooksCannotEscapeEnvironment(t *testing.T) {
	good := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\n")
	bad := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\n  namespace: prod\n")
	for _, raw := range [][]byte{bad, []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: outside\n")} {
		c := &helmGuardClient{}
		guard := helmScopedKubeClient{Interface: c, namespace: "test"}
		if _, err := guard.Build(bytes.NewReader(raw), false); err == nil || c.called {
			t.Fatal("unsafe hook reached Kubernetes client")
		}
	}
	p := helmPostRenderer{namespace: "test", prepare: func([]byte) ([]byte, error) { return bad, nil }}
	if _, err := p.Run(bytes.NewBuffer(good)); err == nil {
		t.Fatal("post-render mutation escaped namespace")
	}
	c := &helmGuardClient{}
	guard := helmScopedKubeClient{Interface: c, namespace: "test"}
	if _, err := guard.Build(bytes.NewReader(good), true); err != nil || !c.called {
		t.Fatal("valid scoped resource rejected")
	}
}

func TestHelmPublicPreviewDoesNotLeakTransformedCredentials(t *testing.T) {
	private := "do-not-expose-this-secret"
	values := map[string]interface{}{"password": private}
	snapshot := map[string]interface{}{"password": map[string]interface{}{"credential_id": 1}}
	definition := testHelmDefinition("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\ndata:\n  encoded: {{ .Values.password | b64enc | quote }}\n")
	publicValues := redactedHelmValues(values, snapshot)
	raw, err := renderHelmManifest(context.Background(), HelmRequest{Namespace: "test", ReleaseName: "api", Chart: definition, Values: publicValues}, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), private) || strings.Contains(string(raw), "ZG8tbm90LWV4cG9zZS10aGlzLXNlY3JldA==") {
		t.Fatal("encoded credential exposed")
	}
}

type actualRenderHelmProvider struct {
	fakeHelmProvider
	actual []byte
}

func (p *actualRenderHelmProvider) UpgradeInstall(_ context.Context, request HelmRequest) (HelmExecution, error) {
	if _, err := request.PrepareManifest(p.actual); err != nil {
		return HelmExecution{}, err
	}
	return HelmExecution{ExternalID: "actual-render", Completed: true, Revision: 2}, nil
}
func TestHelmSnapshotUsesActualUpgradePostRender(t *testing.T) {
	provider := &actualRenderHelmProvider{actual: []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: upgrade-only\n")}
	snapshots := &fakeHelmSnapshotStore{}
	request := helmLogRequest()
	_, err := NewHelmDeployExecutor(fakeHelmResolver{HelmTarget{ClusterID: 2, Namespace: "test", ReleaseName: "api", ChartPath: "."}}, provider, snapshots).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if snapshots.prepared == nil || !strings.Contains(string(snapshots.prepared.RenderedManifest), "upgrade-only") || snapshots.applied != 2 {
		t.Fatal("snapshot retained preliminary install render")
	}
}

func TestHelmOwnerLabelsRequireProjectEnvironmentAndService(t *testing.T) {
	expected := map[string]string{"axiops-project-id": "7", "axiops-environment-id": "10", "axiops-service-id": "8"}
	if !helmOwnerLabelsMatch(expected, expected) {
		t.Fatal("matching owner rejected")
	}
	for _, key := range []string{"axiops-project-id", "axiops-environment-id", "axiops-service-id"} {
		actual := map[string]string{}
		for k, v := range expected {
			actual[k] = v
		}
		actual[key] = "other"
		if helmOwnerLabelsMatch(actual, expected) {
			t.Fatal("foreign release owner accepted", key)
		}
	}
	if helmOwnerLabelsMatch(nil, expected) {
		t.Fatal("external release silently adopted")
	}
}
