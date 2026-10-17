package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/secretbox"
)

type SQLExecutionResolver struct {
	repo      *Repository
	namespace string
	box       *secretbox.Box
}

func NewSQLExecutionResolver(db *sql.DB, namespace string, box ...*secretbox.Box) *SQLExecutionResolver {
	if namespace == "" {
		namespace = "axiops-cicd"
	}
	resolver := &SQLExecutionResolver{repo: NewRepository(db), namespace: namespace}
	if len(box) > 0 {
		resolver.box = box[0]
	}
	return resolver
}

func (r *SQLExecutionResolver) BuildRequest(ctx context.Context, buildRunID int64) (ProviderBuildRequest, error) {
	run, err := r.repo.FindRunByID(ctx, buildRunID)
	if err != nil {
		return ProviderBuildRequest{}, err
	}
	if run == nil {
		return ProviderBuildRequest{}, ErrRunNotFound
	}
	if run.Status != StatusDispatching {
		return ProviderBuildRequest{}, fmt.Errorf("build run %d is not dispatching", buildRunID)
	}
	if run.ConfigSnapshot == nil {
		return ProviderBuildRequest{}, fmt.Errorf("build run %d has no execution snapshot", buildRunID)
	}
	var snapshot BuildExecutionSnapshot
	if err := json.Unmarshal([]byte(*run.ConfigSnapshot), &snapshot); err != nil {
		return ProviderBuildRequest{}, fmt.Errorf("decode build execution snapshot: %w", err)
	}
	request, err := providerRequestFromSnapshot(*run, snapshot, r.namespace)
	if err != nil {
		return ProviderBuildRequest{}, err
	}
	if snapshot.CodeSourceID > 0 {
		request.GitCredential, err = r.loadGitCredential(ctx, snapshot.CodeSourceID, run.ProjectID)
		if err != nil {
			return ProviderBuildRequest{}, err
		}
	}
	if request.ImageURL != "" {
		auth, err := NewSQLRegistryAuthStore(r.repo.db, r.box).Load(ctx, run.ProjectID, run.ServiceID)
		if err != nil {
			return ProviderBuildRequest{}, fmt.Errorf("load build registry credential: %w", err)
		}
		// Existing catalog entries and credentials can predate the UI check or
		// expire after they were saved. Verify again before creating a provider
		// job so an invalid registry never consumes a K8s/host build slot.
		if auth != nil {
			if err := testRegistryConnectivity(ctx, auth.BaseURL, auth); err != nil {
				return ProviderBuildRequest{}, fmt.Errorf("verify image registry before build: %w", err)
			}
		}
		request.RegistryDockerConfig, err = registryDockerConfig(request.ImageURL, auth)
		if err != nil {
			return ProviderBuildRequest{}, err
		}
	}
	return request, nil
}

func (r *SQLExecutionResolver) loadGitCredential(ctx context.Context, codeSourceID, projectID int64) (string, error) {
	if r.box == nil {
		return "", fmt.Errorf("credential decryptor is not configured")
	}
	var encrypted, credentialType string
	err := r.repo.db.QueryRowContext(ctx, `SELECT cred.data, cred.type
		FROM code_source source JOIN credential cred ON cred.id=source.credential_id
		WHERE source.id=? AND source.status='active' AND source.project_id IN (0, ?)
		  AND cred.project_id IN (0, ?)`, codeSourceID, projectID, projectID).Scan(&encrypted, &credentialType)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("active code source credential was not found")
	}
	if err != nil {
		return "", err
	}
	if credentialType != "git_token" {
		return "", fmt.Errorf("code source credential type must be git_token")
	}
	plaintext, err := r.box.Decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt code source credential: authentication failed")
	}
	var data struct {
		Token string `json:"token"`
	}
	if json.Unmarshal([]byte(plaintext), &data) != nil || strings.TrimSpace(data.Token) == "" {
		return "", fmt.Errorf("git_token credential requires token")
	}
	return data.Token, nil
}

func providerRequestFromSnapshot(run BuildRun, snapshot BuildExecutionSnapshot, namespace string) (ProviderBuildRequest, error) {
	if snapshot.RepositoryURL == "" {
		return ProviderBuildRequest{}, fmt.Errorf("repository URL is missing from build execution snapshot")
	}
	request := ProviderBuildRequest{
		BuildRunID: run.ID, ServiceID: run.ServiceID, ProjectID: run.ProjectID,
		RepositoryURL: snapshot.RepositoryURL, Branch: run.GitBranch, Commit: run.GitCommit,
		ImageURL: run.ImageURL, Namespace: namespace, ConfigSnapshot: valueOrEmpty(run.ConfigSnapshot),
	}
	if snapshot.Config != nil {
		// The comparison baseline is frozen when this build is queued.
		request.BuilderImage = snapshot.Config.BuildImage
		request.BuildMode = snapshot.Config.BuildMode
		request.BuildCommand = snapshot.Config.BuildCommand
		if request.BuildMode == "jar_image" {
			image, command := DefaultJarCompileImage, DefaultJarCompileCommand
			if snapshot.Config.TemplateType == "gradle" {
				image = "gradle:8-jdk17"
				command = `gradle --no-daemon --gradle-user-home "${CACHE_DIR:-/workspace/.cache/gradle}" clean build`
			}
			request.BuilderImage = defaultString(request.BuilderImage, image)
			request.BuildCommand = defaultString(request.BuildCommand, command)
		}
		request.DockerfilePath = snapshot.Config.DockerfilePath
		request.DockerfileSource = snapshot.Config.DockerfileSource
		request.DockerfileContent = snapshot.Config.DockerfileContent
		request.BuildContext = snapshot.Config.BuildContext
		request.ImageRepo = snapshot.Config.ImageRepo
		request.CacheEnabled = snapshot.Config.CacheEnabled
		request.CacheDir = snapshot.Config.CacheDir
		request.CacheStrategy = snapshot.Config.CacheStrategy
		request.CachePVCName = snapshot.Config.CachePVCName
		request.TimeoutSeconds = snapshot.Config.TimeoutSeconds
		request.ResourceCPU = snapshot.Config.ResourceCPU
		request.ResourceMemory = snapshot.Config.ResourceMemory
		if _, err := buildResources(request.ResourceCPU, request.ResourceMemory); err != nil {
			return ProviderBuildRequest{}, fmt.Errorf("服务 %d 的构建配置快照无效: %w；请修改服务构建配置中的 CPU/内存（如 2Gi，或清空继承平台），保存后创建新构建。仅清空平台 CI 配置不会覆盖服务已有值", run.ServiceID, err)
		}
		if snapshot.Config.EnvVars != nil && strings.TrimSpace(*snapshot.Config.EnvVars) != "" {
			if err := json.Unmarshal([]byte(*snapshot.Config.EnvVars), &request.EnvVars); err != nil {
				return ProviderBuildRequest{}, fmt.Errorf("decode build environment variables: %w", err)
			}
			seen := make(map[string]struct{}, len(request.EnvVars))
			for _, env := range request.EnvVars {
				name := strings.TrimSpace(env.Key)
				if name == "" {
					return ProviderBuildRequest{}, fmt.Errorf("build environment variable name is required")
				}
				if _, exists := seen[name]; exists {
					return ProviderBuildRequest{}, fmt.Errorf("duplicate build environment variable %q", name)
				}
				seen[name] = struct{}{}
			}
		}
	}
	if snapshot.ChangeBaseline != nil {
		request.ChangeBaseCommit = snapshot.ChangeBaseline.Commit
	}
	return request, nil
}

func (r *SQLExecutionResolver) Execution(ctx context.Context, buildRunID int64) (ProviderExecution, error) {
	run, err := r.repo.FindRunByID(ctx, buildRunID)
	if err != nil {
		return ProviderExecution{}, err
	}
	if run == nil {
		return ProviderExecution{}, ErrRunNotFound
	}
	return ProviderExecution{BuildRunID: run.ID, ExternalID: run.JobName, JobName: run.JobName, PodName: run.PodName, NodeName: run.NodeName}, nil
}

func (r *SQLExecutionResolver) MarkSubmitted(ctx context.Context, buildRunID int64, execution ProviderExecution) error {
	execution.JobName = defaultString(execution.JobName, execution.ExternalID)
	if execution.JobName == "" {
		return fmt.Errorf("submitted build job name is required")
	}
	// Persist identity even if cancellation won the race. Never turn a
	// cancelling/terminal run back into a running run.
	_, err := r.repo.db.ExecContext(ctx, `UPDATE build_run SET job_name=?,
		pod_name=IF(?='',pod_name,?),node_name=IF(?='',node_name,?),
		started_at=COALESCE(started_at,CURRENT_TIMESTAMP),
		status=IF(status='dispatching','running',status)
		WHERE id=? AND status IN ('dispatching','running','cancelling')`, execution.JobName, execution.PodName, execution.PodName, execution.NodeName, execution.NodeName, buildRunID)
	return err
}

func (r *SQLExecutionResolver) MarkCancelled(ctx context.Context, buildRunID int64) error {
	if err := r.repo.TransitionRunStatus(ctx, buildRunID, StatusCancelled, ""); err != nil {
		return err
	}
	_, err := r.repo.db.ExecContext(ctx, `UPDATE build_command
		SET status='failed', last_error='build cancelled before submission'
		WHERE build_run_id=? AND command_type='submit' AND status='pending'`, buildRunID)
	return err
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
