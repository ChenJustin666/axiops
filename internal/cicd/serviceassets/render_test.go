package serviceassets

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"
)

func baseManifest() []unstructured.Unstructured {
	return []unstructured.Unstructured{
		{Object: map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "risk-server"}, "spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{}, "spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"name": "risk-server", "image": "example/app"}}}}}}},
		{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "risk-server"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": int64(9099)}}}}},
	}
}

func TestApplyGeneratesAssetsWithoutPlaintextSecret(t *testing.T) {
	s := Empty()
	s.Files = []File{{Name: "config", Filename: "application.yml", Content: "port: 9099", MountPath: "/etc/app/application.yml"}}
	s.Secrets = []Secret{{Name: "db", CredentialID: 42, CredentialKey: "password", Key: "password", EnvName: "DB_PASSWORD"}}
	s.Routes = []Route{{Name: "api", Host: "api.example.com", Path: "/", Service: "risk-server", Port: 9099, Class: "nginx"}}
	s.Volumes = []Volume{{Name: "data", Capacity: "1Gi", StorageClass: "standard", AccessMode: "ReadWriteOnce", MountPath: "/data"}}
	out, refs, err := Apply(baseManifest(), s, 17, "risk-test")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].CredentialID != 42 {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	foundCM, foundPVC, foundIngress, foundSecretRef := false, false, false, false
	for _, item := range out {
		switch item.GetKind() {
		case "ConfigMap":
			foundCM = true
		case "PersistentVolumeClaim":
			foundPVC = true
		case "Ingress":
			foundIngress = true
		case "Secret":
			foundSecretRef = true
		}
		if item.GetKind() == "Deployment" {
			encoded, _ := json.Marshal(item.Object)
			text := string(encoded)
			if !strings.Contains(text, "axiops-s17-secret-db") {
				t.Fatal("secret reference missing")
			}
			if strings.Contains(text, `"value":"`) {
				t.Fatal("plaintext secret value present")
			}
		}
	}
	if !foundCM || !foundPVC || !foundIngress || foundSecretRef {
		t.Fatalf("generated resources cm=%v pvc=%v ingress=%v secret=%v", foundCM, foundPVC, foundIngress, foundSecretRef)
	}
}

func TestPVCDefaultStorageClassIsOmittedAndExplicitClassIsPreserved(t *testing.T) {
	for _, class := range []string{"", "standard"} {
		s := Empty()
		s.Volumes = []Volume{{Name: "data", Capacity: "1Gi", StorageClass: class, AccessMode: "ReadWriteOnce", MountPath: "/data"}}
		objects, _, err := Apply(baseManifest(), s, 17, "test")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range objects {
			if item.GetKind() != "PersistentVolumeClaim" {
				continue
			}
			found = true
			value, exists, err := unstructured.NestedString(item.Object, "spec", "storageClassName")
			if err != nil || exists != (class != "") || value != class {
				t.Fatalf("wrong storage class: %q exists=%v", value, exists)
			}
		}
		if !found {
			t.Fatal("PVC not rendered")
		}
	}
}

func TestValidateRejectsConflictsAndLiteralTemplateContent(t *testing.T) {
	s := Empty()
	s.Files = []File{{Name: "a", Filename: "a", Content: "{{.Password}}", MountPath: "/etc/a"}, {Name: "b", Filename: "b", MountPath: "/etc/a/sub"}}
	if err := Validate(s); err == nil {
		t.Fatal("expected mount conflict")
	}
	s = Empty()
	s.Secrets = []Secret{{Name: "bad", CredentialID: 1, CredentialKey: "password", Key: "x", EnvName: "BAD-NAME"}}
	if err := Validate(s); err == nil {
		t.Fatal("expected env name validation")
	}
}

func TestApplyRequiresExplicitTargetsForMultipleWorkloads(t *testing.T) {
	in := baseManifest()
	in = append(in, *baseManifest()[0].DeepCopy())
	in[2].Object["metadata"].(map[string]interface{})["name"] = "other"
	s := Empty()
	s.Files = []File{{Name: "a", Filename: "a", Content: "x", MountPath: "/etc/a"}}
	if _, _, err := Apply(in, s, 1, "ns"); err == nil {
		t.Fatal("expected explicit workload error")
	}
}

func TestCredentialReplacementUsesIndependentSecretAndRollsOnlyItsConsumer(t *testing.T) {
	s := Empty()
	s.Secrets = []Secret{{Name: "db", CredentialID: 42, CredentialKey: "password", Key: "password", EnvName: "DB_PASSWORD"}}
	before, oldRefs, err := Apply(baseManifest(), s, 17, "risk-test")
	if err != nil {
		t.Fatal(err)
	}
	s.Secrets[0].CredentialID = 43
	after, newRefs, err := Apply(baseManifest(), s, 17, "risk-test")
	if err != nil {
		t.Fatal(err)
	}
	if oldRefs[0].Name == newRefs[0].Name {
		t.Fatal("replacement would overwrite the old Secret")
	}
	oldJSON, _ := json.Marshal(before[0].Object)
	newJSON, _ := json.Marshal(after[0].Object)
	if !strings.Contains(string(oldJSON), oldRefs[0].Name) || !strings.Contains(string(newJSON), newRefs[0].Name) {
		t.Fatal("workload does not reference its credential version")
	}
	_, repeatRefs, err := Apply(baseManifest(), s, 17, "risk-test")
	if err != nil || repeatRefs[0].Name != newRefs[0].Name {
		t.Fatal("same credential must reuse the same Secret")
	}
	_, otherRefs, err := Apply(baseManifest(), s, 18, "risk-test")
	if err != nil || otherRefs[0].Name == newRefs[0].Name {
		t.Fatal("services must not share generated Secret names")
	}
	s.Secrets[0].CredentialID = 0
	s.Secrets[0].ExistingSecret = "external-db"
	external, refs, err := Apply(baseManifest(), s, 17, "risk-prod")
	data, _ := json.Marshal(external)
	if err != nil || len(refs) != 0 || !strings.Contains(string(data), "external-db") {
		t.Fatal("existing external Secret reference must remain unchanged")
	}
}

func TestApplyEnvironmentQuickSettingsOverrideWorkload(t *testing.T) {
	in := baseManifest()
	unstructured.SetNestedField(in[0].Object, int64(1), "spec", "replicas")
	s := Empty()
	replicas := 3
	s.Replicas = &replicas
	s.Command = []string{"java"}
	s.Args = []string{"-jar", "app.jar"}
	out, _, err := Apply(in, s, 1, "app")
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := unstructured.NestedInt64(out[0].Object, "spec", "replicas")
	if err != nil || !found || got != 3 {
		t.Fatalf("replicas=%d found=%v err=%v", got, found, err)
	}
	containers, _, _ := unstructured.NestedSlice(out[0].Object, "spec", "template", "spec", "containers")
	container := containers[0].(map[string]interface{})
	if values, _, _ := unstructured.NestedStringSlice(container, "command"); len(values) != 1 || values[0] != "java" {
		t.Fatalf("command=%v", values)
	}
	if values, _, _ := unstructured.NestedStringSlice(container, "args"); len(values) != 2 || values[1] != "app.jar" {
		t.Fatalf("args=%v", values)
	}
}

func TestMultiKeySecretIsOneResourceWithStableVersionAndIsolatedConsumers(t *testing.T) {
	s := Empty()
	s.Secrets = []Secret{
		{Name: "sk", Group: "auth", CredentialID: 42, CredentialKey: "SK", Key: "SK", EnvName: "API_SK"},
		{Name: "tls", Group: "auth", CredentialID: 42, CredentialKey: "tls.key", Key: "tls.key", MountPath: "/secrets/tls.key"},
	}
	_, refs, err := Apply(baseManifest(), s, 17, "test")
	if err != nil || len(refs) != 2 || refs[0].Name != refs[1].Name {
		t.Fatalf("group is not one Secret: %v", err)
	}
	old := refs[0].Name
	s.Secrets[0], s.Secrets[1] = s.Secrets[1], s.Secrets[0]
	_, reordered, err := Apply(baseManifest(), s, 17, "test")
	if err != nil || reordered[0].Name != old {
		t.Fatal("reordering changed version")
	}
	s.Secrets = s.Secrets[:1]
	_, removed, err := Apply(baseManifest(), s, 17, "test")
	if err != nil || removed[0].Name == old {
		t.Fatal("removed key would change previous Secret")
	}
	s.Secrets[0].CredentialID = 43
	_, replaced, err := Apply(baseManifest(), s, 17, "test")
	if err != nil || replaced[0].Name == removed[0].Name {
		t.Fatal("new credential reused old name")
	}
	_, other, err := Apply(baseManifest(), s, 18, "test")
	if err != nil || replaced[0].Name == other[0].Name {
		t.Fatal("service isolation lost")
	}
}

func TestSecretGroupRejectsMixedSourcesAndDuplicateKeys(t *testing.T) {
	s := Empty()
	s.Secrets = []Secret{{Name: "a", Group: "auth", CredentialID: 42, CredentialKey: "token", Key: "token", EnvName: "A"}, {Name: "b", Group: "auth", CredentialID: 43, CredentialKey: "SK", Key: "SK", EnvName: "B"}}
	if Validate(s) == nil {
		t.Fatal("mixed credentials accepted")
	}
	s.Secrets[1].CredentialID = 42
	s.Secrets[1].Key = "token"
	if Validate(s) == nil {
		t.Fatal("duplicate grouped keys accepted")
	}
	s.Secrets[1].Key = "SK"
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
}
