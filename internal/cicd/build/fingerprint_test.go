package build

import (
	"strings"
	"testing"
)

func TestBuildFingerprintStableForEnvironmentReuse(t *testing.T) {
	envA := `[ {"key":"Z","value":"2"},{"key":"A","value":"1"} ]`
	envB := `[ {"key":"A","value":"1"},{"key":"Z","value":"2"} ]`
	configA := &BuildConfig{BuildImage: "builder:v1", TemplateType: "docker", BuildMode: "dockerfile", BuildCommand: "docker build .", EnvVars: &envA}
	configB := &BuildConfig{BuildImage: "builder:v1", TemplateType: "docker", BuildMode: "dockerfile", BuildCommand: "docker build .", EnvVars: &envB}
	left := BuildFingerprint(7, "https://git.example/repo.git", "main", strings.Repeat("a", 40), configA, "", false)
	right := BuildFingerprint(7, "https://git.example/repo.git", "main", strings.Repeat("a", 40), configB, "", false)
	if left == "" || left != right {
		t.Fatalf("fingerprint is not stable: %q != %q", left, right)
	}
	changed := BuildFingerprint(7, "https://git.example/repo.git", "main", strings.Repeat("b", 40), configA, "", false)
	if changed == left {
		t.Fatal("different commits must produce different fingerprints")
	}
}

func TestMovingSourceCannotBeReused(t *testing.T) {
	for _, commit := range []string{"", "main", "abc"} {
		if BuildFingerprint(7, "repo", "main", commit, nil, "", false) != "" {
			t.Fatal("mutable source reused")
		}
	}
}
