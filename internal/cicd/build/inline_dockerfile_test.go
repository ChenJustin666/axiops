package build

import (
	"strings"
	"testing"
)

func TestInlineDockerfileMountIsFrozenAndNotShellInterpolated(t *testing.T) {
	content := "FROM eclipse-temurin:17-jre\nCOPY target/app.jar app.jar\nRUN echo '$(echo should-not-expand)'\n"
	req := ProviderBuildRequest{BuildRunID: 71, ServiceID: 1, ProjectID: 1, BuildMode: "dockerfile", DockerfileSource: "inline", DockerfileContent: content, BuildContext: "gateway"}
	job, err := BuildKubeJob("ci", req)
	if err != nil {
		t.Fatal(err)
	}
	git := job.Spec.Template.Spec.InitContainers[0]
	found := false
	for _, env := range git.Env {
		if env.Name == "AXIOPS_DOCKERFILE_CONTENT" {
			found = env.Value == content
		}
	}
	if !found || strings.Contains(git.Args[0], content) || !strings.Contains(git.Args[0], `printf '%s\n' "$AXIOPS_DOCKERFILE_CONTENT"`) {
		t.Fatal("Dockerfile must be literal env data, never shell code")
	}
	builder := job.Spec.Template.Spec.Containers[0]
	found = false
	for _, mount := range builder.VolumeMounts {
		if mount.Name == "inline-dockerfile" {
			found = mount.ReadOnly && mount.SubPath == "Dockerfile" && mount.MountPath == "/workspace/gateway/.axiops-generated.Dockerfile"
		}
	}
	if !found {
		t.Fatal("frozen Dockerfile not mounted into selected context")
	}
	for _, path := range []string{"../other", "/tmp", "a/../../b", "a\\b"} {
		req.BuildContext = path
		if _, err = BuildKubeJob("ci", req); err == nil {
			t.Fatalf("accepted unsafe context %q", path)
		}
	}
	for _, invalid := range []string{"", "# FROM only a comment", "RUN echo hi", strings.Repeat("x", 65537)} {
		if validateDockerfileInput("inline", invalid, ".") == nil {
			t.Fatal("accepted invalid inline file")
		}
	}
}

func TestInlineDockerfileChangesBuildFingerprint(t *testing.T) {
	cfg := &BuildConfig{BuildImage: "builder:v1", BuildCommand: "build", DockerfileSource: "inline", DockerfileContent: "FROM alpine:3.20"}
	one := BuildFingerprint(1, "https://example/repo", "main", strings.Repeat("a", 40), cfg, "", false)
	cfg.DockerfileContent = "FROM alpine:3.21"
	two := BuildFingerprint(1, "https://example/repo", "main", strings.Repeat("a", 40), cfg, "", false)
	if one == "" || one == two {
		t.Fatal("changed Dockerfile reused an old artifact fingerprint")
	}
}
