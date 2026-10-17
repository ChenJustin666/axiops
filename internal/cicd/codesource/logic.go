package codesource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
	"github.com/ChenJustin666/axiops/pkg/gitprovider"
)

var (
	ErrNotFound     = errors.New("code source not found")
	ErrNoPermission = errors.New("code source permission denied")
	ErrInvalid      = errors.New("invalid code source")
)

type store interface {
	Create(context.Context, *CodeSource) (int64, error)
	FindByID(context.Context, int64) (*CodeSource, error)
	List(context.Context, int64, authz.ScopeSet) ([]CodeSource, error)
	Update(context.Context, *CodeSource) error
	Delete(context.Context, int64) error
	FindCredential(context.Context, int64) (*CredentialRef, error)
}

type Logic struct {
	repo            store
	authorizer      authz.Checker
	box             *secretbox.Box
	providerFactory func(*gitprovider.Config) (gitprovider.Provider, error)
}

func NewLogic(db *sql.DB, authorizer authz.Checker, encryptionKey string) *Logic {
	box, _ := secretbox.New(encryptionKey)
	return &Logic{repo: NewRepository(db), authorizer: authorizer, box: box, providerFactory: gitprovider.New}
}

func (l *Logic) Create(ctx context.Context, userID, projectID int64, request CreateRequest) (int64, error) {
	if !validIdentity(request.Name, request.Type, request.URL) || request.CredentialID <= 0 {
		return 0, ErrInvalid
	}
	if err := l.authorize(ctx, userID, "create", projectID, 0); err != nil {
		return 0, err
	}
	if _, err := l.credentialToken(ctx, userID, request.CredentialID, projectID); err != nil {
		return 0, err
	}
	if request.WebhookCredentialID > 0 {
		if err := l.validateWebhookCredential(ctx, userID, request.WebhookCredentialID, projectID); err != nil {
			return 0, err
		}
	}
	return l.repo.Create(ctx, &CodeSource{ProjectID: projectID, Name: strings.TrimSpace(request.Name), Type: request.Type,
		URL: strings.TrimRight(strings.TrimSpace(request.URL), "/"), CredentialID: request.CredentialID, WebhookCredentialID: request.WebhookCredentialID, Status: "active", CreatedBy: userID})
}

func (l *Logic) Get(ctx context.Context, userID, id int64) (*CodeSource, error) {
	source, err := l.repo.FindByID(ctx, id)
	if err != nil || source == nil {
		if err == nil {
			err = ErrNotFound
		}
		return nil, err
	}
	if err := l.authorize(ctx, userID, "view", source.ProjectID, source.ID); err != nil {
		return nil, err
	}
	return source, nil
}

func (l *Logic) List(ctx context.Context, userID, projectID int64) ([]CodeSource, error) {
	scope, err := l.authorizer.Scopes(ctx, userID, "code_source", "view")
	if err != nil {
		return nil, err
	}
	if !scope.HasAnyAllow() {
		return nil, ErrNoPermission
	}
	return l.repo.List(ctx, projectID, scope)
}

func (l *Logic) Update(ctx context.Context, userID, id int64, request UpdateRequest) error {
	source, err := l.repo.FindByID(ctx, id)
	if err != nil || source == nil {
		if err == nil {
			return ErrNotFound
		}
		return err
	}
	if err := l.authorize(ctx, userID, "update", source.ProjectID, source.ID); err != nil {
		return err
	}
	if request.Name != "" {
		source.Name = strings.TrimSpace(request.Name)
	}
	if request.Type != "" {
		source.Type = strings.TrimSpace(request.Type)
	}
	if request.URL != "" {
		source.URL = strings.TrimRight(strings.TrimSpace(request.URL), "/")
	}
	if request.CredentialID > 0 {
		if _, err := l.credentialToken(ctx, userID, request.CredentialID, source.ProjectID); err != nil {
			return err
		}
		source.CredentialID = request.CredentialID
	}
	if request.WebhookCredentialID > 0 {
		if err := l.validateWebhookCredential(ctx, userID, request.WebhookCredentialID, source.ProjectID); err != nil {
			return err
		}
		source.WebhookCredentialID = request.WebhookCredentialID
	}
	if request.Status != "" {
		if request.Status != "active" && request.Status != "inactive" {
			return ErrInvalid
		}
		source.Status = request.Status
	}
	if !validIdentity(source.Name, source.Type, source.URL) {
		return ErrInvalid
	}
	return l.repo.Update(ctx, source)
}

func (l *Logic) Delete(ctx context.Context, userID, id int64) error {
	source, err := l.repo.FindByID(ctx, id)
	if err != nil || source == nil {
		if err == nil {
			return ErrNotFound
		}
		return err
	}
	if err := l.authorize(ctx, userID, "delete", source.ProjectID, source.ID); err != nil {
		return err
	}
	return l.repo.Delete(ctx, id)
}

func (l *Logic) Check(ctx context.Context, userID, id int64) error {
	source, err := l.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := l.authorize(ctx, userID, "test", source.ProjectID, source.ID); err != nil {
		return err
	}
	provider, err := l.provider(ctx, userID, source)
	if err != nil {
		return err
	}
	// A constructor-only check misses bad endpoints and expired tokens. A
	// bounded repository request verifies the complete connection path.
	_, err = provider.ListRepositories(ctx)
	return err
}

func (l *Logic) ListRepositories(ctx context.Context, userID, id int64) ([]gitprovider.Repository, error) {
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return provider.ListRepositories(ctx)
}

// ListRepoFiles returns one directory level so the UI can build a bounded
// repository browser without exposing credentials or cloning the repository.
func (l *Logic) ListRepoFiles(ctx context.Context, userID, id int64, repoPath, ref, dirPath string) ([]gitprovider.RepoFile, error) {
	if id <= 0 || strings.TrimSpace(repoPath) == "" || strings.TrimSpace(ref) == "" {
		return nil, ErrInvalid
	}
	cleanPath := path.Clean(strings.TrimSpace(dirPath))
	if cleanPath == "." {
		cleanPath = ""
	}
	if strings.HasPrefix(cleanPath, "/") || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return provider.ListRepoFiles(ctx, repoPath, ref, cleanPath)
}

func (l *Logic) ListBranches(ctx context.Context, userID, id int64, repoPath string) ([]gitprovider.Branch, error) {
	if id <= 0 || strings.TrimSpace(repoPath) == "" {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return provider.ListBranches(ctx, repoPath)
}

func (l *Logic) ListTags(ctx context.Context, userID, id int64, repoPath string) ([]gitprovider.Tag, error) {
	if id <= 0 || strings.TrimSpace(repoPath) == "" {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return provider.ListTags(ctx, repoPath)
}

func (l *Logic) GetCommit(ctx context.Context, userID, id int64, repoPath, ref string) (*gitprovider.Commit, error) {
	if id <= 0 || strings.TrimSpace(repoPath) == "" || strings.TrimSpace(ref) == "" {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return provider.GetCommit(ctx, repoPath, ref)
}

// ReadFiles resolves ref to an immutable commit before reading any content.
// Limits protect the API from using repository access as an unbounded file proxy.
func (l *Logic) ReadFiles(ctx context.Context, userID, id int64, repoPath, ref string, paths []string) (*RepositorySnapshot, error) {
	if id <= 0 || strings.TrimSpace(repoPath) == "" || strings.TrimSpace(ref) == "" || len(paths) == 0 || len(paths) > 50 {
		return nil, ErrInvalid
	}
	provider, err := l.authorizedProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	source, err := l.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	commit, err := provider.GetCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}
	if commit == nil || strings.TrimSpace(commit.ID) == "" {
		return nil, ErrInvalid
	}
	snapshot := &RepositorySnapshot{Source: *source, RepoPath: strings.TrimSpace(repoPath), Ref: strings.TrimSpace(ref), CommitID: commit.ID}
	total := 0
	seen := map[string]bool{}
	for _, raw := range paths {
		clean := path.Clean(strings.TrimSpace(raw))
		if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") || seen[clean] {
			return nil, ErrInvalid
		}
		seen[clean] = true
		content, err := provider.GetFileContent(ctx, repoPath, clean, commit.ID)
		if err != nil {
			return nil, err
		}
		if len(content) > 1<<20 {
			return nil, ErrInvalid
		}
		total += len(content)
		if total > 4<<20 {
			return nil, ErrInvalid
		}
		snapshot.Files = append(snapshot.Files, RepositoryFile{Path: clean, Content: content})
	}
	return snapshot, nil
}

func (l *Logic) authorizedProvider(ctx context.Context, userID, id int64) (gitprovider.Provider, error) {
	source, err := l.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if source.Status != "active" {
		return nil, ErrInvalid
	}
	return l.provider(ctx, userID, source)
}

func (l *Logic) provider(ctx context.Context, userID int64, source *CodeSource) (gitprovider.Provider, error) {
	token, err := l.credentialToken(ctx, userID, source.CredentialID, source.ProjectID)
	if err != nil {
		return nil, err
	}
	factory := l.providerFactory
	if factory == nil {
		factory = gitprovider.New
	}
	return factory(&gitprovider.Config{Type: source.Type, BaseURL: source.URL, Token: token})
}

func (l *Logic) credentialToken(ctx context.Context, userID, credentialID, projectID int64) (string, error) {
	if l.box == nil {
		return "", fmt.Errorf("credential decryptor is not configured")
	}
	credential, err := l.repo.FindCredential(ctx, credentialID)
	if err != nil || credential == nil {
		if err == nil {
			return "", ErrInvalid
		}
		return "", err
	}
	if credential.Type != "git_token" || (credential.ProjectID != 0 && credential.ProjectID != projectID) {
		return "", ErrInvalid
	}
	// Referencing a credential is a read operation even though the plaintext is
	// never returned to the caller.
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "credential", Action: "view", ProjectID: projectID})
	if err != nil {
		return "", err
	}
	if !allowed && projectID == 0 {
		allowed, err = l.hasProjectScopedAccess(ctx, userID, "credential", "view")
		if err != nil {
			return "", err
		}
	}
	if !allowed {
		return "", ErrNoPermission
	}
	plaintext, err := l.box.Decrypt(credential.Data)
	if err != nil {
		return "", fmt.Errorf("decrypt code source credential: authentication failed")
	}
	var data struct {
		Token string `json:"token"`
	}
	if json.Unmarshal([]byte(plaintext), &data) != nil || strings.TrimSpace(data.Token) == "" {
		return "", ErrInvalid
	}
	return data.Token, nil
}

func (l *Logic) validateWebhookCredential(ctx context.Context, userID, credentialID, projectID int64) error {
	credential, err := l.repo.FindCredential(ctx, credentialID)
	if err != nil || credential == nil || credential.Type != "git_webhook_secret" || (credential.ProjectID != 0 && credential.ProjectID != projectID) {
		return ErrInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "credential", Action: "view", ProjectID: projectID})
	if err != nil {
		return err
	}
	if !allowed {
		return ErrNoPermission
	}
	return nil
}

func (l *Logic) authorize(ctx context.Context, userID int64, action string, projectID, sourceID int64) error {
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "code_source", Action: action, ProjectID: projectID})
	if err != nil {
		return err
	}
	if !allowed {
		// A global source can be consumed by project-scoped users. Mutation
		// operations remain platform-only because they do not use this fallback.
		if projectID == 0 && (action == "view" || action == "test") {
			allowed, err = l.hasProjectScopedAccess(ctx, userID, "code_source", action)
			if err != nil {
				return err
			}
		}
		if !allowed {
			return ErrNoPermission
		}
	}
	return nil
}

func (l *Logic) hasProjectScopedAccess(ctx context.Context, userID int64, resource, action string) (bool, error) {
	scope, err := l.authorizer.Scopes(ctx, userID, resource, action)
	if err != nil {
		return false, err
	}
	return scope.AllowAll || len(scope.ProjectIDs) > 0, nil
}

func validIdentity(name, providerType, rawURL string) bool {
	if strings.TrimSpace(name) == "" || len(strings.TrimSpace(name)) > 64 {
		return false
	}
	switch providerType {
	case "gitlab", "github", "gitee", "gitea":
	default:
		return false
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	// Internal GitLab/Gitea deployments commonly expose plain HTTP. TLS is
	// recommended for production, but transport security is controlled by the
	// deployment network and must not make private repositories unusable.
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil
}
