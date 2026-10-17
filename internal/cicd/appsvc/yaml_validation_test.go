package appsvc

import (
	"strings"
	"testing"
)

func TestValidatePortableServiceYAML(t *testing.T) {
	valid := "# AxiOps-Variables: [{\"key\":\"port\",\"value\":\"\"}]\n# AxiOps-File: service.yaml\napiVersion: v1\nkind: Service\nmetadata:\n  name: {{.Service}}\nspec:\n  ports:\n    - port: {{.port}}\n---\n# AxiOps-File: configmap.yaml\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: $Service$-config\ndata:\n  opts: >-\n    -XX:+UseG1GC\n    -Dspring.application.name={{.Service}}\n"
	if err := validateYAMLDefinition(valid); err != nil {
		t.Fatal(err)
	}
	for name, definition := range map[string]string{
		"broken block":             "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\ndata:\n  opts: >-\n-XX:+UseG1GC\n",
		"missing separator":        "apiVersion: v1\nkind: Service\nmetadata:\n  name: app\napiVersion: v1\nkind: Service\nmetadata:\n  name: second\n",
		"scalar document":          valid + "---\nhello\n",
		"missing name":             "apiVersion: v1\nkind: Service\n",
		"empty":                    "# only comment\n---\n",
		"secret":                   "apiVersion: v1\nkind: Secret\nmetadata:\n  name: credentials\n",
		"invalid metadata":         "# AxiOps-Variables: {\n" + valid,
		"duplicate vars":           "# AxiOps-Variables: [{\"key\":\"port\",\"value\":\"80\"},{\"key\":\"port\",\"value\":\"90\"}]\napiVersion: v1\nkind: Service\nmetadata:\n  name: svc\n",
		"invalid rendered default": strings.Replace(valid, `"value":""`, `"value":"["`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateYAMLDefinition(definition); err == nil {
				t.Fatal("invalid definition accepted")
			}
		})
	}
}
