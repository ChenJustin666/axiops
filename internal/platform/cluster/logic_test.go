package cluster

import (
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

const testKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: test
`

func TestEncryptKubeconfigValidatesAndEncrypts(t *testing.T) {
	box, _ := secretbox.New("test-key")
	encrypted, err := encryptKubeconfig(box, testKubeconfig)
	if err != nil || !strings.HasPrefix(encrypted, "v1:") || strings.Contains(encrypted, "127.0.0.1") {
		t.Fatalf("encrypted=%q err=%v", encrypted, err)
	}
	plain, err := box.Decrypt(encrypted)
	if err != nil || plain != testKubeconfig {
		t.Fatalf("plain=%q err=%v", plain, err)
	}
}

func TestEncryptKubeconfigRejectsInvalidInput(t *testing.T) {
	box, _ := secretbox.New("test-key")
	if _, err := encryptKubeconfig(box, "not kubeconfig"); err == nil {
		t.Fatal("invalid kubeconfig was accepted")
	}
}

func TestResourceCapacityFormatting(t *testing.T) {
	if got := formatCPU(2500); got != "2.50 cores" {
		t.Fatalf("cpu=%q", got)
	}
	if got := formatMemory(2 * 1024 * 1024 * 1024); got != "2.00 GB" {
		t.Fatalf("memory=%q", got)
	}
}

func TestClusterInputValidation(t *testing.T) {
	if !validClusterIdentity("测试集群", "test-cluster") {
		t.Fatal("expected a valid cluster identity")
	}
	if validClusterIdentity("测试集群", "Test_Cluster") || validClusterIdentity("", "test") {
		t.Fatal("invalid cluster identity was accepted")
	}
	for _, value := range []string{"https://127.0.0.1:6443", "http://10.0.0.5:8080"} {
		if !validAPIServer(value) {
			t.Fatalf("expected API server %q to be valid", value)
		}
	}
	for _, value := range []string{"127.0.0.1:6443", "ftp://example.com"} {
		if validAPIServer(value) {
			t.Fatalf("expected API server %q to be rejected", value)
		}
	}
}
