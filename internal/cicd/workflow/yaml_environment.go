package workflow

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func validateContainerEnvironment(resource unstructured.Unstructured) error {
	for _, group := range []string{"containers", "initContainers"} {
		containers, _, err := unstructured.NestedSlice(resource.Object, append(workloadPodSpec(resource), group)...)
		if err != nil {
			return err
		}
		names := map[string]bool{}
		for _, raw := range containers {
			c, ok := raw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%s/%s has invalid container", resource.GetKind(), resource.GetName())
			}
			name, _ := c["name"].(string)
			if name == "" || names[name] {
				return fmt.Errorf("%s/%s container name is empty or duplicated", resource.GetKind(), resource.GetName())
			}
			names[name] = true
			vars, found, err := unstructured.NestedSlice(c, "env")
			if err != nil {
				return err
			}
			if !found {
				continue
			}
			keys := map[string]bool{}
			for _, v := range vars {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("container %s has invalid env", name)
				}
				key, _ := item["name"].(string)
				if key == "" || keys[key] {
					return fmt.Errorf("container %s has empty or duplicate env name %q", name, key)
				}
				keys[key] = true
				value, hasValue := item["value"]
				from, hasFrom := item["valueFrom"]
				if hasValue && hasFrom {
					return fmt.Errorf("container %s env %s cannot use value and valueFrom together", name, key)
				}
				if hasValue {
					if _, ok := value.(string); !ok {
						return fmt.Errorf("container %s env %s value must be a string; quote numbers and booleans", name, key)
					}
				}
				if hasFrom {
					refs, ok := from.(map[string]interface{})
					if !ok || len(refs) != 1 {
						return fmt.Errorf("container %s env %s needs exactly one valueFrom source", name, key)
					}
				}
			}
		}
	}
	return nil
}
