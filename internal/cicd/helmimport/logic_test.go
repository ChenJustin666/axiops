package helmimport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func TestNativeChartImportRendersHelpersAndBundledSubchart(t *testing.T) {
	snapshot := &codesource.RepositorySnapshot{CommitID: "frozen"}
	for name, content := range map[string]string{
		"Chart.yaml":                          "apiVersion: v2\nname: native\nversion: 1.0.0\ndependencies:\n- name: child\n  version: 1.0.0\n  repository: file://charts/child\n",
		"values.yaml":                         "message: hello\n",
		"templates/_helpers.tpl":              `{{ define "native.name" }}{{ .Release.Name }}-native{{ end }}`,
		"templates/configmap.yaml":            "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ include \"native.name\" . }}\ndata:\n  message: {{ .Values.message | quote }}\n",
		"charts/child/Chart.yaml":             "apiVersion: v2\nname: child\nversion: 1.0.0\n",
		"charts/child/templates/service.yaml": "apiVersion: v1\nkind: Service\nmetadata:\n  name: {{ .Release.Name }}-child\n",
	} {
		snapshot.Files = append(snapshot.Files, codesource.RepositoryFile{Path: "deploy/chart/" + name, Content: []byte(content)})
	}
	logic := &Logic{services: &fakeServices{service: &appsvc.AppService{ID: 7, DeliveryType: appsvc.DeliveryTypeHelm, CodeSourceID: 9}}, sources: &fakeSources{snapshot: snapshot}}
	p, code := logic.Preview(context.Background(), 5, 7, Request{CodeSourceID: 9, ChartPath: "deploy/chart"})
	if code != errcode.CodeSuccess {
		t.Fatalf("native import failed: %d", code)
	}
	c, err := helmconfig.LoadChart([]byte(p.Definition))
	if err != nil {
		t.Fatal(err)
	}
	values, err := chartutil.ToRenderValues(c, map[string]interface{}{}, chartutil.ReleaseOptions{Name: "qa", Namespace: "qa", IsInstall: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := engine.Render(c, values)
	if err != nil || !strings.Contains(rendered["native/templates/configmap.yaml"], "name: qa-native") || !strings.Contains(rendered["native/charts/child/templates/service.yaml"], "name: qa-child") {
		t.Fatalf("native Chart structure did not survive import: %v / %v", rendered, err)
	}
}

type fakeServices struct {
	service         *appsvc.AppService
	createdRevision *appsvc.CreateServiceRevisionRequest
}

func (f *fakeServices) GetByID(context.Context, int64, int64) (*appsvc.AppService, int) {
	return f.service, errcode.CodeSuccess
}

func (f *fakeServices) CreateRevision(_ context.Context, _, serviceID int64, request *appsvc.CreateServiceRevisionRequest) (*appsvc.ServiceRevision, int) {
	f.createdRevision = request
	return &appsvc.ServiceRevision{ID: 31, ServiceID: serviceID, SourceCommit: request.SourceCommit, Definition: request.Definition}, errcode.CodeSuccess
}

type fakeSources struct {
	snapshot *codesource.RepositorySnapshot
	calls    int
}

func (f *fakeSources) ReadFiles(context.Context, int64, int64, string, string, []string) (*codesource.RepositorySnapshot, error) {
	f.calls++
	return f.snapshot, nil
}
func (f *fakeSources) ReadChartFiles(context.Context, int64, int64, string, string, string) (*codesource.RepositorySnapshot, error) {
	f.calls++
	return f.snapshot, nil
}

func chartSnapshot() *codesource.RepositorySnapshot {
	return &codesource.RepositorySnapshot{
		CommitID: "d34db33f",
		Files: []codesource.RepositoryFile{
			{Path: "deploy/payment/templates/deployment.yaml", Content: []byte("kind: Deployment\n")},
			{Path: "deploy/payment/values.yaml", Content: []byte("image:\n  repository: registry.example/payment\n  tag: v2\nworker:\n  image: registry.example/worker:v1\nexistingSecret: payment-db\n")},
			{Path: "deploy/payment/Chart.yaml", Content: []byte("apiVersion: v2\nname: payment\ndescription: Payment service\ntype: application\nversion: 1.2.3\nappVersion: 2.0.0\n")},
		},
	}
}

func TestBuildPreviewCreatesDeterministicChartSnapshot(t *testing.T) {
	preview, err := buildPreview(chartSnapshot(), "deploy/payment")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Metadata.Name != "payment" || preview.Metadata.Version != "1.2.3" {
		t.Fatalf("unexpected metadata: %+v", preview.Metadata)
	}
	if preview.FileCount != 3 || preview.TemplateCount != 1 || preview.ComponentCount != 2 {
		t.Fatalf("unexpected chart counts: %+v", preview)
	}
	if len(preview.Images) != 2 || preview.Images[0] != "registry.example/payment:v2" {
		t.Fatalf("unexpected images: %#v", preview.Images)
	}
	if len(preview.DefaultValueKeys) != 3 || preview.DefaultValueKeys[0] != "existingSecret" {
		t.Fatalf("unexpected values keys: %#v", preview.DefaultValueKeys)
	}
	var definition Definition
	if err := json.Unmarshal([]byte(preview.Definition), &definition); err != nil {
		t.Fatal(err)
	}
	if definition.SchemaVersion != 1 || definition.Files[0].Path != "Chart.yaml" || definition.Files[2].Path != "values.yaml" {
		t.Fatalf("unexpected definition: %+v", definition)
	}
}

func TestBuildPreviewRejectsUnsafeChartInput(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *codesource.RepositorySnapshot
		path     string
	}{
		{name: "path traversal", snapshot: chartSnapshot(), path: "../payment"},
		{name: "outside file", snapshot: &codesource.RepositorySnapshot{CommitID: "a", Files: []codesource.RepositoryFile{{Path: "Chart.yaml", Content: []byte("apiVersion: v2\nname: bad\nversion: 1.0.0\n")}}}, path: "deploy/payment"},
		{name: "plaintext password", snapshot: &codesource.RepositorySnapshot{CommitID: "a", Files: []codesource.RepositoryFile{
			{Path: "chart/Chart.yaml", Content: []byte("apiVersion: v2\nname: bad\nversion: 1.0.0\n")},
			{Path: "chart/values.yaml", Content: []byte("database:\n  password: changeme\n")},
		}}, path: "chart"},
		{name: "library chart", snapshot: &codesource.RepositorySnapshot{CommitID: "a", Files: []codesource.RepositoryFile{{Path: "chart/Chart.yaml", Content: []byte("apiVersion: v2\nname: helpers\ntype: library\nversion: 1.0.0\n")}}}, path: "chart"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := buildPreview(test.snapshot, test.path); err == nil {
				t.Fatal("unsafe chart input was accepted")
			}
		})
	}
}

func TestImportRequiresMatchingPreviewCommit(t *testing.T) {
	services := &fakeServices{service: &appsvc.AppService{ID: 7, DeliveryType: appsvc.DeliveryTypeHelm, CodeSourceID: 9}}
	sources := &fakeSources{snapshot: chartSnapshot()}
	logic := &Logic{services: services, sources: sources}
	request := Request{CodeSourceID: 9, RepoPath: "team/payment", Ref: "main", ChartPath: "deploy/payment", Paths: []string{"deploy/payment/Chart.yaml"}}

	request.ExpectedCommit = "old-commit"
	if _, code := logic.Import(context.Background(), 5, 7, request); code != errcode.CodeParamInvalid {
		t.Fatalf("commit mismatch code=%d", code)
	}
	if services.createdRevision != nil {
		t.Fatal("revision was created after commit mismatch")
	}

	request.ExpectedCommit = "d34db33f"
	revision, code := logic.Import(context.Background(), 5, 7, request)
	if code != errcode.CodeSuccess || revision == nil || services.createdRevision == nil {
		t.Fatalf("import failed: code=%d revision=%+v", code, revision)
	}
	if services.createdRevision.SourceCommit != "d34db33f" || services.createdRevision.ComponentCount != 2 {
		t.Fatalf("unexpected revision request: %+v", services.createdRevision)
	}
}

func TestPreviewRejectsNonHelmServiceBeforeRepositoryRead(t *testing.T) {
	services := &fakeServices{service: &appsvc.AppService{ID: 7, DeliveryType: appsvc.DeliveryTypeK8sYAML, CodeSourceID: 9}}
	sources := &fakeSources{snapshot: chartSnapshot()}
	logic := &Logic{services: services, sources: sources}
	if _, code := logic.Preview(context.Background(), 5, 7, Request{CodeSourceID: 9}); code != errcode.CodeParamInvalid {
		t.Fatalf("unexpected code=%d", code)
	}
	if sources.calls != 0 {
		t.Fatal("repository was read for a non-Helm service")
	}
}

func TestNativeDirectoryImportGuardsPlatformChangesAndRecordsSource(t *testing.T) {
	services := &fakeServices{service: &appsvc.AppService{ID: 7, DeliveryType: appsvc.DeliveryTypeHelm, CodeSourceID: 9, CurrentRevisionID: 12, SourceRef: json.RawMessage(`{"helm_repository":{"revision_id":10}}`)}}
	sources := &fakeSources{snapshot: chartSnapshot()}
	logic := &Logic{services: services, sources: sources}
	request := Request{CodeSourceID: 9, RepoPath: "team/payment", Ref: "main", ChartPath: "deploy/payment"}
	preview, code := logic.Preview(context.Background(), 5, 7, request)
	if code != 0 || len(preview.Files) != 3 || !preview.LocalChanges || preview.CurrentRevisionID != 12 {
		t.Fatalf("preview=%+v code=%d", preview, code)
	}
	request.ExpectedCommit = "d34db33f"
	if _, code = logic.Import(context.Background(), 5, 7, request); code != errcode.CodeServiceRevisionChanged {
		t.Fatalf("unguarded import=%d", code)
	}
	current := int64(12)
	request.ExpectedRevisionID = &current
	if _, code = logic.Import(context.Background(), 5, 7, request); code != errcode.CodeParamInvalid {
		t.Fatalf("local edit overwritten=%d", code)
	}
	request.AcceptLocalChanges = true
	if _, code = logic.Import(context.Background(), 5, 7, request); code != 0 {
		t.Fatalf("confirmed import=%d", code)
	}
	if services.createdRevision.RepositoryBinding == nil || services.createdRevision.RepositoryBinding.ChartPath != "deploy/payment" || *services.createdRevision.ExpectedRevisionID != 12 {
		t.Fatal("source binding or concurrency guard missing")
	}
	services.service.CurrentRevisionID = 13
	if _, code = logic.Import(context.Background(), 5, 7, request); code != errcode.CodeServiceRevisionChanged {
		t.Fatalf("concurrent update accepted=%d", code)
	}
}

func TestNativeImportRejectsMissingPackagedDependencies(t *testing.T) {
	snapshot := chartSnapshot()
	snapshot.Files[2].Content = []byte("apiVersion: v2\nname: payment\nversion: 1.2.3\ndependencies:\n- name: common\n  version: 1.0.0\n  repository: https://charts.example.com\n")
	logic := &Logic{services: &fakeServices{service: &appsvc.AppService{ID: 7, DeliveryType: appsvc.DeliveryTypeHelm, CodeSourceID: 9}}, sources: &fakeSources{snapshot: snapshot}}
	if _, code := logic.Preview(context.Background(), 5, 7, Request{CodeSourceID: 9, ChartPath: "deploy/payment"}); code != errcode.CodeServiceDefinitionInvalid {
		t.Fatalf("missing dependency accepted=%d", code)
	}
}
