package templatelib

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

func renderWorkload(t *testing.T, values, credentials string) ([]unstructured.Unstructured, error) {
	return renderBuiltinChart(t, "helm-workload-v1", values, credentials)
}

func renderBuiltinChart(t *testing.T, key, values, credentials string) ([]unstructured.Unstructured, error) {
	t.Helper()
	var template Template
	for _, item := range Defaults() {
		if item.BuiltinKey == key {
			template = item
		}
	}
	definition, err := Snapshot(template)
	if err != nil {
		return nil, err
	}
	chart, err := helmconfig.LoadChart([]byte(definition))
	if err != nil {
		return nil, err
	}
	plain, err := helmconfig.ParseValues(values)
	if err != nil {
		return nil, err
	}
	bound, err := helmconfig.ParseValues(credentials)
	if err != nil {
		return nil, err
	}
	runtime := workflow.MergeHelmValues(plain, bound)
	if err = helmconfig.ValidateWorkloadValues(chart, runtime, plain); err != nil {
		return nil, err
	}
	renderValues, err := chartutil.ToRenderValues(chart, runtime, chartutil.ReleaseOptions{Name: "gateway", Namespace: "qa", Revision: 7, IsUpgrade: true}, nil)
	if err != nil {
		return nil, err
	}
	files, err := engine.Render(chart, renderValues)
	if err != nil {
		return nil, err
	}
	var manifests []string
	for path, content := range files {
		if !strings.HasSuffix(path, ".tpl") && strings.TrimSpace(content) != "" {
			manifests = append(manifests, content)
		}
	}
	resources, err := workflow.ValidateHelmManifest([]byte(strings.Join(manifests, "\n---\n")), "qa")
	if err != nil {
		return nil, err
	}
	decoder := serializer.NewCodecFactory(scheme.Scheme, serializer.EnableStrict).UniversalDeserializer()
	for _, r := range resources {
		raw, _ := json.Marshal(r.Object)
		if _, _, err = decoder.Decode(raw, nil, nil); err != nil {
			return nil, fmt.Errorf("strict %s: %w; manifest: %s", r.GetKind(), err, raw)
		}
	}
	return resources, nil
}

func TestUniversalChartControllersAndNativeKubernetesTypes(t *testing.T) {
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"} {
		t.Run(kind, func(t *testing.T) {
			resources, err := renderWorkload(t, "workload: {kind: "+kind+"}\nservice: {port: 9099, targetPort: 28080}\n", "")
			if err != nil {
				t.Fatal(err)
			}
			counts := map[string]int{}
			var workload unstructured.Unstructured
			for _, r := range resources {
				counts[r.GetKind()]++
				if r.GetKind() == kind {
					workload = r
				}
			}
			if counts[kind] != 1 {
				t.Fatalf("missing workload %s", kind)
			}
			path := []string{"spec", "template", "spec"}
			if kind == "CronJob" {
				path = []string{"spec", "jobTemplate", "spec", "template", "spec"}
			}
			containers, _, _ := unstructured.NestedSlice(workload.Object, append(path, "containers")...)
			if len(containers) != 1 {
				t.Fatal("application container missing")
			}
			c := containers[0].(map[string]interface{})
			if _, exists := c["readinessProbe"]; exists {
				t.Fatal("probes enabled without selection")
			}
			ports := c["ports"].([]interface{})
			if ports[0].(map[string]interface{})["containerPort"] != int64(28080) {
				t.Fatal("container port ignored")
			}
			if kind == "Job" {
				if workload.GetName() != "gateway-r7" {
					t.Fatal("Job must use a new name per release revision")
				}
			}
			if kind == "Job" || kind == "CronJob" {
				if counts["Service"] != 0 {
					t.Fatal("batch workload exposed a Service")
				}
			} else if counts["Service"] == 0 {
				t.Fatal("network Service missing")
			}
		})
	}
}

func TestUniversalChartOptionalResourcesAndALBFollowService(t *testing.T) {
	values := `replicaCount: 2
service: {port: 9099, targetPort: 28080}
ingress:
  enabled: true
  className: alb
  labels: {ingress-controller: alb}
  annotations: {alb.ingress.kubernetes.io/listen-ports: '[{"HTTP":80},{"HTTPS":443}]'}
  hosts: [{host: gateway.example.com, paths: [{path: /, pathType: Prefix}, {path: /api, pathType: Exact}, {path: /app/(.*), pathType: ImplementationSpecific}]}]
  tls: [{hosts: [gateway.example.com]}]
autoscaling: {enabled: true, minReplicas: 2, maxReplicas: 5}
pdb: {enabled: true, maxUnavailable: "25%"}
persistence: {enabled: true, accessModes: [ReadWriteMany], size: 20Gi, storageClass: ""}
configMaps:
  app: {data: {application.yaml: "server:\n  port: 28080\n"}, mountPath: /app/config}
secrets: {auth: {type: Opaque}}
secretMounts: [{name: gateway-auth, mountPath: /run/secrets, items: [{key: SK, path: sk}]}]
probes: {readiness: {tcpSocket: {port: http}}, liveness: {httpGet: {path: /health, port: 28080}}, startup: {exec: {command: [sh, -c, "test -f /tmp/ready"]}}}
monitoring: {enabled: true, port: 28080}
serviceAccount: {create: true}
rbac: {create: true, rules: [{apiGroups: [""], resources: [configmaps], verbs: [get]}]}
networkPolicy: {enabled: true, policyTypes: [Ingress, Egress], ingress: [{from: [{podSelector: {}}]}], egress: [{}]}
env: [{name: PASSWORD, valueFrom: {secretKeyRef: {name: gateway-auth, key: password}}}]
volumes: [{name: cache, emptyDir: {}}]
volumeMounts: [{name: cache, mountPath: /cache}]
`
	resources, err := renderWorkload(t, values, "secrets: {auth: {stringData: {password: 'private-test-value', SK: 'secret-key'}}}\n")
	if err != nil {
		t.Fatal(err)
	}
	byKind := map[string]unstructured.Unstructured{}
	for _, r := range resources {
		byKind[r.GetKind()] = r
	}
	for _, kind := range []string{"Deployment", "Service", "Ingress", "ConfigMap", "Secret", "PersistentVolumeClaim", "HorizontalPodAutoscaler", "PodDisruptionBudget", "ServiceAccount", "Role", "RoleBinding", "NetworkPolicy"} {
		if _, exists := byKind[kind]; !exists {
			t.Fatalf("missing %s", kind)
		}
	}
	d := byKind["Deployment"]
	if _, found, _ := unstructured.NestedFieldNoCopy(d.Object, "spec", "replicas"); found {
		t.Fatal("HPA replica ownership overwritten")
	}
	in := byKind["Ingress"]
	rules, _, _ := unstructured.NestedSlice(in.Object, "spec", "rules")
	paths, _, _ := unstructured.NestedSlice(rules[0].(map[string]interface{}), "http", "paths")
	for _, path := range paths {
		backend := path.(map[string]interface{})["backend"].(map[string]interface{})["service"].(map[string]interface{})
		if backend["name"] != "gateway" || backend["port"].(map[string]interface{})["number"] != int64(9099) {
			t.Fatal("ALB did not follow Service name/port")
		}
	}
	claim := byKind["PersistentVolumeClaim"]
	if claim.GetAnnotations()["helm.sh/resource-policy"] != "keep" {
		t.Fatal("PVC data retention not defaulted")
	}
	secret := byKind["Secret"]
	if fields, _, _ := unstructured.NestedStringMap(secret.Object, "stringData"); len(fields) != 2 {
		t.Fatal("multiple secret keys lost")
	}
}

func TestUniversalChartStatefulClaimsAndTLSKeys(t *testing.T) {
	resources, err := renderWorkload(t, "workload: {kind: StatefulSet}\nreplicaCount: 3\npersistence: {enabled: true}\nsecrets: {tls: {type: kubernetes.io/tls, fields: {cert: {key: tls.crt}, private: {key: tls.key}}}}\n", "secrets: {tls: {fields: {cert: {value: cert-placeholder}, private: {value: key-placeholder}}}}\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range resources {
		if r.GetKind() == "PersistentVolumeClaim" {
			t.Fatal("StatefulSet must use per-replica claims")
		}
		if r.GetKind() == "StatefulSet" {
			claims, _, _ := unstructured.NestedSlice(r.Object, "spec", "volumeClaimTemplates")
			if len(claims) != 1 {
				t.Fatal("missing per-replica claim")
			}
		}
	}
}

func TestUniversalChartRejectsInvalidAndIncompatibleValues(t *testing.T) {
	for name, values := range map[string]string{
		"unknown controller": "workload: {kind: ReplicaSet}", "invalid memory": "resources: {limits: {memory: '2'}}", "request over limit": "resources: {requests: {memory: 1Gi}, limits: {memory: 512Mi}}",
		"bad port": "service: {targetPort: 70000}", "bad env type": "env: [{name: PORT, value: 8080}]", "dual probe": "probes: {readiness: {httpGet: {port: 8080, path: /}, tcpSocket: {port: 8080}}}",
		"invalid liveness": "probes: {liveness: {tcpSocket: {port: 8080}, successThreshold: 2}}", "invalid cron": "workload: {kind: CronJob}\ncronJob: {schedule: every-minute}",
		"batch HPA": "workload: {kind: Job}\nautoscaling: {enabled: true}", "inverted HPA": "autoscaling: {enabled: true, minReplicas: 5, maxReplicas: 2}",
		"shared RWO": "replicaCount: 2\npersistence: {enabled: true}", "HPA shared RWO": "persistence: {enabled: true}\nautoscaling: {enabled: true}", "batch ingress": "workload: {kind: Job}\ningress: {enabled: true}",
		"plaintext Secret": "secrets: {auth: {stringData: {password: password}}}", "plaintext field": "secrets: {tls: {fields: {cert: {key: tls.crt, value: plaintext}}}}",
		"label collision": "podLabels: {app.kubernetes.io/instance: other}", "container collision": "sidecars: [{name: application, image: busybox}]",
		"bad pathType": "ingress: {enabled: true, hosts: [{host: a.example.com, paths: [{path: /, pathType: Wildcard}]}]}", "RBAC missing SA": "rbac: {create: true}",
		"invalid class":    "ingress: {className: 'not a class'}",
		"IP ingress host":  "ingress: {enabled: true, hosts: [{host: '192.168.1.1', paths: [{path: /, pathType: Prefix}]}]}",
		"unresolved mount": "volumeMounts: [{name: missing, mountPath: /config}]",
		"duplicate mount":  "persistence: {enabled: true}\nconfigMaps: {app: {data: {a: b}, mountPath: /data}}",
		"duplicate volume": "persistence: {enabled: true}\nvolumes: [{name: data, emptyDir: {}}]",
		"zero HPA request": "autoscaling: {enabled: true}\nresources: {requests: {cpu: '0'}}",
		"RWOP combination": "persistence: {enabled: true, accessModes: [ReadWriteOncePod, ReadWriteMany]}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := renderWorkload(t, values, ""); err == nil {
				t.Fatal("invalid values accepted")
			}
		})
	}
}
