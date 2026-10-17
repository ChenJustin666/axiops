package assetdeploy

import (
	"context"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestEnsureSecretOwnershipAndUpdate(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	desired := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "axiops-s17-secret-db", Namespace: "risk-test", Labels: map[string]string{"axiops.io/service-id": "17", "axiops.io/managed-by": "service-config"}}, Data: map[string][]byte{"password": []byte("one")}, Type: corev1.SecretTypeOpaque}
	if err := ensureSecret(ctx, client, desired); err != nil {
		t.Fatal(err)
	}
	desired.Data["password"] = []byte("two")
	if err := ensureSecret(ctx, client, desired); err != nil {
		t.Fatal(err)
	}
	got, err := client.CoreV1().Secrets("risk-test").Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil || string(got.Data["password"]) != "two" {
		t.Fatalf("secret was not updated: %v %#v", err, got.Data)
	}
	got.Labels["axiops.io/service-id"] = "other"
	if _, err = client.CoreV1().Secrets("risk-test").Update(ctx, got, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err = ensureSecret(ctx, client, desired); err == nil {
		t.Fatal("expected ownership refusal")
	}
}

func TestCredentialReplacementPreservesPreviousSecret(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	workload := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "risk"},
		"spec": map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{
			"containers": []interface{}{map[string]interface{}{"name": "risk", "image": "test-only"}},
		}}},
	}}
	var names []string
	for i, value := range []string{"old-test-key", "new-test-key"} {
		settings := serviceassets.Settings{Secrets: []serviceassets.Secret{{Name: "api", CredentialID: int64(42 + i), CredentialKey: "value", Key: "token", EnvName: "API_TOKEN"}}}
		_, refs, err := serviceassets.Apply([]unstructured.Unstructured{workload}, settings, 17, "risk-test")
		if err != nil || len(refs) != 1 {
			t.Fatalf("credential rendering failed: %v", err)
		}
		names = append(names, refs[0].Name)
		secret := corev1.Secret{ObjectMeta: serviceassets.SecretMetadata(refs[0].Name, "risk-test", 17), Type: corev1.SecretTypeOpaque, Data: map[string][]byte{"token": []byte(value)}}
		if err := ensureSecret(ctx, client, secret); err != nil {
			t.Fatal(err)
		}
	}
	if names[0] == names[1] {
		t.Fatal("replacement reused the previous Secret")
	}
	for i, expected := range []string{"old-test-key", "new-test-key"} {
		got, err := client.CoreV1().Secrets("risk-test").Get(ctx, names[i], metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if string(got.Data["token"]) != expected {
			t.Fatal("replacement changed the other credential's Secret")
		}
	}
}
