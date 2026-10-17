package appsvc

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var definitionVariable = regexp.MustCompile(`\{\{\s*\.?([A-Za-z][A-Za-z0-9_]*)\s*\}\}|\$([A-Za-z][A-Za-z0-9_-]*)\$`)
var definitionVariableKey = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// ValidateYAMLDefinition also protects first-time environment configuration.
func ValidateYAMLDefinition(definition string) error { return validateYAMLDefinition(definition) }

// validateYAMLDefinition checks portable structure at save time. Missing
// environment values remain placeholders; deployment performs full rendering,
// resource validation and target namespace checks using the frozen snapshot.
func validateYAMLDefinition(definition string) error {
	values := map[string]string{}
	metadataSeen := false
	for _, line := range strings.Split(definition, "\n") {
		if !strings.HasPrefix(line, "# AxiOps-Variables:") {
			continue
		}
		if metadataSeen {
			return fmt.Errorf("变量元数据重复")
		}
		metadataSeen = true
		var variables []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "# AxiOps-Variables:"))), &variables); err != nil {
			return fmt.Errorf("变量元数据不是有效的 JSON 变量列表")
		}
		for _, variable := range variables {
			if !definitionVariableKey.MatchString(variable.Key) {
				return fmt.Errorf("变量名 %q 无效", variable.Key)
			}
			if _, ok := values[variable.Key]; ok {
				return fmt.Errorf("变量名 %q 重复", variable.Key)
			}
			values[variable.Key] = variable.Value
		}
	}
	rendered := definitionVariable.ReplaceAllStringFunc(definition, func(match string) string {
		parts := definitionVariable.FindStringSubmatch(match)
		key := parts[1]
		if key == "" {
			key = parts[2]
		}
		if value := values[key]; strings.TrimSpace(value) != "" {
			return value
		}
		return "axiops-variable"
	})
	decoder := yaml.NewDecoder(strings.NewReader(rendered))
	count := 0
	for {
		var resource map[string]interface{}
		if err := decoder.Decode(&resource); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("YAML 语法错误：%w（多个资源请用 --- 分隔）", err)
		}
		if resource == nil {
			continue
		}
		count++
		metadata, _ := resource["metadata"].(map[string]interface{})
		version, _ := resource["apiVersion"].(string)
		kind, _ := resource["kind"].(string)
		name, _ := metadata["name"].(string)
		if strings.TrimSpace(version) == "" || strings.TrimSpace(kind) == "" || strings.TrimSpace(name) == "" {
			return fmt.Errorf("第 %d 个资源需要有效的 apiVersion、kind 和 metadata.name", count)
		}
		if kind == "Secret" {
			return fmt.Errorf("Secret 请通过环境凭据管理并在 YAML 中引用，不保存密码正文")
		}
	}
	if count == 0 {
		return fmt.Errorf("YAML 中没有可部署的资源")
	}
	return nil
}
