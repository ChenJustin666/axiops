package helmconfig

import (
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"helm.sh/helm/v3/pkg/chart"
	"text/template"
)

// Check syntax without executing required, lookup, schema validation or any
// values-dependent expression. Actual rendering still uses Helm's engine.
func ValidateChartSyntax(c *chart.Chart) error {
	functions := sprig.TxtFuncMap()
	delete(functions, "env")
	delete(functions, "expandenv")
	placeholder := func(...interface{}) interface{} { return nil }
	for _, name := range []string{"toToml", "fromToml", "toYaml", "toYamlPretty", "fromYaml", "fromYamlArray", "toJson", "fromJson", "fromJsonArray", "include", "tpl", "required", "lookup"} {
		functions[name] = placeholder
	}
	root := template.New("chart").Funcs(functions)
	var check func(*chart.Chart) error
	check = func(current *chart.Chart) error {
		if len(current.Schema) > 0 && !json.Valid(current.Schema) {
			return fmt.Errorf("Helm values.schema.json syntax is invalid")
		}
		for _, file := range current.Templates {
			if _, err := root.New(current.ChartFullPath() + "/" + file.Name).Parse(string(file.Data)); err != nil {
				return fmt.Errorf("Helm template syntax is invalid: %s", file.Name)
			}
		}
		for _, child := range current.Dependencies() {
			if err := check(child); err != nil {
				return err
			}
		}
		return nil
	}
	return check(c)
}
