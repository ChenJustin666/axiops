package build

import (
	"strings"
	"testing"
)

func TestSnapshotResourceErrorIdentifiesServiceOverride(t *testing.T) {
	snapshot := BuildExecutionSnapshot{RepositoryURL: "https://git.example/app.git", Config: &BuildConfig{ResourceCPU: "1", ResourceMemory: "2"}}
	_, err := providerRequestFromSnapshot(BuildRun{ID: 5, ServiceID: 17}, snapshot, "ci")
	if err == nil || !strings.Contains(err.Error(), "服务 17") || !strings.Contains(err.Error(), `memory "2"`) || !strings.Contains(err.Error(), "仅清空平台") {
		t.Fatalf("resource error lacks configuration source and remedy: %v", err)
	}
	snapshot.Config.ResourceCPU, snapshot.Config.ResourceMemory = "", ""
	request, err := providerRequestFromSnapshot(BuildRun{ServiceID: 17}, snapshot, "ci")
	if err != nil || request.ResourceCPU != "" || request.ResourceMemory != "" {
		t.Fatalf("cleared service resources must pass through for platform inheritance: %+v %v", request, err)
	}
}

func TestJarSnapshotDefaultsRespectJavaTemplate(t *testing.T) {
	for _, template := range []string{"maven", "gradle"} {
		request, err := providerRequestFromSnapshot(BuildRun{ID: 7}, BuildExecutionSnapshot{
			RepositoryURL: "https://git.example/app.git", Config: &BuildConfig{BuildMode: "jar_image", TemplateType: template},
		}, "ci")
		if err != nil || !strings.HasPrefix(request.BuilderImage, template+":") || !strings.Contains(request.BuilderImage, "17") || request.BuildCommand == "" {
			t.Fatalf("wrong Java template defaults: %+v %v", request, err)
		}
	}
}

func TestProviderRequestUsesImmutableExecutionSnapshot(t *testing.T) {
	raw := `{"repository_url":"https://git.example.com/team/api.git"}`
	envJSON := `[{"key":"GOFLAGS","value":"-mod=readonly"}]`
	request, err := providerRequestFromSnapshot(BuildRun{
		ID: 31, ServiceID: 9, ProjectID: 10, GitBranch: "main", GitCommit: "abc", ConfigSnapshot: &raw,
	}, BuildExecutionSnapshot{
		RepositoryURL: "https://git.example.com/team/api.git",
		Config:        &BuildConfig{BuildImage: "golang:1.25", BuildCommand: "go test ./...", DockerfilePath: "docker/Dockerfile", BuildContext: "services/api", ImageRepo: "registry.example/api", CacheEnabled: true, CacheDir: ".cache", EnvVars: &envJSON, TimeoutSeconds: 600, ResourceCPU: "1", ResourceMemory: "2Gi"},
	}, "axiops-build")
	if err != nil || request.RepositoryURL != "https://git.example.com/team/api.git" || request.BuildCommand != "go test ./..." || request.Namespace != "axiops-build" || request.DockerfilePath != "docker/Dockerfile" || request.BuildContext != "services/api" || request.ImageRepo != "registry.example/api" || !request.CacheEnabled || len(request.EnvVars) != 1 || request.EnvVars[0].Key != "GOFLAGS" {
		t.Fatalf("request=%+v err=%v", request, err)
	}
}

func TestProviderRequestRejectsMalformedBuildEnvironment(t *testing.T) {
	envJSON := `[{"key":"A"},]`
	_, err := providerRequestFromSnapshot(BuildRun{ID: 31}, BuildExecutionSnapshot{RepositoryURL: "https://git.example.com/team/api.git", Config: &BuildConfig{EnvVars: &envJSON}}, "axiops-build")
	if err == nil {
		t.Fatal("malformed build environment variables were accepted")
	}
}

func TestProviderRequestRejectsMissingRepositorySnapshot(t *testing.T) {
	if _, err := providerRequestFromSnapshot(BuildRun{ID: 31}, BuildExecutionSnapshot{}, "axiops-build"); err == nil {
		t.Fatal("missing repository URL was accepted")
	}
}
