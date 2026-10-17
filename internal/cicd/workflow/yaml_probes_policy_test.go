package workflow

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestProbePolicyAndPortFollow(t *testing.T) {
	definition := []byte(`apiVersion: apps/v1
kind: Deployment
metadata: {name: api}
spec:
  template:
    spec:
      containers:
      - name: api
        image: registry/api:v1
        ports: [{containerPort: 28080}]
`)
	for _, mode := range []string{"none", "tcp", "http"} {
		for _, active := range []string{"readiness", "startup", "liveness", "all", "off", "legacy"} {
			t.Run(mode+"/"+active, func(t *testing.T) {
				v := map[string]string{"healthCheckType": mode, "healthPath": "/ready", "healthPort": "0", "port": "28080"}
				if active != "legacy" {
					for _, key := range []string{"Readiness", "Startup", "Liveness"} {
						v["probe"+key+"Enabled"] = "false"
						if active == strings.ToLower(key) || active == "all" {
							v["probe"+key+"Enabled"] = "true"
						}
					}
				}
				if mode == "tcp" {
					v["healthPath"] = ""
				}
				out, err := applyStandardProbes(definition, v)
				if err != nil {
					t.Fatal(err)
				}
				items, err := ParseYAMLManifest(out)
				if err != nil {
					t.Fatal(err)
				}
				containers, _, _ := unstructured.NestedSlice(items[0].Object, "spec", "template", "spec", "containers")
				c := containers[0].(map[string]interface{})
				for _, name := range []string{"readiness", "startup", "liveness"} {
					p, exists := c[name+"Probe"]
					want := mode != "none" && (active == name || active == "all" || active == "legacy")
					if exists != want {
						t.Fatalf("%s exists=%t want=%t: %s", name, exists, want, out)
					}
					if exists {
						handler := "tcpSocket"
						if mode == "http" {
							handler = "httpGet"
						}
						var probe corev1.Probe
						if err := runtime.DefaultUnstructuredConverter.FromUnstructured(p.(map[string]interface{}), &probe); err != nil {
							t.Fatal(err)
						}
						port := probe.TCPSocket
						correct := port != nil && port.Port.IntValue() == 28080
						if handler == "httpGet" {
							correct = probe.HTTPGet != nil && probe.HTTPGet.Port.IntValue() == 28080
						}
						if !correct {
							t.Fatal("probe ignored application port")
						}
					}
				}
			})
		}
	}
	native := append(append([]byte{}, definition...), []byte("        readinessProbe:\n          httpGet: {path: /native-ready, port: 9000}\n")...)
	for _, mode := range []string{"none", "tcp"} {
		out, e := applyStandardProbes(native, map[string]string{"healthCheckType": mode, "port": "28080", "probeReadinessEnabled": "false"})
		if e != nil || !strings.Contains(string(out), "/native-ready") {
			t.Fatal("native probe was overwritten", e)
		}
	}
	for _, values := range []map[string]string{
		{"healthCheckType": "tcp", "port": "28080", "probeLivenessEnabled": "maybe"},
		{"healthCheckType": "http", "port": "28080", "healthPath": "bad"},
		{"healthCheckType": "tcp", "port": "65536"},
	} {
		if _, e := applyStandardProbes(definition, values); e == nil {
			t.Fatal("invalid probe accepted")
		}
	}
}
