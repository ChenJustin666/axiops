package serviceassets

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"
)

func TestRuntimeResourcesAreIndependentAndPreserveTemplate(t *testing.T) {
	base := baseManifest()
	containers, _, _ := unstructured.NestedSlice(base[0].Object, "spec", "template", "spec", "containers")
	containers[0].(map[string]interface{})["resources"] = map[string]interface{}{"requests": map[string]interface{}{"cpu": "100m", "memory": "256Mi"}, "limits": map[string]interface{}{"memory": "512Mi", "ephemeral-storage": "1Gi"}}
	unstructured.SetNestedSlice(base[0].Object, containers, "spec", "template", "spec", "containers")
	original, _ := json.Marshal(base)
	for _, memory := range []string{"1Gi", "4Gi"} {
		settings := Empty()
		settings.Resources = &RuntimeResources{MemoryLimit: memory}
		out, _, err := Apply(base, settings, 1, "app")
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(out)
		for _, want := range []string{`"memory":"` + memory + `"`, `"memory":"256Mi"`, `"cpu":"100m"`, `"ephemeral-storage":"1Gi"`} {
			if !strings.Contains(string(raw), want) {
				t.Fatalf("missing merged resource %s: %s", want, raw)
			}
		}
	}
	after, _ := json.Marshal(base)
	if string(after) != string(original) {
		t.Fatal("environment override mutated shared input")
	}
	settings := Empty()
	settings.Resources = &RuntimeResources{MemoryLimit: "128Mi"}
	if _, _, err := Apply(base, settings, 1, "app"); err == nil {
		t.Fatal("accepted limit below inherited request")
	}
	for _, bad := range []RuntimeResources{{MemoryLimit: "1g"}, {MemoryLimit: "1"}, {MemoryLimit: "-1Gi"}, {CPULimit: "0.0001"}, {CPURequest: "2", CPULimit: "1"}} {
		if bad.Validate() == nil {
			t.Fatalf("accepted invalid resource %+v", bad)
		}
	}
	restored, _, err := Apply(base, Empty(), 1, "app")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(restored)
	if string(raw) != string(original) {
		t.Fatal("removing override did not restore YAML")
	}
}
