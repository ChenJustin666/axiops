package workflow

import (
	"strings"

	"sigs.k8s.io/yaml"
)

func redactedHelmValues(runtime, snapshot map[string]interface{}) map[string]interface{} {
	values := MergeHelmValues(runtime)
	var walk func(map[string]interface{}, map[string]interface{})
	walk = func(actual, refs map[string]interface{}) {
		for k, value := range refs {
			if ref, ok := value.(map[string]interface{}); ok {
				if _, secret := ref["credential_id"]; secret {
					actual[k] = "axiops-credential-redacted"
				} else if nested, ok := actual[k].(map[string]interface{}); ok {
					walk(nested, ref)
				}
			}
		}
	}
	walk(values, snapshot)
	return values
}

func redactedHelmPreview(manifest []byte, namespace string, runtime, snapshot map[string]interface{}) ([]byte, error) {
	resources, err := parseHelmManifestResources(manifest, namespace)
	if err != nil {
		return nil, err
	}
	var secrets []string
	var collect func(map[string]interface{}, map[string]interface{})
	collect = func(actual, refs map[string]interface{}) {
		for k, v := range refs {
			if ref, ok := v.(map[string]interface{}); ok {
				if _, credential := ref["credential_id"]; credential {
					if value, ok := actual[k].(string); ok && value != "" {
						secrets = append(secrets, value)
					}
				} else if nested, ok := actual[k].(map[string]interface{}); ok {
					collect(nested, ref)
				}
			}
		}
	}
	collect(runtime, snapshot)
	var redact func(interface{}) interface{}
	redact = func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			for _, secret := range secrets {
				v = strings.ReplaceAll(v, secret, "[凭据已隐藏]")
			}
			return v
		case map[string]interface{}:
			for k, x := range v {
				v[k] = redact(x)
			}
			return v
		case []interface{}:
			for i, x := range v {
				v[i] = redact(x)
			}
			return v
		}
		return value
	}
	var out strings.Builder
	for _, r := range resources {
		if r.GetKind() == "Secret" {
			delete(r.Object, "data")
			delete(r.Object, "stringData")
			out.WriteString("# Secret 内容已隐藏\n")
		}
		b, e := yaml.Marshal(redact(r.Object))
		if e != nil {
			return nil, e
		}
		out.Write(b)
		out.WriteString("---\n")
	}
	return []byte(out.String()), nil
}
