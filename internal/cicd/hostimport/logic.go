package hostimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/cicd/codesource"
	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"gopkg.in/yaml.v3"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type Request struct {
	CodeSourceID   int64    `json:"code_source_id"`
	RepoPath       string   `json:"repo_path"`
	Ref            string   `json:"ref"`
	SpecPath       string   `json:"spec_path"`
	Paths          []string `json:"paths"`
	ExpectedCommit string   `json:"expected_commit,omitempty"`
}

type sourceDocument struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		OS       string `yaml:"os"`
		Artifact struct {
			Source string `yaml:"source"`
			URL    string `yaml:"url"`
			SHA256 string `yaml:"sha256"`
		} `yaml:"artifact"`
		Scripts struct {
			Deploy   string `yaml:"deploy"`
			Health   string `yaml:"health"`
			Rollback string `yaml:"rollback"`
		} `yaml:"scripts"`
		TimeoutSeconds int `yaml:"timeoutSeconds"`
		Rollout        struct {
			BatchSize       int `yaml:"batchSize"`
			BatchPercent    int `yaml:"batchPercent"`
			IntervalSeconds int `yaml:"intervalSeconds"`
		} `yaml:"rollout"`
	} `yaml:"spec"`
}

type ScriptSummary struct {
	Phase  string `json:"phase"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

type Preview struct {
	CommitID        string          `json:"commit_id"`
	ContentHash     string          `json:"content_hash"`
	Name            string          `json:"name"`
	OS              string          `json:"os"`
	ArtifactSource  string          `json:"artifact_source"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	BatchSize       int             `json:"batch_size,omitempty"`
	BatchPercent    int             `json:"batch_percent,omitempty"`
	IntervalSeconds int             `json:"interval_seconds,omitempty"`
	Scripts         []ScriptSummary `json:"scripts"`
	Definition      string          `json:"definition,omitempty"`
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
	if service.DeliveryType != appsvc.DeliveryTypeHost || service.CodeSourceID <= 0 || request.CodeSourceID != service.CodeSourceID {
		return nil, errcode.CodeParamInvalid
	}
	snapshot, err := l.sources.ReadFiles(ctx, userID, request.CodeSourceID, request.RepoPath, request.Ref, request.Paths)
	if err != nil {
		return nil, mapSourceError(err)
	}
	preview, err := buildPreview(snapshot, request.SpecPath)
	if err != nil {
		return nil, errcode.CodeParamInvalid
	}
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
	return l.services.CreateRevision(ctx, userID, serviceID, &appsvc.CreateServiceRevisionRequest{
		SourceCommit: preview.CommitID, Definition: preview.Definition, ComponentCount: 1,
	})
}

func buildPreview(snapshot *codesource.RepositorySnapshot, rawSpecPath string) (*Preview, error) {
	if snapshot == nil || strings.TrimSpace(snapshot.CommitID) == "" {
		return nil, errors.New("repository snapshot is incomplete")
	}
	specPath, err := cleanRepositoryPath(rawSpecPath)
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(snapshot.Files))
	for _, file := range snapshot.Files {
		clean, err := cleanRepositoryPath(file.Path)
		if err != nil || !utf8.Valid(file.Content) || files[clean] != nil {
			return nil, errors.New("host service file is invalid")
		}
		files[clean] = file.Content
	}
	source, ok := files[specPath]
	if !ok || len(source) > 256<<10 {
		return nil, errors.New("host service definition is missing or too large")
	}
	document, err := parseDocument(source)
	if err != nil {
		return nil, err
	}
	root := path.Dir(specPath)
	osName := strings.ToLower(strings.TrimSpace(document.Spec.OS))
	if osName != "linux" && osName != "windows" {
		return nil, errors.New("host service os must be linux or windows")
	}

	deploy, deploySummary, err := resolveScript(files, root, osName, "deploy", document.Spec.Scripts.Deploy, 256<<10)
	if err != nil {
		return nil, err
	}
	health, healthSummary, err := resolveScript(files, root, osName, "health", document.Spec.Scripts.Health, 64<<10)
	if err != nil {
		return nil, err
	}
	rollback, rollbackSummary, err := resolveScript(files, root, osName, "rollback", document.Spec.Scripts.Rollback, 256<<10)
	if err != nil {
		return nil, err
	}

	artifactSource := strings.ToLower(strings.TrimSpace(document.Spec.Artifact.Source))
	artifactURL := strings.TrimSpace(document.Spec.Artifact.URL)
	artifactSHA := strings.ToLower(strings.TrimSpace(document.Spec.Artifact.SHA256))
	switch artifactSource {
	case "build":
		if artifactURL != "" || artifactSHA != "" {
			return nil, errors.New("build artifacts cannot contain a fixed URL or checksum")
		}
	case "static":
		if !validArtifact(artifactURL, artifactSHA) {
			return nil, errors.New("static artifact URL or checksum is invalid")
		}
	default:
		return nil, errors.New("artifact source must be build or static")
	}
	timeout := document.Spec.TimeoutSeconds
	if timeout == 0 {
		timeout = 900
	}
	if timeout < 30 || timeout > 86400 {
		return nil, errors.New("timeoutSeconds must be between 30 and 86400")
	}
	rollout := document.Spec.Rollout
	if rollout.BatchSize > 0 && rollout.BatchPercent > 0 || rollout.BatchSize < 0 || rollout.BatchSize > 1000 || rollout.BatchPercent < 0 || rollout.BatchPercent > 100 || rollout.IntervalSeconds < 0 || rollout.IntervalSeconds > 3600 {
		return nil, errors.New("rollout batch policy is invalid")
	}

	spec := workflow.HostSpec{
		OS: osName, ArtifactURL: artifactURL, ArtifactSHA256: artifactSHA,
		DeployScript: deploy, HealthScript: health, RollbackScript: rollback, TimeoutSeconds: timeout,
	}
	spec.Rollout.BatchSize = rollout.BatchSize
	spec.Rollout.BatchPercent = rollout.BatchPercent
	spec.Rollout.IntervalSeconds = rollout.IntervalSeconds
	definitionBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(definitionBytes)
	scripts := []ScriptSummary{deploySummary, healthSummary, rollbackSummary}
	sort.Slice(scripts, func(i, j int) bool { return scripts[i].Phase < scripts[j].Phase })
	return &Preview{
		CommitID: snapshot.CommitID, ContentHash: hex.EncodeToString(digest[:]), Name: document.Metadata.Name,
		OS: osName, ArtifactSource: artifactSource, TimeoutSeconds: timeout,
		BatchSize: rollout.BatchSize, BatchPercent: rollout.BatchPercent, IntervalSeconds: rollout.IntervalSeconds,
		Scripts: scripts, Definition: string(definitionBytes),
	}, nil
}

func parseDocument(content []byte) (*sourceDocument, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)
	var document sourceDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("host service definition must contain one YAML document")
	}
	if document.APIVersion != "axiops.io/v1alpha1" || document.Kind != "HostService" || !serviceNamePattern.MatchString(document.Metadata.Name) {
		return nil, errors.New("host service metadata is invalid")
	}
	return &document, nil
}

func cleanRepositoryPath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" || strings.Contains(raw, `\`) {
		return "", errors.New("repository path is required")
	}
	clean := path.Clean(strings.TrimSpace(raw))
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("repository path is invalid")
	}
	return clean, nil
}

func resolveScript(files map[string][]byte, root, osName, phase, rawPath string, limit int) (string, ScriptSummary, error) {
	clean, err := cleanRepositoryPath(rawPath)
	if err != nil || clean != strings.TrimSpace(rawPath) || strings.Contains(clean, "../") {
		return "", ScriptSummary{}, errors.New(phase + " script path is invalid")
	}
	extension := path.Ext(clean)
	if osName == "linux" && extension != ".sh" || osName == "windows" && extension != ".ps1" {
		return "", ScriptSummary{}, errors.New(phase + " script extension does not match the target os")
	}
	fullPath := clean
	if root != "." {
		fullPath = path.Join(root, clean)
	}
	content, ok := files[fullPath]
	if !ok || len(content) == 0 || len(content) > limit || !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0 {
		return "", ScriptSummary{}, errors.New(phase + " script is missing or invalid")
	}
	digest := sha256.Sum256(content)
	return string(content), ScriptSummary{Phase: phase, Path: clean, SHA256: hex.EncodeToString(digest[:]), Bytes: len(content)}, nil
}

func validArtifact(rawURL, checksum string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || len(checksum) != 64 {
		return false
	}
	_, err = hex.DecodeString(checksum)
	return err == nil
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
