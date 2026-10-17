package helmconfig

import (
	"helm.sh/helm/v3/pkg/chart"
	"testing"
)

func TestChartSyntaxAllowsEnvironmentRequiredWithoutExecutingIt(t *testing.T) {
	c := &chart.Chart{Metadata: &chart.Metadata{Name: "native"}, Templates: []*chart.File{{Name: "templates/config.yaml", Data: []byte(`{{ include "app.name" . }}
{{ required "domain required" .Values.environment.domain | quote }}
{{ tpl .Values.config . }}
{{ lookup "v1" "Secret" .Release.Namespace "reference" | toYaml }}
{{ define "app.name" }}{{ .Release.Name }}{{ end }}`)}}}
	if err := ValidateChartSyntax(c); err != nil {
		t.Fatal("valid native chart needs environment values to check syntax", err)
	}
	c.Templates[0].Data = []byte(`{{ if .Values.enabled }}unterminated`)
	if err := ValidateChartSyntax(c); err == nil {
		t.Fatal("invalid template syntax accepted")
	}
	c.Templates[0].Data = []byte(`{{ madeUpFunction . }}`)
	if err := ValidateChartSyntax(c); err == nil {
		t.Fatal("unknown function accepted")
	}
}
