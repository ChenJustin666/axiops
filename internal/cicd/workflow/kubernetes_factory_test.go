package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

type fakeClusterConnectionStore struct{ connection *ClusterConnection }

func (s fakeClusterConnectionStore) LoadClusterConnection(context.Context, int64) (*ClusterConnection, error) {
	return s.connection, nil
}

func TestKubernetesClientFactoryRequiresEncryptedKubeconfig(t *testing.T) {
	box, _ := secretbox.New("test-key")
	factory := NewSQLKubernetesClientFactory(fakeClusterConnectionStore{connection: &ClusterConnection{
		Status: "connected", EncryptedKubeconfig: "apiVersion: v1",
	}}, box, KubernetesFactoryOptions{})
	if _, err := factory.Client(context.Background(), 2); err == nil || !strings.Contains(err.Error(), "not encrypted") {
		t.Fatalf("err=%v", err)
	}
}

func TestKubernetesClientFactoryBuildsClientWithoutConnecting(t *testing.T) {
	box, _ := secretbox.New("test-key")
	kubeconfig := `apiVersion: v1
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
	encrypted, _ := box.Encrypt(kubeconfig)
	factory := NewSQLKubernetesClientFactory(fakeClusterConnectionStore{connection: &ClusterConnection{
		Status: "connected", EncryptedKubeconfig: encrypted,
	}}, box, KubernetesFactoryOptions{})
	client, err := factory.Client(context.Background(), 2)
	if err != nil || client == nil {
		t.Fatalf("client=%v err=%v", client, err)
	}
}
