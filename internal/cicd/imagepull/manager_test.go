package imagepull

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/build"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type authStore struct{ auth *build.RegistryAuth }

func (s *authStore) Load(_ context.Context, project, service int64) (*build.RegistryAuth, error) {
	if project != 19 || service != 17 {
		return nil, fmt.Errorf("wrong credential scope")
	}
	return s.auth, nil
}
func setup(t *testing.T) (*Manager, *fake.Clientset, *authStore) {
	t.Helper()
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "risk-qa"}})
	store := &authStore{auth: &build.RegistryAuth{BaseURL: "https://registry.example/sinozo", Username: "robot", Password: "test-private-password"}}
	return &Manager{auth: store, client: func(context.Context, int64) (kubernetes.Interface, error) { return client, nil }}, client, store
}
func target(manifest string) workflow.YAMLTarget {
	return workflow.YAMLTarget{ClusterID: 2, Namespace: "risk-qa", Manifest: []byte(manifest)}
}
func prepare(m *Manager, manifest string) (workflow.YAMLTarget, error) {
	return m.PrepareYAMLDeployment(context.Background(), workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 17}, workflow.YAMLDeployConfig{ServiceID: 17, ServiceRevisionID: 13}, target(manifest))
}

const manifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: risk-server
spec:
  template:
    spec:
      imagePullSecrets:
      - name: existing-secret
      containers:
      - name: app
        image: registry.example/sinozo/risk_server:test@sha256:abc
      - name: sidecar
        image: elsewhere.example/team/sidecar:v1
`

func TestManagedPullSecretIsNamespacedReusableAndNotInManifest(t *testing.T) {
	m, client, store := setup(t)
	output, err := prepare(m, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(output.Manifest), store.auth.Password) || strings.Contains(string(output.Manifest), base64.StdEncoding.EncodeToString([]byte("robot:"+store.auth.Password))) {
		t.Fatal("credentials entered snapshot manifest")
	}
	objects, err := workflow.ValidateYAMLManifest(output.Manifest, "risk-qa")
	if err != nil {
		t.Fatal(err)
	}
	refs, _, _ := unstructured.NestedSlice(objects[0].Object, "spec", "template", "spec", "imagePullSecrets")
	if len(refs) != 2 || refs[0].(map[string]interface{})["name"] != "existing-secret" {
		t.Fatalf("existing refs were lost: %v", refs)
	}
	secrets, err := client.CoreV1().Secrets("risk-qa").List(context.Background(), metav1.ListOptions{})
	if err != nil || len(secrets.Items) != 1 {
		t.Fatalf("secrets count wrong: %v", err)
	}
	secret := secrets.Items[0]
	if secret.Name != refs[1].(map[string]interface{})["name"] || secret.Type != corev1.SecretTypeDockerConfigJson {
		t.Fatal("wrong pull secret")
	}
	var docker struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(secret.Data[corev1.DockerConfigJsonKey], &docker); err != nil {
		t.Fatal(err)
	}
	if len(docker.Auths) != 1 || docker.Auths["registry.example"].Auth != base64.StdEncoding.EncodeToString([]byte("robot:"+store.auth.Password)) {
		t.Fatal("wrong credential encoding or host")
	}
	client.ClearActions()
	output, err = prepare(m, string(output.Manifest))
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() == "create" || action.GetVerb() == "update" {
			t.Fatal("unchanged retry should not write")
		}
	}
	objects, _ = workflow.ValidateYAMLManifest(output.Manifest, "risk-qa")
	refs, _, _ = unstructured.NestedSlice(objects[0].Object, "spec", "template", "spec", "imagePullSecrets")
	if len(refs) != 2 {
		t.Fatal("duplicate ref")
	}
	store.auth.Password = "rotated-private-password"
	if _, err = prepare(m, manifest); err != nil {
		t.Fatal(err)
	}
	rotated, _ := client.CoreV1().Secrets("risk-qa").Get(context.Background(), secret.Name, metav1.GetOptions{})
	if string(rotated.Data[corev1.DockerConfigJsonKey]) == string(secret.Data[corev1.DockerConfigJsonKey]) {
		t.Fatal("credential rotation not saved")
	}
	other, _ := client.CoreV1().Secrets("production").List(context.Background(), metav1.ListOptions{})
	if len(other.Items) != 0 {
		t.Fatal("credential leaked to other environment")
	}
}

func TestHelmPostRendererPreservesChartSecretsPVCAndPullReferences(t *testing.T) {
	m, client, store := setup(t)
	input := manifest + `---
apiVersion: v1
kind: Secret
metadata:
  name: application-secret
stringData:
  token: chart-managed-value
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: application-data
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi
`
	raw, err := m.PrepareHelmDeployment(context.Background(), workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 17}, workflow.HelmDeployConfig{ServiceID: 17}, workflow.HelmTarget{ClusterID: 2, Namespace: "risk-qa"}, []byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), store.auth.Password) {
		t.Fatal("registry credential leaked into Helm history")
	}
	objects, err := workflow.ValidateHelmManifest(raw, "risk-qa")
	if err != nil || len(objects) != 3 {
		t.Fatal("Chart resource lost", err)
	}
	if objects[1].GetKind() != "Secret" || objects[2].GetKind() != "PersistentVolumeClaim" {
		t.Fatal("Chart resources changed")
	}
	refs, _, _ := unstructured.NestedSlice(objects[0].Object, "spec", "template", "spec", "imagePullSecrets")
	if len(refs) != 2 {
		t.Fatal("pull refs missing from Helm post-render")
	}
	secrets, err := client.CoreV1().Secrets("risk-qa").List(context.Background(), metav1.ListOptions{})
	if err != nil || len(secrets.Items) != 1 {
		t.Fatal("managed pull secret missing", err)
	}
}

func TestPublicHelmImageStillProvisionsTargetNamespace(t *testing.T) {
	m, client, store := setup(t)
	store.auth = nil
	if err := client.CoreV1().Namespaces().Delete(context.Background(), "risk-qa", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	_, err := m.PrepareHelmDeployment(context.Background(), workflow.WorkflowRun{ProjectID: 19, EnvironmentID: 17}, workflow.HelmDeployConfig{ServiceID: 17}, workflow.HelmTarget{ClusterID: 2, Namespace: "risk-qa"}, []byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "risk-qa", metav1.GetOptions{}); err != nil {
		t.Fatal("public image did not initialize namespace", err)
	}
	secrets, err := client.CoreV1().Secrets("risk-qa").List(context.Background(), metav1.ListOptions{})
	if err != nil || len(secrets.Items) != 0 {
		t.Fatal("public image created unnecessary credentials")
	}
}

func TestUnmanagedSecretIsNotOverwritten(t *testing.T) {
	m, client, store := setup(t)
	if _, err := prepare(m, manifest); err != nil {
		t.Fatal(err)
	}
	list, _ := client.CoreV1().Secrets("risk-qa").List(context.Background(), metav1.ListOptions{})
	secret := list.Items[0]
	secret.Labels = nil
	if _, err := client.CoreV1().Secrets("risk-qa").Update(context.Background(), &secret, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	store.auth.Password = "new-password"
	if _, err := prepare(m, manifest); err == nil || strings.Contains(err.Error(), store.auth.Password) {
		t.Fatalf("expected safe ownership error: %v", err)
	}
	saved, _ := client.CoreV1().Secrets("risk-qa").Get(context.Background(), secret.Name, metav1.GetOptions{})
	if string(saved.Data[corev1.DockerConfigJsonKey]) != string(secret.Data[corev1.DockerConfigJsonKey]) {
		t.Fatal("overwrote user secret")
	}
}

func TestInvalidCredentialOrRepositoryFailsBeforeAnyClusterWrites(t *testing.T) {
	for _, scenario := range []string{"namespace", "token-only", "wrong-target-namespace"} {
		t.Run(scenario, func(t *testing.T) {
			m, client, store := setup(t)
			input := manifest
			switch scenario {
			case "namespace":
				input = strings.ReplaceAll(input, "registry.example/sinozo/", "registry.example/other/")
			case "token-only":
				store.auth.Username = ""
				store.auth.Password = ""
				store.auth.Token = "secret-token"
			case "wrong-target-namespace":
				input = strings.Replace(input, "name: risk-server", "name: risk-server\n  namespace: production", 1)
			}
			if _, err := prepare(m, input); err == nil {
				t.Fatal("expected refusal")
			}
			if len(client.Actions()) != 0 {
				t.Fatal("cluster changed before validation")
			}
		})
	}
}

func TestAnonymousAndUnrelatedImagesNeedNoSecret(t *testing.T) {
	for _, scenario := range []string{"unbound", "anonymous", "unrelated"} {
		t.Run(scenario, func(t *testing.T) {
			m, client, store := setup(t)
			input := manifest
			switch scenario {
			case "unbound":
				store.auth = nil
			case "anonymous":
				store.auth.Username = ""
				store.auth.Password = ""
			case "unrelated":
				input = strings.ReplaceAll(input, "registry.example/sinozo/", "public.example/sinozo/")
			}
			out, err := prepare(m, input)
			if err != nil {
				t.Fatal(err)
			}
			if string(out.Manifest) != input || len(client.Actions()) != 0 {
				t.Fatal("unexpected credential injection")
			}
		})
	}
}

func TestInitContainerAndCronJobTemplatesReceiveCredentials(t *testing.T) {
	m, client, _ := setup(t)
	input := `apiVersion: batch/v1
kind: CronJob
metadata:
  name: scheduled
spec:
  jobTemplate:
    spec:
      template:
        spec:
          initContainers:
          - name: init
            image: registry.example/sinozo/init:v1
          containers:
          - name: app
            image: public.example/app:v1
`
	if err := client.CoreV1().Namespaces().Delete(context.Background(), "risk-qa", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	out, err := prepare(m, input)
	if err != nil {
		t.Fatal(err)
	}
	objects, _ := workflow.ValidateYAMLManifest(out.Manifest, "risk-qa")
	refs, _, _ := unstructured.NestedSlice(objects[0].Object, "spec", "jobTemplate", "spec", "template", "spec", "imagePullSecrets")
	if len(refs) != 1 {
		t.Fatal("CronJob init-container missing pull reference")
	}
	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "risk-qa", metav1.GetOptions{}); err != nil {
		t.Fatal(err)
	}
}
