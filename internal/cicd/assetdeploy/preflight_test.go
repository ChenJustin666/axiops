package assetdeploy

import (
	"context"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"strings"
	"testing"
)

const dependencyManifest = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: app:v1
        env:
        - name: SK
          valueFrom:
            secretKeyRef: {name: auth, key: SK}
        volumeMounts:
        - {name: config, mountPath: /config/app.yaml, subPath: app.yaml}
      volumes:
      - name: config
        configMap: {name: app}
      - name: data
        persistentVolumeClaim: {claimName: data}
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app
spec:
  ingressClassName: nginx
`

func TestPreflightChecksOnlyTargetNamespaceAndNeverWrites(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "prod"}, Data: map[string][]byte{"SK": []byte("do-not-print")}},
	)
	target := workflow.YAMLTarget{Namespace: "test", Manifest: []byte(dependencyManifest)}
	check := func(want string) {
		t.Helper()
		err := preflightDependencies(context.Background(), client, target)
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "do-not-print") {
			t.Fatalf("expected %s, got %v", want, err)
		}
	}
	check("Secret auth")
	client.Tracker().Add(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "test"}, Data: map[string][]byte{"other": []byte("do-not-print")}})
	check("缺少键 SK")
	client.CoreV1().Secrets("test").Update(context.Background(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "auth", Namespace: "test"}, Data: map[string][]byte{"SK": []byte("do-not-print")}}, metav1.UpdateOptions{})
	check("ConfigMap app")
	client.Tracker().Add(&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "test"}})
	check("缺少键 app.yaml")
	target.Manifest = []byte(dependencyManifest + "\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata:\n  app.yaml: config\n")
	check("PVC data")
	client.Tracker().Add(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "test"}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}})
	check("IngressClass nginx")
	client.Tracker().Add(&networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "nginx"}})
	client.ClearActions()
	if err := preflightDependencies(context.Background(), client, target); err != nil {
		t.Fatal(err)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() != "get" {
			t.Fatalf("preflight wrote: %s", action.GetVerb())
		}
	}
}

func TestPreflightOptionalSecretAndTLSKeys(t *testing.T) {
	client := fake.NewSimpleClientset()
	target := workflow.YAMLTarget{Namespace: "test", Manifest: []byte(strings.ReplaceAll(dependencyManifest, "name: auth, key: SK", "name: auth, key: SK, optional: true"))}
	err := preflightDependencies(context.Background(), client, target)
	if err == nil || !strings.Contains(err.Error(), "ConfigMap app") {
		t.Fatalf("optional secret blocked: %v", err)
	}
	target.Manifest = []byte("apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: app\nspec:\n  tls:\n  - secretName: tls\n")
	client.Tracker().Add(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "tls", Namespace: "test"}, Data: map[string][]byte{"tls.crt": []byte("certificate")}})
	err = preflightDependencies(context.Background(), client, target)
	if err == nil || !strings.Contains(err.Error(), "tls.key") {
		t.Fatalf("TLS missing key accepted: %v", err)
	}
}

func TestPreflightRejectsPreviewImageAndNacosPlaceholders(t *testing.T) {
	for _, body := range []string{
		"image: axiops-config-validation.invalid/pending-image:validation",
		"image: app:v1\n        env:\n        - name: SPRING_CLOUD_NACOS_CONFIG_SERVERADDR\n          value: nacos.example.invalid:8848",
	} {
		manifest := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app\nspec:\n  template:\n    spec:\n      containers:\n      - name: app\n        " + body
		err := preflightDependencies(context.Background(), fake.NewSimpleClientset(), workflow.YAMLTarget{Namespace: "test", Manifest: []byte(manifest)})
		if err == nil {
			t.Fatal("placeholder reached deployment")
		}
	}
}

func TestPreflightAcceptsGeneratedSecretAndInlineClaim(t *testing.T) {
	manifest := strings.Split(dependencyManifest, "---")[0] + `
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: app
data:
  app.yaml: config
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi
`
	client := fake.NewSimpleClientset()
	target := workflow.YAMLTarget{Namespace: "test", Manifest: []byte(manifest), AssetSecrets: []serviceassets.SecretRef{{Name: "auth", Key: "SK", CredentialID: 42, CredentialKey: "SK"}}}
	if err := preflightDependencies(context.Background(), client, target); err != nil {
		t.Fatal(err)
	}
	if len(client.Actions()) != 0 {
		t.Fatal("inline dependencies should require no cluster writes or reads")
	}
}
