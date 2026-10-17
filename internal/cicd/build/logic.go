package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/cicd/appsvc"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

type Logic struct {
	repo       *Repository
	svcRepo    *appsvc.Repository
	authorizer authz.Checker
	archive    LogArchive
	decryptor  CredentialDecryptor
}

func (l *Logic) WithLogArchive(archive LogArchive) *Logic {
	l.archive = archive
	return l
}

// WithCredentialDecryptor supplies the platform credential decryptor used by
// configuration-time connectivity checks. Credential plaintext is kept in
// memory only for the duration of the check and is never returned to callers.
func (l *Logic) WithCredentialDecryptor(decryptor CredentialDecryptor) *Logic {
	l.decryptor = decryptor
	return l
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{
		repo:       NewRepository(db),
		svcRepo:    appsvc.NewRepository(db),
		authorizer: authorizer,
	}
}

// ListRuns 构建运行列表（支持按服务、项目、状态过滤）
func (l *Logic) ListRuns(ctx context.Context, userID, serviceID, projectID int64, status string, page, pageSize int) ([]BuildRun, int64, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "build", "view")
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	runs, total, err := l.repo.ListRuns(ctx, scope, serviceID, projectID, status, page, pageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	if err := l.repo.fillCommitLinks(ctx, runs); err != nil {
		return nil, 0, errcode.CodeDBError
	}
	return runs, total, errcode.CodeSuccess
}

// GetRunDetail 构建详情（含日志和阶段解析）
func (l *Logic) GetRunDetail(ctx context.Context, userID, id int64) (*BuildRunDetail, int) {
	run, err := l.repo.FindRunByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if run == nil {
		return nil, errcode.CodeBuildConfigNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "view", ProjectID: run.ProjectID, ServiceID: run.ServiceID, BuildID: run.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	runs := []BuildRun{*run}
	if err := l.repo.fillCommitLinks(ctx, runs); err != nil {
		return nil, errcode.CodeDBError
	}
	run.GitCommitURL = runs[0].GitCommitURL

	logs, _ := l.listLogs(ctx, id)

	detail := &BuildRunDetail{
		BuildRun:              *run,
		StagesParsed:          run.ParseStages(),
		GitChangedFilesParsed: run.ParseGitChangedFiles(),
		Logs:                  logs,
	}
	if run.GitChangesJSON != "" {
		if err := json.Unmarshal([]byte(run.GitChangesJSON), &detail.GitChanges); err != nil {
			return nil, errcode.CodeDBError
		}
	}
	return detail, errcode.CodeSuccess
}

// Trigger 触发构建
func (l *Logic) Trigger(ctx context.Context, req *TriggerBuildRequest, triggerUserID int64) (int64, int) {
	return l.trigger(ctx, req, triggerUserID, false)
}

func (l *Logic) trigger(ctx context.Context, req *TriggerBuildRequest, triggerUserID int64, machineAuthenticated bool) (int64, int) {
	// 获取服务信息
	svc, err := l.svcRepo.FindByID(ctx, req.ServiceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if svc == nil {
		return 0, errcode.CodeServiceNotFound
	}
	if svc.Status != appsvc.ServiceStatusActive {
		return 0, errcode.CodeParamInvalid
	}
	if !machineAuthenticated {
		allowed, err := l.authorizer.Authorize(ctx, triggerUserID, authz.Request{Resource: "build", Action: "run", ProjectID: svc.ProjectID, ServiceID: svc.ID})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}

	// 获取构建配置
	config, err := l.repo.FindConfigByServiceID(ctx, req.ServiceID)
	if err != nil {
		return 0, errcode.CodeDBError
	}

	branch := req.Branch
	if branch == "" {
		branch = svc.RepoBranch
	}
	if config != nil && config.BranchRules != nil && !branchAllowed(*config.BranchRules, branch) {
		return 0, errcode.CodeParamInvalid
	}
	if req.CustomTag != "" {
		if err := validateCustomImageTag(req.CustomTag); err != nil {
			return 0, errcode.CodeParamInvalid
		}
	}

	// Allocate numbers with a per-service row lock. Gaps are acceptable; duplicate
	// numbers are not, because they make audit and retry ordering ambiguous.
	buildNum, err := l.repo.AllocateBuildNumber(ctx, req.ServiceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[build.trigger] AllocateBuildNumber service=%d failed: %v\n", req.ServiceID, err)
		return 0, errcode.CodeDBError
	}

	// 确定 provider
	provider := "kube_job"
	if config != nil && config.Provider != "" {
		provider = normalizeBuildProvider(config.Provider)
	}

	// 生成镜像 URL
	imageURL := ""
	imageRepo := ""
	if config != nil {
		imageRepo = strings.TrimSpace(config.ImageRepo)
	}
	if imageRepo == "" {
		imageRepo, err = l.svcRepo.ResolveImageRepository(ctx, req.ServiceID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
	}
	if imageRepo != "" {
		tag := req.CustomTag
		if tag == "" {
			tag = automaticImageTag(branch, buildNum, time.Now())
		}
		imageURL = imageRepo + ":" + tag
	}

	// Freeze source and build settings. Credentials remain references and are
	// never copied into the run snapshot.
	if config != nil {
		config.ImageRepo = imageRepo
		if req.CustomCommand != "" {
			config.BuildCommand = req.CustomCommand
		}
	}
	fingerprint := BuildFingerprint(req.ServiceID, svc.RepoURL, branch, req.CommitID, config, req.CustomCommand, req.SkipTests)
	if req.CustomTag != "" || req.TriggerKey != "" {
		fingerprint = ""
	}
	if existing, err := l.repo.FindReusableRun(ctx, req.ServiceID, fingerprint); err != nil {
		return 0, errcode.CodeDBError
	} else if existing != nil {
		return existing.ID, errcode.CodeSuccess
	}
	data, _ := json.Marshal(BuildExecutionSnapshot{RepositoryURL: svc.RepoURL, CodeSourceID: svc.CodeSourceID, Fingerprint: fingerprint, Config: config})
	snapshotJSON := string(data)
	configSnapshot := &snapshotJSON

	// 初始阶段
	stages := []BuildStage{
		{Name: "git_clone", Status: "pending"},
		{Name: "build", Status: "pending"},
		{Name: "image_build", Status: "pending"},
		{Name: "image_push", Status: "pending"},
	}
	stagesJSON, _ := json.Marshal(stages)

	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "manual"
	}

	commitShort := req.CommitID
	if len(commitShort) > 8 {
		commitShort = commitShort[:8]
	}

	run := &BuildRun{
		WorkflowRunID:    req.WorkflowRunID,
		WorkflowNodeKey:  req.WorkflowNodeKey,
		ServiceID:        req.ServiceID,
		ProjectID:        svc.ProjectID,
		BuildNumber:      buildNum,
		ConfigSnapshot:   configSnapshot,
		InputFingerprint: fingerprint,
		TriggerType:      triggerType,
		TriggerUserID:    triggerUserID,
		TriggerKey:       strings.TrimSpace(req.TriggerKey),
		Status:           StatusQueued,
		Provider:         provider,
		GitBranch:        branch,
		GitCommit:        req.CommitID,
		GitCommitShort:   commitShort,
		GitAuthor:        strings.TrimSpace(req.Author),
		GitAuthorEmail:   strings.TrimSpace(req.AuthorEmail),
		GitMessage:       strings.TrimSpace(req.Message),
		ImageURL:         imageURL,
		Stages:           string(stagesJSON),
	}

	id, err := l.repo.CreateRunWithSubmitCommand(ctx, run)
	if err != nil {
		if errors.Is(err, projectstate.ErrDeleted) {
			return 0, errcode.CodeProjectDeleted
		}
		fmt.Fprintf(os.Stderr, "[build.trigger] CreateRunWithSubmitCommand service=%d branch=%q failed: %v\n", req.ServiceID, branch, err)
		return 0, errcode.CodeDBError
	}

	return id, errcode.CodeSuccess
}

// branchAllowed applies the same include/exclude semantics to manual and
// webhook triggers. Keeping this check at trigger time prevents a saved branch
// policy from being bypassed by the UI's manual "run" action.
func branchAllowed(raw, branch string) bool {
	if strings.TrimSpace(raw) == "" || raw == "null" || raw == "[]" {
		return true
	}
	var rules BranchRules
	if json.Unmarshal([]byte(raw), &rules) != nil {
		return false
	}
	for _, pattern := range rules.Excludes {
		if branchPatternMatch(pattern, branch) {
			return false
		}
	}
	if len(rules.Includes) == 0 {
		return true
	}
	for _, pattern := range rules.Includes {
		if branchPatternMatch(pattern, branch) {
			return true
		}
	}
	return false
}

func branchPatternMatch(pattern, branch string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	matched, err := path.Match(pattern, branch)
	return err == nil && matched
}

// TriggerFromWebhook queues a build after the webhook handler has authenticated
// the provider secret and matched the source to this service.
func (l *Logic) TriggerFromWebhook(ctx context.Context, req *WebhookBuildRequest) (int64, error) {
	if req == nil || req.ServiceID <= 0 || req.SourceID <= 0 {
		return 0, errors.New("invalid webhook build request")
	}
	svc, err := l.svcRepo.FindByID(ctx, req.ServiceID)
	if err != nil {
		return 0, err
	}
	if svc == nil || svc.CodeSourceID != req.SourceID {
		return 0, errors.New("webhook source is not bound to service")
	}
	id, code := l.trigger(ctx, &TriggerBuildRequest{ServiceID: req.ServiceID, Branch: req.Branch, CommitID: req.CommitID, TriggerType: "webhook", TriggerKey: "webhook:" + fmt.Sprint(req.SourceID) + ":" + strings.TrimSpace(req.DeliveryID), Author: req.Author, AuthorEmail: req.AuthorEmail, Message: req.Message}, 0, true)
	if code == errcode.CodeSuccess {
		return id, nil
	}
	return 0, fmt.Errorf("trigger webhook build: %s", errcode.Message(code))
}

// Retry 重新构建
func (l *Logic) Retry(ctx context.Context, id int64, userID int64) (int64, int) {
	run, err := l.repo.FindRunByID(ctx, id)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if run == nil {
		return 0, errcode.CodeBuildConfigNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "run", ProjectID: run.ProjectID, ServiceID: run.ServiceID, BuildID: run.ID})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if !allowed {
		return 0, errcode.CodeNoPermission
	}

	// 用原构建的参数重新触发
	req := &TriggerBuildRequest{
		ServiceID:   run.ServiceID,
		Branch:      run.GitBranch,
		CommitID:    run.GitCommit,
		TriggerType: "manual",
	}
	return l.Trigger(ctx, req, userID)
}

// Cancel 取消构建
func (l *Logic) Cancel(ctx context.Context, userID, id int64) int {
	run, err := l.repo.FindRunByID(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if run == nil {
		return errcode.CodeBuildConfigNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "cancel", ProjectID: run.ProjectID, ServiceID: run.ServiceID, BuildID: run.ID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	if err := l.repo.RequestCancel(ctx, id); err != nil {
		if errors.Is(err, ErrInvalidTransition) {
			return errcode.CodeParamInvalid
		}
		return errcode.CodeDBError
	}

	return errcode.CodeSuccess
}

// GetBuildConfig 获取构建配置
func (l *Logic) GetBuildConfig(ctx context.Context, userID, serviceID int64) (*BuildConfig, int) {
	svc, err := l.svcRepo.FindByID(ctx, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if svc == nil {
		return nil, errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_config", Action: "view", ProjectID: svc.ProjectID, ServiceID: svc.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	config, err := l.repo.FindConfigByServiceID(ctx, serviceID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if config == nil {
		return nil, errcode.CodeBuildConfigNotFound
	}
	return config, errcode.CodeSuccess
}

// SaveBuildConfig 创建或更新构建配置
func (l *Logic) SaveBuildConfig(ctx context.Context, req *CreateBuildConfigRequest, userID int64) int {
	if req == nil || req.ServiceID <= 0 {
		return errcode.CodeParamInvalid
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds != 0 && (*req.TimeoutSeconds < 60 || *req.TimeoutSeconds > 86400) {
		return errcode.CodeParamInvalid
	}
	if _, err := buildResources(req.ResourceCPU, req.ResourceMemory); err != nil {
		return errcode.CodeParamInvalid
	}
	// 检查服务存在
	svc, err := l.svcRepo.FindByID(ctx, req.ServiceID)
	if err != nil {
		return errcode.CodeDBError
	}
	if svc == nil {
		return errcode.CodeServiceNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_config", Action: "update", ProjectID: svc.ProjectID, ServiceID: svc.ID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}

	// 处理 JSON 字段
	if binding := req.ServiceBinding; binding != nil {
		allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "update", ProjectID: svc.ProjectID, ServiceID: svc.ID})
		if err != nil {
			return errcode.CodeDBError
		}
		if !allowed {
			return errcode.CodeNoPermission
		}
		if binding.CodeSourceID != nil {
			if *binding.CodeSourceID < 0 {
				return errcode.CodeParamInvalid
			}
			if *binding.CodeSourceID > 0 {
				allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "code_source", Action: "view", ProjectID: svc.ProjectID})
				if err != nil {
					return errcode.CodeDBError
				}
				if !allowed {
					return errcode.CodeNoPermission
				}
				allowed, err = l.svcRepo.CodeSourceBelongsToProject(ctx, *binding.CodeSourceID, svc.ProjectID)
				if err != nil {
					return errcode.CodeDBError
				}
				if !allowed {
					return errcode.CodeNoPermission
				}
			}
		}
		if binding.RegistryID != nil {
			if *binding.RegistryID < 0 || (*binding.RegistryID > 0 && strings.TrimSpace(req.ImageRepo) == "") {
				return errcode.CodeParamInvalid
			}
			if *binding.RegistryID > 0 {
				allowed, err = l.svcRepo.RegistryBelongsToProject(ctx, *binding.RegistryID, svc.ProjectID)
				if err != nil {
					return errcode.CodeDBError
				}
				if !allowed {
					return errcode.CodeNoPermission
				}
			}
		}
	}
	var branchRulesStr *string
	if req.BranchRules != nil {
		data, _ := json.Marshal(req.BranchRules)
		s := string(data)
		branchRulesStr = &s
	}
	var envVarsStr *string
	if req.EnvVars != nil {
		data, _ := json.Marshal(req.EnvVars)
		s := string(data)
		envVarsStr = &s
	}

	config := &BuildConfig{
		DockerfileSource:  strings.TrimSpace(req.DockerfileSource),
		DockerfileContent: req.DockerfileContent,
		ServiceID:         req.ServiceID,
		TemplateType:      req.TemplateType,
		BuildMode:         normalizeBuildMode(req.BuildMode),
		BuildImage:        req.BuildImage,
		BuildCommand:      req.BuildCommand,
		DockerfilePath:    strings.TrimSpace(req.DockerfilePath),
		BuildContext:      strings.TrimSpace(req.BuildContext),
		ImageRepo:         req.ImageRepo,
		CacheDir:          req.CacheDir,
		CacheStrategy:     normalizeCacheStrategy(req.CacheStrategy),
		CachePVCName:      strings.TrimSpace(req.CachePVCName),
		BranchRules:       branchRulesStr,
		EnvVars:           envVarsStr,
		ResourceCPU:       req.ResourceCPU,
		ResourceMemory:    req.ResourceMemory,
		Provider:          req.Provider,
		JenkinsJobName:    req.JenkinsJobName,
		CreatedBy:         userID,
	}
	if config.DockerfilePath == "" {
		config.DockerfilePath = strings.TrimSpace(svc.DockerfilePath)
	}
	if config.BuildContext == "" {
		config.BuildContext = strings.TrimSpace(svc.BuildContext)
	}
	if config.ImageRepo == "" {
		config.ImageRepo, err = l.svcRepo.ResolveImageRepository(ctx, req.ServiceID)
		if err != nil {
			return errcode.CodeDBError
		}
	}

	if req.CacheEnabled != nil {
		config.CacheEnabled = *req.CacheEnabled
	} else {
		config.CacheEnabled = true
	}
	if req.TimeoutSeconds != nil {
		config.TimeoutSeconds = *req.TimeoutSeconds
	} else {
		config.TimeoutSeconds = 0
	}
	if req.HostAgentID != nil {
		config.HostAgentID = *req.HostAgentID
	}
	config.Provider = normalizeBuildProvider(config.Provider)
	if config.DockerfilePath == "" {
		config.DockerfilePath = "Dockerfile"
	}
	if config.BuildContext == "" {
		config.BuildContext = "."
	}

	if config.DockerfileSource == "" {
		config.DockerfileSource = "repository"
	}
	if validateDockerfileInput(config.DockerfileSource, config.DockerfileContent, config.BuildContext) != nil ||
		(config.DockerfileSource == "inline" && (config.BuildMode == "artifact" || config.Provider != "" && config.Provider != KubeJobProviderName)) {
		return errcode.CodeParamInvalid
	}
	if config.DockerfileSource == "repository" {
		config.DockerfileContent = ""
	}
	if err := l.repo.SaveConfigWithBinding(ctx, config, req.ServiceBinding); err != nil {
		return errcode.CodeDBError
	}
	return errcode.CodeSuccess
}

func normalizeBuildMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jar", "jar_image", "java":
		return "jar_image"
	case "artifact", "binary", "jar_artifact":
		return "artifact"
	case "dockerfile", "container", "":
		return "dockerfile"
	default:
		return "dockerfile"
	}
}

func normalizeCacheStrategy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inherit":
		return "inherit"
	case "pvc":
		return "pvc"
	case "registry", "registry_cache", "buildkit":
		return "registry"
	case "emptydir", "ephemeral", "":
		return "emptydir"
	default:
		return "emptydir"
	}
}

func normalizeBuildProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ecs_agent", "host_agent", "ecs":
		return "ecs_agent"
	case "kube_job", "kubernetes", "k8s", "":
		return "kube_job"
	default:
		return "kube_job"
	}
}

// ListTemplates 构建模板列表
func (l *Logic) ListTemplates(ctx context.Context, userID int64) ([]BuildTemplate, int) {
	scope, err := l.authorizer.Scopes(ctx, userID, "build_template", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !scope.HasAnyAllow() {
		return nil, errcode.CodeNoPermission
	}
	templates, err := l.repo.ListTemplates(ctx)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return templates, errcode.CodeSuccess
}

// GetLogs 获取构建日志
func (l *Logic) GetLogs(ctx context.Context, userID, buildRunID int64) ([]BuildLog, int) {
	run, err := l.repo.FindRunByID(ctx, buildRunID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if run == nil {
		return nil, errcode.CodeBuildConfigNotFound
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build_log", Action: "view", ProjectID: run.ProjectID, ServiceID: run.ServiceID, BuildID: run.ID})
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if !allowed {
		return nil, errcode.CodeNoPermission
	}
	logs, err := l.listLogs(ctx, buildRunID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	// Current state is available before a container starts and for historical
	// runs whose worker never persisted stdout. This is explicitly system data.
	text := fmt.Sprintf("[system] Build #%d status=%s", run.ID, run.Status)
	if run.JobName != "" {
		text += " Job=" + run.JobName
	}
	if run.PodName != "" {
		text += " Pod=" + run.PodName
	}
	if run.ErrorMessage != "" {
		text += "\n[system] " + run.ErrorMessage
	}
	if len(logs) == 0 {
		if IsTerminalStatus(run.Status) {
			text += "\n[system] 此构建未保存容器输出；请查看上述状态和原因。"
		} else {
			text += "\n[system] 尚未采集到容器输出，等待调度、容器启动或日志采集。"
		}
	}
	for _, entry := range logs {
		if entry.Truncated {
			text += "\n[system] 日志记录存在截断或降级标记，当前输出可能不完整；请查看各阶段提示。"
			break
		}
	}
	logs = append([]BuildLog{{BuildRunID: run.ID, Stage: "status", LogText: text, LineCount: countLogLines(text)}}, logs...)
	return logs, errcode.CodeSuccess
}

func (l *Logic) listLogs(ctx context.Context, buildRunID int64) ([]BuildLog, error) {
	logs, err := l.repo.ListLogsByRunID(ctx, buildRunID)
	if err != nil || l.archive == nil {
		return logs, err
	}
	for index := range logs {
		if !strings.HasPrefix(logs[index].LogURL, "s3://") {
			continue
		}
		archiveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		content, err := l.archive.Get(archiveCtx, logs[index].LogURL)
		cancel()
		if err != nil {
			logs[index].LogText = "[system] 归档存储暂时不可用，显示数据库保留的日志片段。\n" + logs[index].LogText
			logs[index].Truncated = true
			continue
		}
		logs[index].LogText = string(content)
		// Fetching the archived snapshot cannot restore bytes already omitted
		// by the bounded Kubernetes log reader.
	}
	return logs, nil
}

// UpdateCallback 构建引擎回调（状态更新）
func (l *Logic) UpdateCallback(ctx context.Context, id int64, status, errMsg string, imageURL, imageDigest string, imageSize int64) int {
	if status != "" {
		if err := l.repo.TransitionRunStatus(ctx, id, status, errMsg); err != nil {
			if errors.Is(err, ErrInvalidTransition) || errors.Is(err, ErrRunNotFound) {
				return errcode.CodeParamInvalid
			}
			return errcode.CodeDBError
		}
	}
	if imageURL != "" {
		if err := l.repo.UpdateRunArtifact(ctx, id, imageURL, imageDigest, imageSize); err != nil {
			return errcode.CodeDBError
		}
	}
	return errcode.CodeSuccess
}
