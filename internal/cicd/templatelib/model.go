package templatelib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/cicd/helmimport"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type Variable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Label       string `json:"label,omitempty"`
	Group       string `json:"group,omitempty"`
}
type Template struct {
	ID          int64      `json:"id"`
	Kind        string     `json:"kind"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Files       []File     `json:"files"`
	Variables   []Variable `json:"variables"`
	Revision    int        `json:"revision"`
	BuiltinKey  string     `json:"builtin_key"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

var keyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
var placeholder = regexp.MustCompile(`\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}`)
var builtins = map[string]string{"Service": "example-service", "Image": "example.invalid/team/service:test", "Namespace": "example-test", "EnvName": "test", "Product": "example-project", "ClusterName": "example-cluster"}

func Validate(t Template) error {
	if t.Kind != "k8s_yaml" && t.Kind != "helm" && t.Kind != "dockerfile" && t.Kind != "build" && t.Kind != "workflow" {
		return fmt.Errorf("不支持的模板类型")
	}
	if strings.TrimSpace(t.Name) == "" || len([]rune(t.Name)) > 128 || len([]rune(t.Description)) > 512 {
		return fmt.Errorf("请填写模板名称（最多 128 字）和有效描述")
	}
	maxFiles, maxBytes := 64, 512*1024
	if t.Kind == "helm" {
		// A Git-imported Chart must remain editable under the same snapshot limits.
		maxFiles, maxBytes = 500, 4<<20
	}
	if len(t.Files) == 0 || len(t.Files) > maxFiles || len(t.Variables) > 64 {
		return fmt.Errorf("模板需要 1–%d 个文件，最多 64 个变量", maxFiles)
	}
	values := map[string]string{}
	for k, v := range builtins {
		values[k] = v
	}
	for _, v := range t.Variables {
		if len(v.Label) > 128 || len(v.Group) > 64 {
			return fmt.Errorf("参数名称或分组过长")
		}
		if !keyPattern.MatchString(v.Key) {
			return fmt.Errorf("变量名 %q 无效", v.Key)
		}
		if _, ok := values[v.Key]; ok {
			return fmt.Errorf("变量 %q 重复或属于系统内置变量", v.Key)
		}
		if len(v.Value) > 4096 || len(v.Description) > 1024 {
			return fmt.Errorf("变量内容过长")
		}
		values[v.Key] = v.Value
	}
	size := 0
	seen := map[string]bool{}
	chartFiles := []*loader.BufferedFile{}
	for _, f := range t.Files {
		if f.Path == "" || f.Path == "." || len(f.Path) > 200 || strings.ContainsAny(f.Path, "\\\r\n:") || path.IsAbs(f.Path) || path.Clean(f.Path) != f.Path || f.Path == ".." || strings.HasPrefix(f.Path, "../") || seen[f.Path] {
			return fmt.Errorf("文件路径无效或重复：%q", f.Path)
		}
		seen[f.Path] = true
		size += len(f.Content)
		if size > maxBytes || t.Kind != "helm" && strings.TrimSpace(f.Content) == "" {
			return fmt.Errorf("文件内容无效或模板总大小超过 %d KiB", maxBytes/1024)
		}
		if t.Kind == "k8s_yaml" {
			missing := ""
			rendered := placeholder.ReplaceAllStringFunc(f.Content, func(m string) string {
				key := placeholder.FindStringSubmatch(m)[1]
				v, ok := values[key]
				if !ok || v == "" {
					missing = key
					return "missing"
				}
				return v
			})
			if missing != "" {
				return fmt.Errorf("请为变量 %s 配置默认值", missing)
			}
			decoder := yaml.NewDecoder(strings.NewReader(rendered))
			for {
				var document interface{}
				err := decoder.Decode(&document)
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("%s: %w（多个资源之间请使用 --- 分隔）", f.Path, err)
				}
			}
			if _, err := workflow.ParseYAMLManifest([]byte(rendered)); err != nil {
				return fmt.Errorf("%s: %w", f.Path, err)
			}
		}
		chartFiles = append(chartFiles, &loader.BufferedFile{Name: f.Path, Data: []byte(f.Content)})
	}
	if t.Kind == "workflow" {
		if len(t.Files) != 1 || t.Files[0].Path != "workflow.json" || len(t.Variables) != 0 {
			return fmt.Errorf("工作流模板使用一个 workflow.json 文件；服务和环境在创建工作流时选择")
		}
		decoder := json.NewDecoder(strings.NewReader(t.Files[0].Content))
		decoder.DisallowUnknownFields()
		var def workflow.Definition
		if err := decoder.Decode(&def); err != nil {
			return fmt.Errorf("工作流 JSON 无效：%w", err)
		}
		if decoder.Decode(new(interface{})) != io.EOF {
			return fmt.Errorf("工作流 JSON 只能包含一个定义")
		}
		if issues := workflow.ValidateDefinition(def); len(issues) > 0 {
			return fmt.Errorf("工作流模板校验失败：%s", issues[0].Message)
		}
		for _, node := range def.Nodes {
			var cfg map[string]interface{}
			if len(node.Config) > 0 {
				if err := json.Unmarshal(node.Config, &cfg); err != nil {
					return fmt.Errorf("节点配置必须是对象")
				}
			}
			for _, key := range []string{"environment_id", "cluster_id", "namespace", "host_group_ref", "frozen_config"} {
				if _, exists := cfg[key]; exists {
					return fmt.Errorf("模板不能固定目标字段 %s", key)
				}
			}
			for _, key := range []string{"service_id", "service_revision_id"} {
				if value, exists := cfg[key]; exists && value != float64(0) {
					return fmt.Errorf("模板的 %s 必须为 0，由创建时选择服务填入", key)
				}
			}
			if node.Type == workflow.NodeTest {
				if command, ok := cfg["command"].(string); !ok || strings.TrimSpace(command) == "" {
					return fmt.Errorf("测试节点需要 command")
				}
			}
		}
	}
	if t.Kind == "helm" {
		c, err := loader.LoadFiles(chartFiles)
		if err != nil {
			return fmt.Errorf("Chart 校验失败：%w", err)
		}
		if c.Metadata.APIVersion != "v2" || c.Metadata.Type != "" && c.Metadata.Type != "application" {
			return fmt.Errorf("仅支持 Helm v2 application Chart")
		}
		if err := helmconfig.ValidateWorkloadValues(c, map[string]interface{}{}, map[string]interface{}{}); err != nil {
			return err
		}
		for _, dependency := range c.Metadata.Dependencies {
			found := false
			for _, bundled := range c.Dependencies() {
				if bundled.Name() == dependency.Name || dependency.Alias != "" && bundled.Name() == dependency.Alias {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("Chart 依赖 %s 未包含在 charts/ 中，请先打包依赖", dependency.Name)
			}
		}
		values, err := chartutil.ToRenderValues(c, map[string]interface{}{}, chartutil.ReleaseOptions{Name: "example", Namespace: "example-test", IsInstall: true}, nil)
		if err != nil {
			return fmt.Errorf("Chart 默认参数校验失败：%w", err)
		}
		if _, err := engine.Render(c, values); err != nil {
			return fmt.Errorf("Chart 默认参数渲染失败：%w", err)
		}
	}
	if t.Kind == "dockerfile" {
		valid := false
		for _, f := range t.Files {
			if f.Path == "Dockerfile" {
				valid = regexp.MustCompile(`(?im)^\s*FROM\s+\S+`).MatchString(f.Content)
			}
		}
		if !valid {
			return fmt.Errorf("Dockerfile 模板需要 Dockerfile 文件和 FROM 指令")
		}
	}
	if t.Kind == "build" {
		if len(t.Files) != 1 || t.Files[0].Path != "build.yaml" {
			return fmt.Errorf("构建模板使用一个 build.yaml 文件")
		}
		var cfg map[string]interface{}
		if err := yaml.Unmarshal([]byte(t.Files[0].Content), &cfg); err != nil {
			return fmt.Errorf("构建配置 YAML 无效：%w", err)
		}
		for _, k := range []string{"build_image", "build_command"} {
			if value, ok := cfg[k].(string); !ok || strings.TrimSpace(value) == "" {
				return fmt.Errorf("构建模板需要 %s", k)
			}
		}
		for k := range cfg {
			if k != "build_image" && k != "build_command" && k != "cache_dir" {
				return fmt.Errorf("构建模板不支持字段 %s", k)
			}
		}
		if value, exists := cfg["cache_dir"]; exists {
			if _, ok := value.(string); !ok {
				return fmt.Errorf("cache_dir 必须是路径字符串")
			}
		}
	}
	return nil
}

// Snapshot is copied into a service revision. Editing the source template never
// mutates a service or triggers a deployment.
func Snapshot(t Template) (string, error) {
	if err := Validate(t); err != nil {
		return "", err
	}
	if t.Kind == "workflow" {
		return t.Files[0].Content, nil
	}
	if t.Kind == "helm" {
		// Use the same immutable, digested chart representation as Git imports.
		definition := helmimport.Definition{SchemaVersion: 1, ChartPath: "."}
		for _, f := range t.Files {
			digest := sha256.Sum256([]byte(f.Content))
			definition.Files = append(definition.Files, helmimport.ChartFile{Path: f.Path, Content: f.Content, SHA256: hex.EncodeToString(digest[:])})
			if f.Path == "Chart.yaml" {
				if err := yaml.Unmarshal([]byte(f.Content), &definition.Metadata); err != nil {
					return "", err
				}
			}
		}
		data, err := json.Marshal(definition)
		if err == nil {
			_, err = helmconfig.LoadChart(data)
		}
		return string(data), err
	}
	var parts []string
	for _, f := range t.Files {
		parts = append(parts, "# AxiOps-File: "+f.Path+"\n"+f.Content)
	}
	metadata, _ := json.Marshal(t.Variables)
	return "# AxiOps-Variables: " + string(metadata) + "\n" + strings.Join(parts, "\n---\n"), nil
}
