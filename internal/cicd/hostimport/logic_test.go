package hostimport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type fakeServices struct {
	service         *appsvc.AppService
	createdRevision *appsvc.CreateServiceRevisionRequest
}

func (f *fakeServices) GetByID(context.Context, int64, int64) (*appsvc.AppService, int) {
	return f.service, errcode.CodeSuccess
}

func (f *fakeServices) CreateRevision(_ context.Context, _, serviceID int64, request *appsvc.CreateServiceRevisionRequest) (*appsvc.ServiceRevision, int) {
	f.createdRevision = request
	return &appsvc.ServiceRevision{ID: 41, ServiceID: serviceID, SourceCommit: request.SourceCommit, Definition: request.Definition}, errcode.CodeSuccess
}

type fakeSources struct {
	snapshot *codesource.RepositorySnapshot
	calls    int
}

func (f *fakeSources) ReadFiles(context.Context, int64, int64, string, string, []string) (*codesource.RepositorySnapshot, error) {
	f.calls++
	return f.snapshot, nil
}

func hostSnapshot() *codesource.RepositorySnapshot {
	definition := `apiVersion: axiops.io/v1alpha1
kind: HostService
metadata:
  name: payment
spec:
  os: linux
  artifact:
    source: build
  scripts:
    deploy: scripts/deploy.sh
    health: scripts/health.sh
    rollback: scripts/rollback.sh
  timeoutSeconds: 600
  rollout:
    batchSize: 2
    intervalSeconds: 30
`
	return &codesource.RepositorySnapshot{CommitID: "abc123", Files: []codesource.RepositoryFile{
		{Path: "deploy/host/scripts/rollback.sh", Content: []byte("#!/bin/sh\nexit 0\n")},
		{Path: "deploy/host/service.yaml", Content: []byte(definition)},
		{Path: "deploy/host/scripts/deploy.sh", Content: []byte("#!/bin/sh\ntar -xf \"$AXIOPS_ARTIFACT_PATH\"\n")},
		{Path: "deploy/host/scripts/health.sh", Content: []byte("#!/bin/sh\ncurl -f http://127.0.0.1:8080/health\n")},
	}}
}

func TestBuildPreviewFreezesValidatedScripts(t *testing.T) {
	preview, err := buildPreview(hostSnapshot(), "deploy/host/service.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Name != "payment" || preview.OS != "linux" || preview.ArtifactSource != "build" || preview.TimeoutSeconds != 600 || preview.BatchSize != 2 || preview.IntervalSeconds != 30 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if len(preview.Scripts) != 3 || preview.Scripts[0].Phase != "deploy" || len(preview.Scripts[0].SHA256) != 64 {
		t.Fatalf("unexpected script summaries: %+v", preview.Scripts)
	}
	var spec workflow.HostSpec
	if err := json.Unmarshal([]byte(preview.Definition), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OS != "linux" || !strings.Contains(spec.DeployScript, "AXIOPS_ARTIFACT_PATH") || strings.TrimSpace(spec.HealthScript) == "" || strings.TrimSpace(spec.RollbackScript) == "" || spec.Rollout.BatchSize != 2 {
		t.Fatalf("unexpected frozen host spec: %+v", spec)
	}
}

func TestBuildPreviewRejectsUnsafeDefinitions(t *testing.T) {
	tests := []struct {
		name       string
		definition string
	}{
		{name: "script traversal", definition: strings.Replace(string(hostSnapshot().Files[1].Content), "scripts/deploy.sh", "../deploy.sh", 1)},
		{name: "os extension mismatch", definition: strings.Replace(string(hostSnapshot().Files[1].Content), "os: linux", "os: windows", 1)},
		{name: "unknown inline script", definition: string(hostSnapshot().Files[1].Content) + "  inlineScript: echo unsafe\n"},
		{name: "invalid static artifact", definition: strings.Replace(string(hostSnapshot().Files[1].Content), "source: build", "source: static\n    url: http://example.com/app.tgz\n    sha256: bad", 1)},
		{name: "conflicting rollout", definition: strings.Replace(string(hostSnapshot().Files[1].Content), "batchSize: 2", "batchSize: 2\n    batchPercent: 50", 1)},
		{name: "excessive rollout interval", definition: strings.Replace(string(hostSnapshot().Files[1].Content), "intervalSeconds: 30", "intervalSeconds: 3601", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := hostSnapshot()
			snapshot.Files[1].Content = []byte(test.definition)
			if _, err := buildPreview(snapshot, "deploy/host/service.yaml"); err == nil {
				t.Fatal("unsafe host definition was accepted")
			}
		})
	}
}

func TestBuildPreviewAcceptsPinnedStaticArtifact(t *testing.T) {
	snapshot := hostSnapshot()
	static := "source: static\n    url: https://artifacts.example.com/payment.tgz\n    sha256: " + strings.Repeat("a", 64)
	snapshot.Files[1].Content = []byte(strings.Replace(string(snapshot.Files[1].Content), "source: build", static, 1))
	preview, err := buildPreview(snapshot, "deploy/host/service.yaml")
	if err != nil || preview.ArtifactSource != "static" {
		t.Fatalf("static artifact preview=%+v err=%v", preview, err)
	}
}

func TestImportRequiresMatchingCommit(t *testing.T) {
	services := &fakeServices{service: &appsvc.AppService{ID: 8, DeliveryType: appsvc.DeliveryTypeHost, CodeSourceID: 10}}
	sources := &fakeSources{snapshot: hostSnapshot()}
	logic := &Logic{services: services, sources: sources}
	request := Request{CodeSourceID: 10, RepoPath: "team/payment", Ref: "main", SpecPath: "deploy/host/service.yaml", Paths: []string{"deploy/host/service.yaml"}}

	request.ExpectedCommit = "old"
	if _, code := logic.Import(context.Background(), 5, 8, request); code != errcode.CodeParamInvalid || services.createdRevision != nil {
		t.Fatalf("commit mismatch code=%d revision=%+v", code, services.createdRevision)
	}
	request.ExpectedCommit = "abc123"
	revision, code := logic.Import(context.Background(), 5, 8, request)
	if code != errcode.CodeSuccess || revision == nil || services.createdRevision == nil || services.createdRevision.ComponentCount != 1 {
		t.Fatalf("import failed: code=%d revision=%+v request=%+v", code, revision, services.createdRevision)
	}
}

func TestPreviewRejectsNonHostServiceBeforeRepositoryRead(t *testing.T) {
	services := &fakeServices{service: &appsvc.AppService{ID: 8, DeliveryType: appsvc.DeliveryTypeHelm, CodeSourceID: 10}}
	sources := &fakeSources{snapshot: hostSnapshot()}
	logic := &Logic{services: services, sources: sources}
	if _, code := logic.Preview(context.Background(), 5, 8, Request{CodeSourceID: 10}); code != errcode.CodeParamInvalid {
		t.Fatalf("unexpected code=%d", code)
	}
	if sources.calls != 0 {
		t.Fatal("repository was read for a non-host service")
	}
}
