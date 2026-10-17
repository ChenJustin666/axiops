package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"strings"
)

// Roll only consumers of ConfigMaps managed in this manifest. External maps
// and Secret payloads are deliberately not read or hashed here.
func stampManagedConfigMaps(manifest []byte, namespace string) ([]byte, error) {
	// Preserve the resolver's existing output when there are no ConfigMaps;
	// the ordinary preflight still validates the complete deployment manifest.
	if !strings.Contains(string(manifest), "ConfigMap") {
		return manifest, nil
	}
	objects, err := ValidateYAMLManifest(manifest, namespace)
	if err != nil {
		return nil, err
	}
	maps := map[string]interface{}{}
	for _, object := range objects {
		if object.GetKind() == "ConfigMap" {
			maps[object.GetName()] = map[string]interface{}{"data": object.Object["data"], "binaryData": object.Object["binaryData"]}
		}
	}
	if len(maps) == 0 {
		return manifest, nil
	}
	changed := false
	for i := range objects {
		object := &objects[i]
		path := workloadPodSpec(*object)
		spec, found, _ := unstructured.NestedMap(object.Object, path...)
		if !found {
			continue
		}
		refs := map[string]interface{}{}
		var visit func(interface{})
		visit = func(v interface{}) {
			switch item := v.(type) {
			case map[string]interface{}:
				for k, value := range item {
					if k == "configMap" || k == "configMapRef" || k == "configMapKeyRef" {
						if m, ok := value.(map[string]interface{}); ok {
							if name, ok := m["name"].(string); ok {
								if content, ok := maps[name]; ok {
									refs[name] = content
								}
							}
						}
					}
					visit(value)
				}
			case []interface{}:
				for _, entry := range item {
					visit(entry)
				}
			}
		}
		visit(spec)
		if len(refs) == 0 {
			continue
		}
		raw, _ := json.Marshal(refs)
		sum := sha256.Sum256(raw)
		annotationPath := append(append([]string{}, path[:len(path)-1]...), "metadata", "annotations", "axiops.io/managed-configmap-hash")
		if err = unstructured.SetNestedField(object.Object, hex.EncodeToString(sum[:]), annotationPath...); err != nil {
			return nil, err
		}
		changed = true
	}
	if !changed {
		return manifest, nil
	}
	out := make([]string, 0, len(objects))
	for _, object := range objects {
		raw, e := yaml.Marshal(object.Object)
		if e != nil {
			return nil, e
		}
		out = append(out, string(raw))
	}
	return []byte(strings.Join(out, "\n---\n")), nil
}
