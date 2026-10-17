package workflow

import (
	"context"
	"encoding/json"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"testing"
)

type fakeHelmSourceStore struct {
	source    *HelmSource
	variables []HelmVariable
}

func TestHelmSharedValuesPrecedenceAndCredentialProtection(t *testing.T) {
	source := validHelmSource()
	settings := helmconfig.Default()
	source.Settings = &settings
	settings.SharedValues = "resources:\n  limits:\n    cpu: 2\n    memory: 4Gi\nregion: qa\nenv: [shared]\n"
	settings.Values = "resources:\n  limits:\n    memory: 8Gi\nenv: [service]\n"
	store := fakeHelmSourceStore{source: source, variables: []HelmVariable{{Key: "resources.limits.cpu", Value: "1"}, {Key: "keep", Value: "yes"}}}
	target, err := NewDefaultHelmResolver(store, nil).RenderHelmConfig(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"})
	if err != nil {
		t.Fatal(err)
	}
	limits := target.Values["resources"].(map[string]interface{})["limits"].(map[string]interface{})
	if limits["cpu"] != 2 || limits["memory"] != "8Gi" || target.Values["keep"] != "yes" || target.Values["env"].([]interface{})[0] != "service" {
		t.Fatal("wrong values precedence")
	}
	settings.SharedValues = "runtime:\n  password: forged\n"
	store.variables = []HelmVariable{{Key: "runtime.password", Secret: true, CredentialID: 1, CredentialType: "secret", CredentialProjectID: 7}}
	resolver := NewDefaultHelmResolver(store, fakeCredentialDecryptor{plaintext: `{"value":"secret-payload"}`})
	if _, err = resolver.RenderHelmConfig(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"}); err == nil {
		t.Fatal("shared values overwrote credential binding")
	}
}

func TestHelmCredentialFieldSelectionAndPrecedence(t *testing.T) {
	source := validHelmSource()
	settings := helmconfig.Default()
	settings.CredentialKeys = map[string]string{"runtime.password": "password", "runtime.sk": "SK"}
	source.Settings = &settings
	variables := []HelmVariable{
		{Key: "runtime.password", Secret: true, CredentialID: 1, CredentialType: "secret", CredentialProjectID: 99}, // overridden, must never be read
		{Key: "runtime.password", Secret: true, CredentialID: 2, CredentialType: "secret", CredentialProjectID: 7, CredentialData: "encrypted"},
		{Key: "runtime.sk", Secret: true, CredentialID: 2, CredentialType: "secret", CredentialProjectID: 7, CredentialData: "encrypted"},
	}
	store := fakeHelmSourceStore{source: source, variables: variables}
	resolver := NewDefaultHelmResolver(store, fakeCredentialDecryptor{plaintext: `{"password":"test-password","SK":"test-sk"}`})
	target, err := resolver.RenderHelmConfig(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"})
	if err != nil {
		t.Fatal(err)
	}
	runtime := target.Values["runtime"].(map[string]interface{})
	if runtime["password"] != "test-password" || runtime["sk"] != "test-sk" {
		t.Fatal("multi-field credential mapping failed")
	}
	snapshot, _ := json.Marshal(target.SnapshotValues)
	if containsString(string(snapshot), "test-password") || containsString(string(snapshot), "test-sk") || !containsString(string(snapshot), `"key":"SK"`) {
		t.Fatal("invalid reference snapshot")
	}
	settings.CredentialKeys["runtime.sk"] = "missing"
	if _, err = resolver.RenderHelmConfig(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"}); err == nil {
		t.Fatal("missing credential key accepted")
	}
}

func (s fakeHelmSourceStore) LoadHelmSource(context.Context, int64, int64, int64) (*HelmSource, error) {
	return s.source, nil
}

func (s fakeHelmSourceStore) LoadHelmVariables(context.Context, int64, int64, int64) ([]HelmVariable, error) {
	return s.variables, nil
}

type fakeCredentialDecryptor struct{ plaintext string }

func (d fakeCredentialDecryptor) Decrypt(string) (string, error) { return d.plaintext, nil }

func validHelmSource() *HelmSource {
	definition := testHelmDefinition("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n")
	return &HelmSource{
		ProjectID: 7, ServiceID: 8, ServiceRevisionID: 9, EnvironmentID: 10, ClusterID: 11,
		DeliveryType: DeliveryTypeHelm, DefinitionFormat: "helm_chart", EnvironmentType: "k8s_namespace",
		BindingStatus: "validated", EnvironmentStatus: "active", Namespace: "test", Definition: definition,
	}
}

func TestHelmResolverLayersValuesAndRedactsCredential(t *testing.T) {
	store := fakeHelmSourceStore{source: validHelmSource(), variables: []HelmVariable{
		{Key: "replicaCount", Value: "1"},
		{Key: "replicaCount", Value: "3"},
		{Key: "db.password", CredentialID: 12, CredentialType: "secret", CredentialProjectID: 7, CredentialData: "encrypted", Secret: true},
	}}
	target, err := NewDefaultHelmResolver(store, fakeCredentialDecryptor{plaintext: `{"value":"runtime-secret"}`}).RenderHelm(
		context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if target.Values["replicaCount"] != 3 || target.Values["db"].(map[string]interface{})["password"] != "runtime-secret" {
		t.Fatalf("runtime values=%+v", target.Values)
	}
	snapshot, _ := json.Marshal(target.SnapshotValues)
	if string(snapshot) == "" || string(snapshot) == "null" || containsString(string(snapshot), "runtime-secret") || !containsString(string(snapshot), "credential_id") {
		t.Fatalf("snapshot values=%s", snapshot)
	}
}

func TestHelmResolverRejectsCrossProjectCredential(t *testing.T) {
	store := fakeHelmSourceStore{source: validHelmSource(), variables: []HelmVariable{{
		Key: "db.password", CredentialID: 12, CredentialType: "secret", CredentialProjectID: 99, CredentialData: "encrypted", Secret: true,
	}}}
	_, err := NewDefaultHelmResolver(store, fakeCredentialDecryptor{plaintext: `{"value":"secret"}`}).RenderHelm(
		context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api"},
	)
	if err == nil {
		t.Fatal("cross-project Helm secret credential was accepted")
	}
}

func containsString(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
