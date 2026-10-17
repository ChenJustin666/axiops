package templatelib

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func TestGitSizedChartRemainsEditableWithoutRelaxingYAMLLimits(t *testing.T) {
	chart := Template{Kind: "helm", Name: "native", Files: []File{
		{Path: "Chart.yaml", Content: "apiVersion: v2\nname: native\nversion: 1.0.0\n"},
		{Path: "values.yaml", Content: "{}\n"},
		{Path: ".helmignore", Content: ""},
	}}
	for i := 0; i < 70; i++ {
		chart.Files = append(chart.Files, File{Path: fmt.Sprintf("files/config-%d.txt", i), Content: strings.Repeat("a", 8192)})
	}
	if _, err := Snapshot(chart); err != nil {
		t.Fatalf("Git-sized native Chart cannot be edited: %v", err)
	}
	chart.Kind = "k8s_yaml"
	if err := Validate(chart); err == nil {
		t.Fatal("non-Helm file limit was relaxed")
	}
	chart.Kind = "helm"
	chart.Files = append(chart.Files, File{Path: "files/too-large.txt", Content: strings.Repeat("b", 4<<20)})
	if _, err := Snapshot(chart); err == nil {
		t.Fatal("oversized Chart accepted")
	}
}

func TestJavaNacosTemplateUsesEnvironmentValuesWithoutUnselectedSecrets(t *testing.T) {
	var template Template
	for _, item := range Defaults() {
		if item.BuiltinKey == "k8s-java17-nacos" {
			template = item
		}
	}
	if err := Validate(template); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for key, value := range builtins {
		values[key] = value
	}
	for _, variable := range template.Variables {
		values[variable.Key] = variable.Value
	}
	values["Service"] = "card-scout-gateway"
	values["springProfile"] = "prod"
	values["nacosNamespace"] = "production-namespace-id"
	values["nacosServerAddr"] = "prod-nacos:8848"
	rendered := placeholder.ReplaceAllStringFunc(template.Files[0].Content, func(match string) string {
		return values[placeholder.FindStringSubmatch(match)[1]]
	})
	var deployment struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Env []struct {
							Name  string `yaml:"name"`
							Value string `yaml:"value"`
						} `yaml:"env"`
					} `yaml:"containers"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal([]byte(rendered), &deployment); err != nil {
		t.Fatal(err)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatal("expected one application container")
	}
	env := map[string]string{}
	for _, item := range deployment.Spec.Template.Spec.Containers[0].Env {
		env[item.Name] = item.Value
		if item.Name == "SPRING_CLOUD_NACOS_USERNAME" || item.Name == "SPRING_CLOUD_NACOS_PASSWORD" {
			t.Fatalf("Nacos credentials must be added only when selected: %+v", item)
		}
	}
	for key, want := range map[string]string{
		"SPRING_PROFILES_ACTIVE": "prod", "SPRING_CONFIG_IMPORT": "optional:nacos:card-scout-gateway",
		"SPRING_CLOUD_NACOS_DISCOVERY_SERVERADDR": "prod-nacos:8848", "SPRING_CLOUD_NACOS_CONFIG_SERVERADDR": "prod-nacos:8848",
		"SPRING_CLOUD_NACOS_CONFIG_NAMESPACE": "production-namespace-id", "SPRING_CLOUD_NACOS_DISCOVERY_NAMESPACE": "production-namespace-id",
	} {
		if env[key] != want {
			t.Fatalf("%s=%q, want %q", key, env[key], want)
		}
	}
	if strings.Contains(env["JAVA_OPTS"], "\n") || !strings.Contains(env["JAVA_OPTS"], "-XX:+UseG1GC -Dfile.encoding=UTF-8") || strings.Contains(rendered, "spring.profiles.active=test") {
		t.Fatalf("incorrect folded options or hardcoded target: %s", rendered)
	}
}

func TestDefaultsAreValidAndSnapshotsIndependent(t *testing.T) {
	for _, template := range Defaults() {
		t.Run(template.Kind, func(t *testing.T) {
			if err := Validate(template); err != nil {
				t.Fatal(err)
			}
			snapshot, err := Snapshot(template)
			if err != nil {
				t.Fatal(err)
			}
			if template.Kind == "k8s_yaml" {
				if !strings.Contains(snapshot, "# AxiOps-Variables:") || !strings.Contains(snapshot, "\n---\n") || !strings.Contains(snapshot, "{{.Image}}") || strings.Contains(snapshot, "10Mi") || strings.Contains(snapshot, "command:") {
					t.Fatalf("incorrect portable definition %s", snapshot)
				}
			}
			if template.Kind == "helm" {
				var def helmimport.Definition
				if err := json.Unmarshal([]byte(snapshot), &def); err != nil {
					t.Fatal(err)
				}
				if def.SchemaVersion != 1 || def.Metadata.Name == "" || def.Files[0].SHA256 == "" {
					t.Fatal("incomplete chart snapshot")
				}
				files := []*loader.BufferedFile{}
				for _, f := range def.Files {
					files = append(files, &loader.BufferedFile{Name: f.Path, Data: []byte(f.Content)})
				}
				chart, err := loader.LoadFiles(files)
				if err != nil {
					t.Fatal(err)
				}
				values, err := chartutil.ToRenderValues(chart, map[string]interface{}{}, chartutil.ReleaseOptions{Name: "test", Namespace: "qa", IsInstall: true}, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := engine.Render(chart, values); err != nil {
					t.Fatal(err)
				}
			}
			template.Files[0].Content = "changed"
			if strings.Contains(snapshot, "changed") {
				t.Fatal("snapshot followed template mutation")
			}
		})
	}
}

func TestHelmStarterRendersEnvironmentRuntimeSettings(t *testing.T) {
	var files []*loader.BufferedFile
	for _, item := range Defaults() {
		if item.BuiltinKey == "helm-service" {
			for _, f := range item.Files {
				files = append(files, &loader.BufferedFile{Name: f.Path, Data: []byte(f.Content)})
			}
		}
	}
	chart, err := loader.LoadFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	var overrides map[string]interface{}
	if err := yaml.Unmarshal([]byte(`replicaCount: 3
service:
  port: 28080
resources:
  limits:
    memory: 4Gi
env:
  - name: SPRING_PROFILES_ACTIVE
    value: prod
envFrom:
  - secretRef:
      name: app-runtime
command: ["sh", "-c"]
args: ["exec java $JAVA_OPTS -jar app.jar"]
`), &overrides); err != nil {
		t.Fatal(err)
	}
	values, err := chartutil.ToRenderValues(chart, overrides, chartutil.ReleaseOptions{Name: "api", Namespace: "prod", IsInstall: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := engine.Render(chart, values)
	if err != nil {
		t.Fatal(err)
	}
	var deployment struct {
		Spec struct {
			Replicas int `yaml:"replicas"`
			Template struct {
				Spec struct {
					Containers []struct {
						Command []string                       `yaml:"command"`
						Args    []string                       `yaml:"args"`
						Env     []struct{ Name, Value string } `yaml:"env"`
						EnvFrom []struct {
							SecretRef struct{ Name string } `yaml:"secretRef"`
						} `yaml:"envFrom"`
						Resources struct{ Limits struct{ Memory string } } `yaml:"resources"`
						Ports     []struct {
							ContainerPort int `yaml:"containerPort"`
						} `yaml:"ports"`
					} `yaml:"containers"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal([]byte(rendered["application/templates/deployment.yaml"]), &deployment); err != nil {
		t.Fatal(err)
	}
	if deployment.Spec.Replicas != 3 || len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatal("environment replicas or container missing")
	}
	c := deployment.Spec.Template.Spec.Containers[0]
	if len(c.Command) != 2 || len(c.Args) != 1 || c.Args[0] != "exec java $JAVA_OPTS -jar app.jar" || len(c.Env) != 1 || c.Env[0].Value != "prod" || len(c.EnvFrom) != 1 || c.EnvFrom[0].SecretRef.Name != "app-runtime" || c.Resources.Limits.Memory != "4Gi" || len(c.Ports) != 1 || c.Ports[0].ContainerPort != 28080 {
		t.Fatalf("environment runtime settings not rendered correctly: %+v", c)
	}
}

func TestWorkflowTemplateRejectsTargetsUnsupportedNodesAndCycles(t *testing.T) {
	for _, body := range []string{
		`{"delivery_type":"k8s_yaml","nodes":[{"key":"db","name":"DB","type":"database"}]}`,
		`{"delivery_type":"k8s_yaml","nodes":[{"key":"build","name":"Build","type":"build","config":{"service_id":17}}]}`,
		`{"delivery_type":"k8s_yaml","nodes":[{"key":"build","name":"Build","type":"build","config":{"namespace":"production"}}]}`,
		`{"delivery_type":"k8s_yaml","nodes":[{"key":"build","name":"Build","type":"build","depends_on":["build"]}]}`,
	} {
		template := Template{Kind: "workflow", Name: "test", Files: []File{{Path: "workflow.json", Content: body}}}
		if err := Validate(template); err == nil {
			t.Fatal("accepted unsafe workflow template")
		}
	}
}
func TestRejectUnsafeAndIncompleteTemplates(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Template)
	}{
		{"undefined variable", func(x *Template) { x.Variables = nil }},
		{"override builtin", func(x *Template) { x.Variables = append(x.Variables, Variable{Key: "Image", Value: "oops"}) }},
		{"duplicate files", func(x *Template) { x.Files = append(x.Files, x.Files[0]) }},
		{"traversal", func(x *Template) { x.Files[0].Path = "../outside.yaml" }},
		{"missing separator", func(x *Template) {
			x.Files = []File{{Path: "all.yaml", Content: x.Files[0].Content + "\n" + x.Files[1].Content}}
		}},
		{"secret body", func(x *Template) {
			x.Files = []File{{Path: "secret.yaml", Content: "apiVersion: v1\nkind: Secret\nmetadata:\n  name: private\n"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			x := Defaults()[0]
			test.mutate(&x)
			if err := Validate(x); err == nil {
				t.Fatal("accepted invalid template")
			}
		})
	}
}

func TestRejectBrokenChartAndBuildParameters(t *testing.T) {
	chart := Defaults()[1]
	chart.Files[2].Content = "{{ if }}"
	if err := Validate(chart); err == nil {
		t.Fatal("accepted broken Helm syntax")
	}
	chart = Defaults()[1]
	chart.Files[0].Content += "dependencies:\n  - name: missing\n    version: 1.0.0\n    repository: https://example.invalid\n"
	if err := Validate(chart); err == nil {
		t.Fatal("accepted unbundled dependency")
	}
	build := Defaults()[3]
	build.Files[0].Content = "build_image: maven:3\nbuild_command: mvn package\ncache_dir: 123\n"
	if err := Validate(build); err == nil {
		t.Fatal("accepted non-string cache directory")
	}
}
