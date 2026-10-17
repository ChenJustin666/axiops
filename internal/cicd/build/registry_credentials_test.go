package build

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestRegistryDockerConfigScopesCredentials(t *testing.T) {
	auth := &RegistryAuth{BaseURL: "https://registry.example/team", Username: "robot", Password: strings.Repeat("long-password", 12)}
	config, err := registryDockerConfig("registry.example/team/api:v1", auth)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(parsed.Auths["registry.example"].Auth)
	if err != nil || string(decoded) != auth.Username+":"+auth.Password {
		t.Fatal("invalid registry auth encoding")
	}
	for _, image := range []string{"other.example/team/api:v1", "registry.example/team-other/api:v1"} {
		if _, err := registryDockerConfig(image, auth); err == nil {
			t.Fatalf("accepted credential scope mismatch: %s", image)
		}
	}
}

func TestRegistryCredentialsSupportConfiguredNamespaces(t *testing.T) {
	for _, base := range []string{"https://registry.example", "https://registry.example/sinozo"} {
		auth := &RegistryAuth{BaseURL: base, Namespaces: []string{"sinozo", "abc", "dbc/team"}, Username: "robot", Password: "test-password"}
		for _, path := range []string{"sinozo/risk_server", "abc/risk_server", "dbc/team/custom"} {
			if _, err := registryDockerConfig("registry.example/"+path+":v1", auth); err != nil {
				t.Fatalf("configured namespace rejected: %s: %v", path, err)
			}
		}
		for _, image := range []string{"other.example/abc/risk_server:v1", "registry.example/abcd/risk_server:v1", "registry.example/dbc/other:v1", "registry.example/abc:v1"} {
			if _, err := registryDockerConfig(image, auth); err == nil {
				t.Fatalf("namespace scope bypassed: %s", image)
			}
		}
	}
}

type recordingBuildSecrets struct {
	data    map[string]string
	deleted string
}

func (s *recordingBuildSecrets) Ensure(_ context.Context, _, _ string, data map[string]string) error {
	s.data = data
	return nil
}
func (s *recordingBuildSecrets) Delete(_ context.Context, name string) error {
	s.deleted = name
	return nil
}

func TestRegistryCredentialsAreMountedAndCleanedUp(t *testing.T) {
	client, secrets := &fakeJobClient{}, &recordingBuildSecrets{}
	provider := NewKubeJobProvider("ci", client, client).WithRuntimeData(secrets, nil, 1024)
	config, _ := registryDockerConfig("registry.example/api:v1", &RegistryAuth{BaseURL: "registry.example", Username: "robot", Password: "private-password"})
	req := ProviderBuildRequest{BuildRunID: 9, IdempotencyKey: "build:9:submit", RepositoryURL: "https://git.example/api.git", BuilderImage: "registry.example/builder:v1", BuildCommand: "build", GitCredential: "private-git-token", RegistryDockerConfig: config, EnvVars: []EnvVar{{Key: "DOCKER_CONFIG", Value: "/wrong"}}}
	execution, err := provider.Submit(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if secrets.data[corev1.DockerConfigJsonKey] != config || secrets.data["token"] != req.GitCredential {
		t.Fatal("credentials not stored together")
	}
	raw, _ := json.Marshal(client.created)
	if strings.Contains(string(raw), "private-") || strings.Contains(string(raw), config) {
		t.Fatal("Job leaks credential values")
	}
	pod := client.created.Spec.Template.Spec
	if len(pod.ImagePullSecrets) != 1 || pod.ImagePullSecrets[0].Name != "axiops-build-9-git" {
		t.Fatal("missing private image pull authentication")
	}
	found := false
	for _, env := range pod.Containers[0].Env {
		if env.Name == "DOCKER_CONFIG" {
			found = env.Value == "/run/axiops-registry"
		}
	}
	if !found {
		t.Fatal("registry config can be overridden")
	}
	for _, mount := range pod.InitContainers[0].VolumeMounts {
		if mount.Name == "registry-auth" {
			t.Fatal("git clone should not receive registry credentials")
		}
	}
	if err := provider.Cancel(context.Background(), execution); err != nil || secrets.deleted != "axiops-build-9-git" {
		t.Fatal("credentials not cleaned on cancellation")
	}
}
