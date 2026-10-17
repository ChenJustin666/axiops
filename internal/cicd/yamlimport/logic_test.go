package yamlimport

import (
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
)

func TestSummarizeExtractsComponentsAndImages(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      initContainers:
      - name: migrate
        image: registry.example/migrate:v1
      containers:
      - name: api
        image: registry.example/api:v2
---
apiVersion: v1
kind: Service
metadata:
  name: api
`
	resources, err := workflow.ParseYAMLManifest([]byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	preview := summarize("abc", manifest, resources)
	if preview.ResourceCount != 2 || preview.ComponentCount != 1 || len(preview.Images) != 2 || preview.Components[0] != "api" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestParseRejectsSecretAndUnsupportedResource(t *testing.T) {
	for _, manifest := range []string{
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: password\n",
		"apiVersion: v1\nkind: Namespace\nmetadata:\n  name: forbidden\n",
	} {
		if _, err := workflow.ParseYAMLManifest([]byte(manifest)); err == nil {
			t.Fatalf("expected rejection for %s", manifest)
		}
	}
}

func TestJoinFilesUsesDocumentBoundary(t *testing.T) {
	got := joinFiles([]codesource.RepositoryFile{{Content: []byte("kind: Service\n")}, {Content: []byte("kind: Deployment")}})
	if !strings.Contains(got, "\n---\n") || !strings.HasSuffix(got, "\n") {
		t.Fatalf("unexpected joined definition %q", got)
	}
}
