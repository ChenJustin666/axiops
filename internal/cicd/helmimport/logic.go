package helmimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/helmconfig"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"gopkg.in/yaml.v3"
)

const maxDefinitionBytes = 4 << 20

var (
	chartNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	semverPattern    = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
)

type Request struct {
	CodeSourceID       int64    `json:"code_source_id"`
	RepoPath           string   `json:"repo_path"`
	Ref                string   `json:"ref"`
	ChartPath          string   `json:"chart_path"`
	Paths              []string `json:"paths"`
	ExpectedCommit     string   `json:"expected_commit,omitempty"`
	ExpectedRevisionID *int64   `json:"expected_revision_id,omitempty"`
	AcceptLocalChanges bool     `json:"accept_local_changes,omitempty"`
}

type ChartDependency struct {
	Name       string   `json:"name" yaml:"name"`
	Version    string   `json:"version" yaml:"version"`
	Repository string   `json:"repository,omitempty" yaml:"repository"`
	Condition  string   `json:"condition,omitempty" yaml:"condition"`
	Tags       []string `json:"tags,omitempty" yaml:"tags"`
}

type ChartMetadata struct {
	APIVersion   string            `json:"api_version" yaml:"apiVersion"`
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description,omitempty" yaml:"description"`
	Type         string            `json:"type" yaml:"type"`
	Version      string            `json:"version" yaml:"version"`
	AppVersion   string            `json:"app_version,omitempty" yaml:"appVersion"`
	KubeVersion  string            `json:"kube_version,omitempty" yaml:"kubeVersion"`
	Dependencies []ChartDependency `json:"dependencies,omitempty" yaml:"dependencies"`
}

type ChartFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
}

type Definition struct {
	SchemaVersion int           `json:"schema_version"`
	ChartPath     string        `json:"chart_path"`
	Metadata      ChartMetadata `json:"metadata"`
	Files         []ChartFile   `json:"files"`
}

type Preview struct {
	CommitID          string        `json:"commit_id"`
	ContentHash       string        `json:"content_hash"`
	ChartPath         string        `json:"chart_path"`
	Metadata          ChartMetadata `json:"metadata"`
	FileCount         int           `json:"file_count"`
	TemplateCount     int           `json:"template_count"`
	ComponentCount    int           `json:"component_count"`
	DefaultValueKeys  []string      `json:"default_value_keys"`
	Images            []string      `json:"images"`
	Definition        string        `json:"definition,omitempty"`
	Files             []ChartFile   `json:"files"`
	CurrentRevisionID int64         `json:"current_revision_id"`
	LocalChanges      bool          `json:"local_changes"`
}

type serviceGateway interface {
	GetByID(context.Context, int64, int64) (*appsvc.AppService, int)
	CreateRevision(context.Context, int64, int64, *appsvc.CreateServiceRevisionRequest) (*appsvc.ServiceRevision, int)
}

type sourceGateway interface {
	ReadFiles(context.Context, int64, int64, string, string, []string) (*codesource.RepositorySnapshot, error)
}

type Logic struct {
	services serviceGateway
	sources  sourceGateway
}

func NewLogic(services *appsvc.Logic, sources *codesource.Logic) *Logic {
	return &Logic{services: services, sources: sources}
}

func (l *Logic) Preview(ctx context.Context, userID, serviceID int64, request Request) (*Preview, int) {
	service, code := l.services.GetByID(ctx, userID, serviceID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	if service.DeliveryType != appsvc.DeliveryTypeHelm || service.CodeSourceID <= 0 || request.CodeSourceID != service.CodeSourceID {
		return nil, errcode.CodeParamInvalid
	}
	var snapshot *codesource.RepositorySnapshot
	var err error
	if len(request.Paths) == 0 {
		source, ok := l.sources.(interface {
			ReadChartFiles(context.Context, int64, int64, string, string, string) (*codesource.RepositorySnapshot, error)
		})
		if !ok {
			return nil, errcode.CodeParamInvalid
		}
		snapshot, err = source.ReadChartFiles(ctx, userID, request.CodeSourceID, request.RepoPath, request.Ref, request.ChartPath)
	} else {
		snapshot, err = l.sources.ReadFiles(ctx, userID, request.CodeSourceID, request.RepoPath, request.Ref, request.Paths)
	}
	if err != nil {
		return nil, mapSourceError(err)
	}
	preview, err := buildPreview(snapshot, request.ChartPath)
	if err != nil {
		return nil, errcode.CodeParamInvalid
	}
	if _, err = helmconfig.LoadChart([]byte(preview.Definition)); err != nil {
		return nil, errcode.CodeServiceDefinitionInvalid
	}
	preview.CurrentRevisionID = service.CurrentRevisionID
	var origin struct {
		Repository *appsvc.HelmRepositoryBinding `json:"helm_repository"`
	}
	_ = json.Unmarshal(service.SourceRef, &origin)
	preview.LocalChanges = service.CurrentRevisionID > 0 && (origin.Repository == nil || origin.Repository.RevisionID != service.CurrentRevisionID)
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
	if request.ExpectedRevisionID == nil {
		zero := int64(0)
		request.ExpectedRevisionID = &zero
	}
	if *request.ExpectedRevisionID != preview.CurrentRevisionID {
		return nil, errcode.CodeServiceRevisionChanged
	}
	if preview.LocalChanges && !request.AcceptLocalChanges {
		return nil, errcode.CodeParamInvalid
	}
	return l.services.CreateRevision(ctx, userID, serviceID, &appsvc.CreateServiceRevisionRequest{
		SourceCommit: preview.CommitID, Definition: preview.Definition, ComponentCount: preview.ComponentCount,
		ExpectedRevisionID: request.ExpectedRevisionID,
		RepositoryBinding:  &appsvc.HelmRepositoryBinding{CodeSourceID: request.CodeSourceID, RepoPath: strings.TrimSpace(request.RepoPath), Ref: strings.TrimSpace(request.Ref), ChartPath: preview.ChartPath, CommitID: preview.CommitID},
	})
}

func buildPreview(snapshot *codesource.RepositorySnapshot, rawChartPath string) (*Preview, error) {
	if snapshot == nil || strings.TrimSpace(snapshot.CommitID) == "" {
		return nil, errors.New("repository snapshot is incomplete")
	}
	chartPath, err := cleanRelativePath(rawChartPath)
	if err != nil {
		return nil, err
	}
	files := make([]ChartFile, 0, len(snapshot.Files))
	seen := map[string]bool{}
	for _, file := range snapshot.Files {
		relative, err := chartRelativePath(chartPath, file.Path)
		if err != nil || seen[relative] || !utf8.Valid(file.Content) {
			return nil, errors.New("chart file path or content is invalid")
		}
		seen[relative] = true
		digest := sha256.Sum256(file.Content)
		files = append(files, ChartFile{Path: relative, Content: string(file.Content), SHA256: hex.EncodeToString(digest[:])})
	}
	if !seen["Chart.yaml"] {
		return nil, errors.New("Chart.yaml is required")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	metadata, err := parseMetadata(files)
	if err != nil {
		return nil, err
	}
	valueKeys, images, err := parseValues(files)
	if err != nil {
		return nil, err
	}
	templateCount := 0
	for _, file := range files {
		if strings.HasPrefix(file.Path, "templates/") {
			templateCount++
		}
	}
	componentCount := len(images)
	if componentCount == 0 {
		componentCount = 1
	}
	definitionBytes, err := json.Marshal(Definition{SchemaVersion: 1, ChartPath: chartPath, Metadata: metadata, Files: files})
	if err != nil || len(definitionBytes) > maxDefinitionBytes {
		return nil, errors.New("chart definition exceeds the size limit")
	}
	digest := sha256.Sum256(definitionBytes)
	return &Preview{
		CommitID: snapshot.CommitID, ContentHash: hex.EncodeToString(digest[:]), ChartPath: chartPath,
		Metadata: metadata, FileCount: len(files), TemplateCount: templateCount, ComponentCount: componentCount,
		DefaultValueKeys: valueKeys, Images: images, Definition: string(definitionBytes),
		Files: files,
	}, nil
}

func cleanRelativePath(raw string) (string, error) {
	if strings.Contains(raw, `\`) || strings.TrimSpace(raw) == "" {
		return "", errors.New("relative path is required")
	}
	clean := path.Clean(strings.TrimSpace(raw))
	if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("path escapes the repository")
	}
	return clean, nil
}

func chartRelativePath(chartPath, filePath string) (string, error) {
	clean, err := cleanRelativePath(filePath)
	if err != nil {
		return "", err
	}
	if chartPath == "." {
		return clean, nil
	}
	prefix := chartPath + "/"
	if !strings.HasPrefix(clean, prefix) {
		return "", errors.New("file is outside chart_path")
	}
	return strings.TrimPrefix(clean, prefix), nil
}

func parseMetadata(files []ChartFile) (ChartMetadata, error) {
	var metadata ChartMetadata
	for _, file := range files {
		if file.Path == "Chart.yaml" {
			if err := yaml.Unmarshal([]byte(file.Content), &metadata); err != nil {
				return metadata, err
			}
			break
		}
	}
	metadata.Type = strings.TrimSpace(metadata.Type)
	if metadata.Type == "" {
		metadata.Type = "application"
	}
	if metadata.APIVersion != "v2" || !chartNamePattern.MatchString(metadata.Name) || !semverPattern.MatchString(metadata.Version) || metadata.Type != "application" {
		return metadata, errors.New("only valid Helm v2 application charts are supported")
	}
	for _, dependency := range metadata.Dependencies {
		if !chartNamePattern.MatchString(dependency.Name) || strings.TrimSpace(dependency.Version) == "" {
			return metadata, errors.New("chart dependency is invalid")
		}
	}
	return metadata, nil
}

func parseValues(files []ChartFile) ([]string, []string, error) {
	values := map[string]interface{}{}
	for _, file := range files {
		if file.Path != "values.yaml" {
			continue
		}
		if err := yaml.Unmarshal([]byte(file.Content), &values); err != nil {
			return nil, nil, err
		}
		break
	}
	if hasPlaintextSecret(values) {
		return nil, nil, errors.New("values.yaml contains a plaintext sensitive default")
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	imageSet := map[string]bool{}
	collectImages(values, imageSet)
	images := make([]string, 0, len(imageSet))
	for image := range imageSet {
		images = append(images, image)
	}
	sort.Strings(images)
	return keys, images, nil
}

func hasPlaintextSecret(value interface{}) bool {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
			if normalized == "password" || normalized == "passwd" || normalized == "token" || normalized == "apikey" || normalized == "secret" {
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					return true
				}
			}
			if hasPlaintextSecret(child) {
				return true
			}
		}
	case []interface{}:
		for _, child := range typed {
			if hasPlaintextSecret(child) {
				return true
			}
		}
	}
	return false
}

func collectImages(value interface{}, images map[string]bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			if strings.EqualFold(key, "image") {
				switch image := child.(type) {
				case string:
					if strings.TrimSpace(image) != "" {
						images[strings.TrimSpace(image)] = true
					}
				case map[string]interface{}:
					repository, _ := image["repository"].(string)
					tag, _ := image["tag"].(string)
					digest, _ := image["digest"].(string)
					candidate := strings.TrimSpace(repository)
					if candidate != "" && strings.TrimSpace(digest) != "" {
						candidate += "@" + strings.TrimSpace(digest)
					} else if candidate != "" && strings.TrimSpace(tag) != "" {
						candidate += ":" + strings.TrimSpace(tag)
					}
					if candidate != "" {
						images[candidate] = true
					}
				}
			}
			collectImages(child, images)
		}
	case []interface{}:
		for _, child := range typed {
			collectImages(child, images)
		}
	}
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
