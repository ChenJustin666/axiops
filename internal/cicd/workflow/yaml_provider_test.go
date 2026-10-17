package workflow

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fakeKubernetesClient struct{ applied int }

func (c *fakeKubernetesClient) Apply(_ context.Context, _ string, resource unstructured.Unstructured, _ string, _ bool) (unstructured.Unstructured, error) {
	c.applied++
	return resource, nil
}
func (c *fakeKubernetesClient) Preview(context.Context, string, unstructured.Unstructured, string) (string, error) {
	return "configured", nil
}

type fakeKubernetesFactory struct{ client *fakeKubernetesClient }

func (f fakeKubernetesFactory) Client(context.Context, int64) (KubernetesResourceApplier, error) {
	return f.client, nil
}

func TestKubernetesYAMLProviderAppliesAndPreviewsResources(t *testing.T) {
	client := &fakeKubernetesClient{}
	provider := NewKubernetesYAMLProvider(fakeKubernetesFactory{client: client}, "axiops-test", false)
	request := YAMLProviderRequest{RunID: 3, NodeKey: "deploy", ClusterID: 2, Namespace: "test", IdempotencyKey: "workflow:3:deploy", Manifest: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n")}
	execution, err := provider.Apply(context.Background(), request)
	if err != nil || !execution.Completed || client.applied != 1 {
		t.Fatalf("execution=%+v applied=%d err=%v", execution, client.applied, err)
	}
	changes, err := provider.Preview(context.Background(), request)
	if err != nil || len(changes) != 1 || changes[0].Action != "configured" || changes[0].APIVersion != "v1" {
		t.Fatalf("changes=%+v err=%v", changes, err)
	}
	helmChanges, err := provider.PreviewHelm(context.Background(), 2, "test", []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: token\nstringData:\n  value: hidden\n"))
	if err != nil || len(helmChanges) != 1 || helmChanges[0].Kind != "Secret" {
		t.Fatalf("helm changes=%+v err=%v", helmChanges, err)
	}
}

func TestYAMLProviderStreamsProgressBeforeCompletion(t *testing.T) {
	client := &fakeKubernetesClient{}
	provider := NewKubernetesYAMLProvider(fakeKubernetesFactory{client: client}, "test", false)
	var messages []string
	request := YAMLProviderRequest{ClusterID: 2, Namespace: "test", IdempotencyKey: "workflow:3:deploy", Manifest: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\n"), ReportProgress: func(message string) error {
		if len(messages) == 0 && client.applied != 0 {
			t.Fatal("first progress was delayed until apply completed")
		}
		messages = append(messages, message)
		return nil
	}}
	result, err := provider.Apply(context.Background(), request)
	if err != nil || !result.Completed || result.LogText != "" || !strings.Contains(strings.Join(messages, "\n"), "Service/api submitted") {
		t.Fatalf("result=%+v messages=%v err=%v", result, messages, err)
	}
	client.applied = 0
	request.ReportProgress = func(string) error { return fmt.Errorf("progress persistence unavailable") }
	if _, err := provider.Apply(context.Background(), request); err == nil || client.applied != 0 {
		t.Fatal("applied resources without saving initial evidence")
	}
}
