package serviceassets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"reflect"
	"regexp"
	"strings"
)

var editTemplateToken = regexp.MustCompile(`\{\{[^}]+\}\}|\$[A-Za-z][A-Za-z0-9_-]*\$`)

// Reject YAML edits to fields still owned by a runtime resource override.
// Literal source comparison preserves expressions instead of evaluating them.
func ValidateYAMLEditOwnership(before, after string, settings Settings) error {
	if settings.Resources == nil && settings.Replicas == nil && len(settings.Command) == 0 && len(settings.Args) == 0 || before == after {
		return nil
	}
	read := func(text string) (map[string]interface{}, error) {
		decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(editTemplateToken.ReplaceAllString(text, "axiops_template_value"))), 4096)
		result := map[string]interface{}{}
		for {
			var raw json.RawMessage
			err := decoder.Decode(&raw)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			var object unstructured.Unstructured
			if len(raw) == 0 || string(raw) == "null" {
				continue
			}
			if err = json.Unmarshal(raw, &object.Object); err != nil {
				return nil, err
			}
			if settings.WorkloadKind != "" && settings.WorkloadKind != object.GetKind() {
				continue
			}
			name := object.GetName()
			target := editTemplateToken.ReplaceAllString(settings.WorkloadName, "axiops_template_value")
			if target != "" && target != name && !strings.Contains(name, "axiops_template_value") {
				continue
			}
			if settings.Replicas != nil {
				value, _, _ := unstructured.NestedInt64(object.Object, "spec", "replicas")
				result[object.GetKind()+"/"+name+"/replicas"] = value
			}
			path := []string{"spec", "template", "spec", "containers"}
			if object.GetKind() == "CronJob" {
				path = []string{"spec", "jobTemplate", "spec", "template", "spec", "containers"}
			}
			containers, _, _ := unstructured.NestedSlice(object.Object, path...)
			for _, item := range containers {
				c, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				n, _ := c["name"].(string)
				if settings.Container != "" && settings.Container != n {
					continue
				}
				for key, value := range map[string]string{"requests/cpu": settings.ResourcesValue("requests/cpu"), "limits/cpu": settings.ResourcesValue("limits/cpu"), "requests/memory": settings.ResourcesValue("requests/memory"), "limits/memory": settings.ResourcesValue("limits/memory")} {
					if value == "" {
						continue
					}
					var section, field string
					for i, ch := range key {
						if ch == '/' {
							section = key[:i]
							field = key[i+1:]
							break
						}
					}
					v, _, _ := unstructured.NestedFieldNoCopy(c, "resources", section, field)
					result[object.GetKind()+"/"+name+"/"+n+"/"+key] = v
				}
				if len(settings.Command) > 0 {
					result[object.GetKind()+"/"+name+"/"+n+"/command"] = c["command"]
				}
				if len(settings.Args) > 0 {
					result[object.GetKind()+"/"+name+"/"+n+"/args"] = c["args"]
				}
			}
		}
		return result, nil
	}
	old, err := read(before)
	if err != nil {
		return nil
	}
	next, err := read(after)
	if err != nil {
		return nil
	}
	for key, value := range next {
		if previous, ok := old[key]; ok && !reflect.DeepEqual(previous, value) {
			return fmt.Errorf("%s 已由当前环境运行资源覆盖；请修改对应资源设置，或关闭覆盖后一起保存 YAML", key)
		}
	}
	return nil
}
