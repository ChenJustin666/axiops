package templatelib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func javaResources(t *testing.T, values string) (appsv1.Deployment, []corev1.Service, []networkingv1.Ingress) {
	t.Helper()
	resources, err := renderBuiltinChart(t, "helm-sinozo-java-v1", values, "")
	if err != nil {
		t.Fatal(err)
	}
	var deployment appsv1.Deployment
	var services []corev1.Service
	var ingresses []networkingv1.Ingress
	for _, r := range resources {
		raw, _ := json.Marshal(r.Object)
		switch r.GetKind() {
		case "Deployment":
			if err := json.Unmarshal(raw, &deployment); err != nil {
				t.Fatal(err)
			}
		case "Service":
			var s corev1.Service
			_ = json.Unmarshal(raw, &s)
			services = append(services, s)
		case "Ingress":
			var i networkingv1.Ingress
			_ = json.Unmarshal(raw, &i)
			ingresses = append(ingresses, i)
		}
	}
	return deployment, services, ingresses
}

func envMap(container corev1.Container) map[string]corev1.EnvVar {
	result := map[string]corev1.EnvVar{}
	for _, value := range container.Env {
		result[value.Name] = value
	}
	return result
}

func TestJavaChartDefaultPortMemoryAndNoOptionalResources(t *testing.T) {
	d, services, ingresses := javaResources(t, "")
	c := d.Spec.Template.Spec.Containers[0]
	e := envMap(c)
	if len(services) != 1 || len(ingresses) != 0 || services[0].Name != "gateway-svc" || services[0].Spec.Ports[0].Port != 28080 || services[0].Spec.Ports[0].TargetPort.StrVal != "http" || c.Ports[0].ContainerPort != 28080 || *d.Spec.Replicas != 1 {
		t.Fatal("defaults or Service wiring wrong")
	}
	if e["SERVER_PORT"].Value != "28080" || e["SPRING_APPLICATION_NAME"].Value != "gateway" || !strings.Contains(e["JAVA_TOOL_OPTIONS"].Value, "MaxRAMPercentage=50.0") || strings.Contains(e["JAVA_TOOL_OPTIONS"].Value, "\n") || c.Resources.Limits.Memory().String() != "512Mi" {
		t.Fatal("Java parameters or resources wrong")
	}
	if c.StartupProbe != nil || len(d.Spec.Template.Spec.InitContainers) != 0 || len(d.Spec.Template.Spec.Volumes) != 0 {
		t.Fatal("disabled features rendered")
	}
	for name := range e {
		if strings.Contains(name, "NACOS") || strings.HasPrefix(name, "SW_") || name == "SPRING_PROFILES_ACTIVE" {
			t.Fatalf("unexpected default env %s", name)
		}
	}
}

func TestJavaChartNacosTracingAuthAndIndependentEnvironmentValues(t *testing.T) {
	d, _, _ := javaResources(t, `service: {port: 8085, replicas: 12}
application: {name: ad-puller, profile: prod}
resources: {requests: {cpu: 500m, memory: 6Gi}, limits: {cpu: "1", memory: 6Gi}}
nacos:
  enabled: true
  serverAddr: nacos-0.nacos.svc.cluster.local:8848,nacos-1.nacos.svc.cluster.local:8848
  namespace: adv_prod
  auth: {enabled: true, secretName: nacos-auth-secret, usernameKey: NACOS_USERNAME, passwordKey: NACOS_PASSWORD}
skywalking: {enabled: true, backendService: "skywalking-oap.skywalking.svc.cluster.local:11800", excludedPlugins: "asf-dubbo,dubbo"}
java: {heapDump: {enabled: true}}
probes: {enabled: true}
`)
	c := d.Spec.Template.Spec.Containers[0]
	e := envMap(c)
	if *d.Spec.Replicas != 12 || e["SPRING_PROFILES_ACTIVE"].Value != "prod" || e["SPRING_CONFIG_IMPORT"].Value != "optional:nacos:ad-puller" || e["SPRING_CLOUD_NACOS_CONFIG_NAMESPACE"].Value != "adv_prod" || e["SERVER_PORT"].Value != "8085" {
		t.Fatal("environment values did not apply")
	}
	if e["SPRING_CLOUD_NACOS_USERNAME"].ValueFrom.SecretKeyRef.Name != "nacos-auth-secret" || e["SPRING_CLOUD_NACOS_CONFIG_PASSWORD"].ValueFrom.SecretKeyRef.Key != "NACOS_PASSWORD" || e["SPRING_CLOUD_NACOS_CONFIG_PASSWORD"].ValueFrom.SecretKeyRef.Optional != nil {
		t.Fatal("authentication must use required Secret references")
	}
	if e["SW_AGENT_NAME"].Value != "ad-puller" || !strings.Contains(e["JAVA_TOOL_OPTIONS"].Value, "-javaagent:/skywalking/agent/skywalking-agent.jar") || !strings.Contains(e["JAVA_TOOL_OPTIONS"].Value, "-XX:HeapDumpPath=/dump/heapdump.hprof") || len(d.Spec.Template.Spec.InitContainers) != 1 || len(d.Spec.Template.Spec.Volumes) != 2 {
		t.Fatal("agent or dump wiring wrong")
	}
	if !strings.Contains(d.Spec.Template.Spec.InitContainers[0].Image, "@sha256:621a50951e2e9466de2b1ebcc18b63814b69d60db1492fc03337b94999a343ee") {
		t.Fatal("Agent image must stay pinned for reproducible rollback")
	}
	if c.StartupProbe.TCPSocket.Port.IntVal != 8085 || c.ReadinessProbe.TCPSocket.Port.IntVal != 8085 || c.LivenessProbe != nil {
		t.Fatal("probe defaults or linked port wrong")
	}
	// Render again without overrides; the shared template must not be mutated.
	_, svcs, _ := javaResources(t, "")
	if svcs[0].Spec.Ports[0].Port != 28080 {
		t.Fatal("previous environment changed shared template")
	}
}

func TestJavaChartALBOptionalTLSNginxAndCatchAll(t *testing.T) {
	for _, tc := range []struct {
		name, values string
		tls, secret  bool
	}{
		{"alb-http", "host: api.example.com", false, false},
		{"alb-catch-all", "host: ''", false, false},
		{"alb-cloud-certificate", "host: api.example.com\n", true, false},
		{"nginx-certificate", "host: api.example.com", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			class, tls := "alb", ""
			if tc.secret {
				class = "nginx"
			}
			if tc.tls {
				tls = "\n    tls: {enabled: true}"
				if tc.secret {
					tls = "\n    tls: {enabled: true, secretName: app-tls}"
				}
			}
			_, _, routes := javaResources(t, "service: {port: 18088}\ningress:\n  enabled: true\n  items:\n  - name: root\n    className: "+class+"\n    labels: {ingress-controller: "+class+"}\n    annotations: {alb.ingress.kubernetes.io/group.name: universal}\n    hosts:\n    - "+strings.TrimSpace(tc.values)+"\n      paths: [{path: /, pathType: Prefix}]"+tls+"\n")
			if len(routes) != 1 {
				t.Fatal("missing route")
			}
			r := routes[0]
			backend := r.Spec.Rules[0].HTTP.Paths[0].Backend.Service
			if backend.Name != "gateway-svc" || backend.Port.Number != 18088 || *r.Spec.IngressClassName != class || r.Namespace != "qa" {
				t.Fatal("route mismatches current service")
			}
			if tc.tls {
				if len(r.Spec.TLS) != 1 || r.Spec.TLS[0].Hosts[0] != "api.example.com" {
					t.Fatal("TLS hosts not derived")
				}
				if tc.secret && r.Spec.TLS[0].SecretName != "app-tls" || !tc.secret && r.Spec.TLS[0].SecretName != "" {
					t.Fatal("TLS Secret wrong")
				}
			} else if len(r.Spec.TLS) != 0 {
				t.Fatal("disabled TLS rendered")
			}
		})
	}
}

func TestJavaChartHTTPProbesMonitoringAndNativeExtensions(t *testing.T) {
	d, services, _ := javaResources(t, `service: {enabled: false, port: 9099}
java: {optionsEnv: JAVA_OPTS}
probes: {enabled: true, type: http, path: /ready, liveness: {enabled: true}}
monitoring: {enabled: true}
extraAnnotations: {prometheus.io/jvm: "true"}
env:
- name: FEISHU_WEBHOOK
  valueFrom: {secretKeyRef: {name: alert-auth, key: webhook}}
command: [java]
args: [-jar, app.jar]
volumes: [{name: app-config, configMap: {name: app-config}}]
volumeMounts: [{name: app-config, mountPath: /app/config, readOnly: true}]
affinity: {podAntiAffinity: {preferredDuringSchedulingIgnoredDuringExecution: [{weight: 100, podAffinityTerm: {labelSelector: {matchLabels: {app.kubernetes.io/instance: gateway}}, topologyKey: kubernetes.io/hostname}}]}}
`)
	c := d.Spec.Template.Spec.Containers[0]
	if len(services) != 0 || c.ReadinessProbe.HTTPGet.Port.IntVal != 9099 || c.ReadinessProbe.HTTPGet.Path != "/ready" || c.LivenessProbe == nil || d.Spec.Template.Annotations["prometheus.io/port"] != "9099" || len(c.VolumeMounts) != 1 || c.Command[0] != "java" || envMap(c)["FEISHU_WEBHOOK"].ValueFrom.SecretKeyRef.Name != "alert-auth" {
		t.Fatal("native fields, monitoring or probes wrong")
	}
}

func TestJavaChartRejectsBrokenInputs(t *testing.T) {
	for name, values := range map[string]string{
		"generated-port-env-collision": "application: {portEnv: SPRING_APPLICATION_NAME}",
		"impossible-jvm-budget":        "java: {opts: '-XX:MaxRAMPercentage=72.0 -XX:MaxMetaspaceSize=256m'}",
		"oversized-xmx":                "java: {opts: '-Xmx512m'}",
		"invalid-percentage":           "java: {opts: '-XX:MaxRAMPercentage=110'}",
		"port-overridden-by-jvm":       "java: {opts: '-Dserver.port=8080'}",
		"bad-nacos-address":            "nacos: {enabled: true, serverAddr: 'http://nacos:8848'}",
		"invalid-memory":               "resources: {limits: {memory: '2'}}",
		"request-over-limit":           "resources: {requests: {memory: 2Gi}}",
		"wrong-port":                   "service: {port: 70000}",
		"typo":                         "servcie: {port: 9099}",
		"no-nacos-address":             "nacos: {enabled: true}",
		"auth-without-nacos":           "nacos: {auth: {enabled: true}}",
		"no-auth-secret":               "nacos: {enabled: true, serverAddr: 'nacos:8848', auth: {enabled: true}}",
		"agent-without-java":           "java: {enabled: false}\nskywalking: {enabled: true, backendService: 'oap:11800'}",
		"no-oap":                       "skywalking: {enabled: true}",
		"oap-no-port":                  "skywalking: {enabled: true, backendService: oap}",
		"duplicate-env":                "application: {profile: prod}\nenv: [{name: SPRING_PROFILES_ACTIVE, value: test}]",
		"nonstring-env":                "env: [{name: CUSTOM, value: 123}]",
		"duplicate-spring-flags":       "java: {opts: '-Dspring.profiles.active=test'}",
		"bad-http-path":                "probes: {enabled: true, type: http, path: health}",
		"agent-volume-collision":       "skywalking: {enabled: true, backendService: 'oap:11800'}\nvolumes: [{name: skywalking-agent, emptyDir: {}}]",
		"missing-volume":               "volumeMounts: [{name: missing, mountPath: /data}]",
		"no-service":                   "service: {enabled: false}\ningress: {enabled: true, items: [{name: root, hosts: [{host: api.example.com, paths: [{path: /}]}]}]}",
		"empty-ingress":                "ingress: {enabled: true}",
		"invalid-host":                 "ingress: {enabled: true, items: [{name: root, hosts: [{host: 'http://api.example.com', paths: [{path: /}]}]}]}",
		"nginx-no-secret":              "ingress: {enabled: true, items: [{name: root, className: nginx, hosts: [{host: api.example.com, paths: [{path: /}]}], tls: {enabled: true}}]}",
		"tls-wrong-host":               "ingress: {enabled: true, items: [{name: root, hosts: [{host: api.example.com, paths: [{path: /}]}], tls: {enabled: true, hosts: [other.example.com]}}]}",
		"recreate-with-rolling":        "deployment: {strategy: {type: Recreate}}",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := renderBuiltinChart(t, "helm-sinozo-java-v1", values, ""); err == nil {
				t.Fatal("invalid values accepted")
			}
		})
	}
}

func TestJavaChartDeliveredExamplesAndALBAnnotations(t *testing.T) {
	read := func(name string) map[string]interface{} {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "02-cicd", "examples", "java-skywalking", name))
		if err != nil {
			t.Fatal(err)
		}
		v, err := helmconfig.ParseValues(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	base := read("values-test.yaml")
	for _, overlay := range []string{"", "values-tracing.yaml", "values-alb-tls.yaml", "values-nginx-tls.yaml"} {
		t.Run(overlay, func(t *testing.T) {
			values := base
			if overlay != "" {
				values = workflow.MergeHelmValues(base, read(overlay))
			}
			raw, _ := yaml.Marshal(values)
			d, services, routes := javaResources(t, string(raw))
			if envMap(d.Spec.Template.Spec.Containers[0])["SPRING_CLOUD_NACOS_CONFIG_NAMESPACE"].Value != "rsc01-test" || len(services) != 1 || len(routes) != 1 {
				t.Fatal("delivered example did not apply")
			}
		})
	}
	prod, _ := yaml.Marshal(read("values-prod.yaml"))
	if _, err := renderBuiltinChart(t, "helm-sinozo-java-v1", string(prod), ""); err == nil {
		t.Fatal("production example with unfilled address must fail")
	}
	_, _, routes := javaResources(t, `service: {port: 9099}
ingress:
  enabled: true
  items:
  - name: gateway
    className: alb
    annotations:
      alb.ingress.kubernetes.io/use-regex: "true"
      alb.ingress.kubernetes.io/rewrite-target: /${2}
      alb.ingress.kubernetes.io/listen-ports: '[{"HTTP":80},{"HTTPS":443}]'
    hosts:
    - host: api.example.com
      paths: [{path: '/prod(/|$)(.*)', pathType: ImplementationSpecific}]
    tls: {enabled: true}
  - name: health
    className: alb
    hosts:
    - host: api.example.com
      paths: [{path: /health, pathType: Exact}]
`)
	if len(routes) != 2 {
		t.Fatal("multiple ingress routes missing")
	}
	found := false
	for _, r := range routes {
		if r.Name == "gateway-gateway-ingress" {
			found = true
			if r.Annotations["alb.ingress.kubernetes.io/rewrite-target"] != "/${2}" || r.Spec.Rules[0].HTTP.Paths[0].Path != "/prod(/|$)(.*)" || r.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number != 9099 {
				t.Fatal("ALB regex annotation or backend was rewritten")
			}
		}
	}
	if !found {
		t.Fatal("gateway route missing")
	}
}
