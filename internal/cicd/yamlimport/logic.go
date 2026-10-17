package yamlimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var ErrCommitChanged = errors.New("repository commit changed after preview")

type Request struct {
	CodeSourceID   int64    `json:"code_source_id"`
	RepoPath       string   `json:"repo_path"`
	Ref            string   `json:"ref"`
	Paths          []string `json:"paths"`
	ExpectedCommit string   `json:"expected_commit,omitempty"`
}
type Resource struct {
	APIVersion string   `json:"api_version"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace,omitempty"`
	Images     []string `json:"images,omitempty"`
}
type Preview struct {
	CommitID       string     `json:"commit_id"`
	ContentHash    string     `json:"content_hash"`
	ResourceCount  int        `json:"resource_count"`
	ComponentCount int        `json:"component_count"`
	Resources      []Resource `json:"resources"`
	Components     []string   `json:"components"`
	Images         []string   `json:"images"`
	Definition     string     `json:"definition,omitempty"`
}

type Logic struct {
	services *appsvc.Logic
	sources  *codesource.Logic
}

func NewLogic(services *appsvc.Logic, sources *codesource.Logic) *Logic {
	return &Logic{services: services, sources: sources}
}

func (l *Logic) Preview(ctx context.Context, userID, serviceID int64, request Request) (*Preview, int) {
	service, code := l.services.GetByID(ctx, userID, serviceID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	if service.DeliveryType != appsvc.DeliveryTypeK8sYAML || service.CodeSourceID <= 0 || request.CodeSourceID != service.CodeSourceID {
		return nil, errcode.CodeParamInvalid
	}
	snapshot, err := l.sources.ReadFiles(ctx, userID, request.CodeSourceID, request.RepoPath, request.Ref, request.Paths)
	if err != nil {
		return nil, mapSourceError(err)
	}
	definition := joinFiles(snapshot.Files)
	resources, err := workflow.ParseYAMLManifest([]byte(definition))
	if err != nil {
		return nil, errcode.CodeParamInvalid
	}
	preview := summarize(snapshot.CommitID, definition, resources)
	return preview, errcode.CodeSuccess
}

func (l *Logic) Import(ctx context.Context, userID, serviceID int64, request Request) (*appsvc.ServiceRevision, int) {
	preview, code := l.Preview(ctx, userID, serviceID, request)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	if strings.TrimSpace(request.ExpectedCommit) == "" || !strings.EqualFold(strings.TrimSpace(request.ExpectedCommit), preview.CommitID) {
		return nil, errcode.CodeParamInvalid
	}
	return l.services.CreateRevision(ctx, userID, serviceID, &appsvc.CreateServiceRevisionRequest{SourceCommit: preview.CommitID, Definition: preview.Definition, ComponentCount: preview.ComponentCount})
}

func joinFiles(files []codesource.RepositoryFile) string {
	var b strings.Builder
	for i, file := range files {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		b.Write(file.Content)
		if !strings.HasSuffix(string(file.Content), "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func summarize(commit, definition string, items []unstructured.Unstructured) *Preview {
	p := &Preview{CommitID: commit, Definition: definition, ResourceCount: len(items)}
	components := map[string]bool{}
	images := map[string]bool{}
	for _, item := range items {
		found := resourceImages(item)
		for _, image := range found {
			images[image] = true
		}
		p.Resources = append(p.Resources, Resource{APIVersion: item.GetAPIVersion(), Kind: item.GetKind(), Name: item.GetName(), Namespace: item.GetNamespace(), Images: found})
		switch item.GetKind() {
		case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
			components[item.GetName()] = true
		}
	}
	for v := range components {
		p.Components = append(p.Components, v)
	}
	for v := range images {
		p.Images = append(p.Images, v)
	}
	sort.Strings(p.Components)
	sort.Strings(p.Images)
	p.ComponentCount = len(p.Components)
	sum := sha256.Sum256([]byte(strings.TrimSpace(definition)))
	p.ContentHash = hex.EncodeToString(sum[:])
	return p
}
func resourceImages(item unstructured.Unstructured) []string {
	paths := [][]string{{"spec", "template", "spec", "containers"}, {"spec", "template", "spec", "initContainers"}, {"spec", "jobTemplate", "spec", "template", "spec", "containers"}, {"spec", "jobTemplate", "spec", "template", "spec", "initContainers"}}
	set := map[string]bool{}
	for _, path := range paths {
		containers, _, _ := unstructured.NestedSlice(item.Object, path...)
		for _, raw := range containers {
			if c, ok := raw.(map[string]interface{}); ok {
				if image, ok := c["image"].(string); ok && strings.TrimSpace(image) != "" {
					set[image] = true
				}
			}
		}
	}
	result := make([]string, 0, len(set))
	for image := range set {
		result = append(result, image)
	}
	sort.Strings(result)
	return result
}
func mapSourceError(err error) int {
	if errors.Is(err, codesource.ErrNoPermission) {
		return errcode.CodeNoPermission
	}
	if errors.Is(err, codesource.ErrNotFound) {
		return errcode.CodeCodeSourceNotFound
	}
	if errors.Is(err, codesource.ErrInvalid) {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeInternalError
}
