package build

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Provider executes a durable build command. Implementations must treat
// IdempotencyKey as stable across retries and return the same execution.
type Provider interface {
	Name() string
	Submit(context.Context, ProviderBuildRequest) (ProviderExecution, error)
	Cancel(context.Context, ProviderExecution) error
}

type ProviderObserver interface {
	Status(context.Context, ProviderExecution) (ProviderState, error)
}

// Recover must look up the stable run identity without needing fresh source or
// registry credentials. Host executors must use a durable run ledger too.
type ProviderRecoverer interface {
	Recover(context.Context, int64, string) (ProviderExecution, bool, error)
}

// Finalize is idempotent and only runs after terminal state/logs are durable.
// Providers must keep completed execution evidence until this acknowledgement.
type ProviderFinalizer interface {
	Finalize(context.Context, ProviderExecution) error
}

var ErrExecutionMissing = errors.New("build execution no longer exists")

// An ambiguous transport failure must not free capacity or rerun user scripts.
type ExecutionUncertainError struct{ Cause error }

func (e *ExecutionUncertainError) Error() string {
	return "execution outcome unconfirmed; recovering original build: " + e.Cause.Error()
}
func (e *ExecutionUncertainError) Unwrap() error { return e.Cause }

type ProviderState struct {
	GitChanges     *GitChangeSet
	SourceCommit   string
	LogsIncomplete bool
	Status         string
	PodName        string
	NodeName       string
	ErrorMessage   string
	LogText        string
	DiagnosticText string
	LogTruncated   bool
	ImageDigest    string
	ImageSize      int64
	ArtifactURL    string
	ArtifactSHA256 string
	ArtifactSize   int64
}

type ProviderBuildRequest struct {
	ChangeBaseCommit  string
	ExecutionPolicy   ExecutionPolicy
	BuildRunID        int64
	ServiceID         int64
	ProjectID         int64
	RepositoryURL     string
	Branch            string
	Commit            string
	ImageURL          string
	Namespace         string
	BuilderImage      string
	BuildCommand      string
	DockerfilePath    string
	DockerfileSource  string
	DockerfileContent string
	BuildContext      string
	ImageRepo         string
	CacheEnabled      bool
	CacheDir          string
	CacheStrategy     string
	CachePVCName      string
	EnvVars           []EnvVar
	TimeoutSeconds    int
	ResourceCPU       string
	ResourceMemory    string
	ConfigSnapshot    string
	BuildMode         string
	// JAR mode compiles source in an init container, then uses the platform
	// image builder for Dockerfile build/push and the immutable result contract.
	CompileImage         string
	CompileCommand       string
	GitCredential        string `json:"-"`
	RegistryDockerConfig string `json:"-"`
	IdempotencyKey       string
}

type ProviderExecution struct {
	BuildRunID int64
	ExternalID string
	JobName    string
	PodName    string
	NodeName   string
}

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewProviderRegistry(providers ...Provider) (*ProviderRegistry, error) {
	r := &ProviderRegistry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		if err := r.Register(provider); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *ProviderRegistry) Register(provider Provider) error {
	if provider == nil || provider.Name() == "" {
		return fmt.Errorf("build provider name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[provider.Name()]; exists {
		return fmt.Errorf("build provider %q is already registered", provider.Name())
	}
	r.providers[provider.Name()] = provider
	return nil
}

func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	provider, ok := r.providers[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("build provider %q is not registered", name)
	}
	return provider, nil
}
