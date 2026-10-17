package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/projectstate"
)

type repository interface {
	FindProjectDeliveryProfile(context.Context, int64) (*DeliveryProfile, error)
	List(context.Context, authz.ScopeSet, int64) ([]Workflow, error)
	FindByID(context.Context, int64) (*Workflow, error)
	Create(context.Context, *Workflow) (int64, error)
	Update(context.Context, *Workflow, int64) (bool, error)
}

type runRepository interface {
	repository
	FindEnvironmentTarget(context.Context, int64) (*EnvironmentTarget, error)
	FindReleaseTarget(context.Context, int64, int64) (*ReleaseTarget, error)
	FindAppliedDeploymentSnapshot(context.Context, int64) (*DeploymentSnapshot, error)
	CreateRun(context.Context, *WorkflowRun, []TaskRun) (int64, error)
	CreateRollbackRun(context.Context, *WorkflowRun, []TaskRun, int64, string) (int64, error)
	FindRun(context.Context, int64) (*WorkflowRun, error)
	RequestCancel(context.Context, int64) (bool, error)
	ApproveRun(context.Context, int64) (bool, error)
}

type runListRepository interface {
	runRepository
	ListRuns(context.Context, authz.ScopeSet, int64, int64, int, int) ([]WorkflowRun, int64, error)
}

// Each rollback submission retains its request identity across uncertain retries.
var rollbackRequestKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

func mustJSON(value interface{}) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

type EnvironmentTarget struct {
	Code            string
	ClusterID       int64
	Namespace       string
	HostGroupRef    string
	ID              int64
	ProjectID       int64
	EnvironmentType string
	BindingStatus   string
	IsProduction    bool
	DeployApproval  bool
	Status          string
}

type ReleaseTarget struct {
	ProjectID    int64
	DeliveryType string
}

type Logic struct {
	repo           repository
	authorizer     authz.Checker
	yamlResolver   YAMLRenderResolver
	helmResolver   *DefaultHelmResolver
	restartFactory HelmRESTConfigFactory
}

func NewLogic(db *sql.DB, authorizer authz.Checker) *Logic {
	return &Logic{repo: NewRepository(db), authorizer: authorizer, yamlResolver: NewDefaultYAMLResolver(NewSQLYAMLSourceStore(db)), helmResolver: NewDefaultHelmResolver(NewSQLHelmSourceStore(db), nil)}
}

func (l *Logic) WithHelmDecryptor(db *sql.DB, d CredentialDecryptor) *Logic {
	l.helmResolver = NewDefaultHelmResolver(NewSQLHelmSourceStore(db), d)
	return l
}

func (l *Logic) List(ctx context.Context, userID, projectID int64) ([]Workflow, int) {
	if projectID <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	scope, err := l.authorizer.Scopes(ctx, userID, "workflow", "view")
	if err != nil {
		return nil, errcode.CodeDBError
	}
	items, err := l.repo.List(ctx, scope, projectID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	return items, errcode.CodeSuccess
}

func (l *Logic) Create(ctx context.Context, userID, projectID int64, req *CreateWorkflowRequest) (int64, int) {
	if req == nil {
		return 0, errcode.CodeParamInvalid
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if !ValidateWorkflowIdentity(req.Name, req.Code) {
		return 0, errcode.CodeParamInvalid
	}
	if code := l.authorizeTarget(ctx, userID, projectID, 0, req.TargetEnvironmentID, "create"); code != errcode.CodeSuccess {
		return 0, code
	}
	profile, err := l.repo.FindProjectDeliveryProfile(ctx, projectID)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	if profile == nil {
		return 0, errcode.CodeProjectNotFound
	}
	if profile.DeliveryStatus != "active" || req.Definition.DeliveryType != profile.DeliveryType {
		return 0, errcode.CodeParamInvalid
	}
	if issues := ValidateDefinition(req.Definition); len(issues) > 0 {
		return 0, errcode.CodeParamInvalid
	}
	if req.TargetEnvironmentID > 0 {
		if code := l.validateWorkflowEnvironment(ctx, userID, projectID, req.Definition.DeliveryType, req.TargetEnvironmentID); code != 0 {
			return 0, code
		}
	} else if req.TargetEnvironmentID < 0 || (requiresEnvironment(req.Definition.Nodes) && req.TargetEnvironmentID == 0) {
		return 0, errcode.CodeParamInvalid
	}
	workflow := &Workflow{ProjectID: projectID, Name: req.Name, Code: req.Code, Description: strings.TrimSpace(req.Description), DeliveryType: profile.DeliveryType, DefinitionVersion: 1, Status: "active", Nodes: req.Definition.Nodes, CreatedBy: userID, UpdatedBy: userID}
	workflow.TargetEnvironmentID = req.TargetEnvironmentID
	id, err := l.repo.Create(ctx, workflow)
	if err != nil {
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

func (l *Logic) Get(ctx context.Context, userID, id int64) (*Workflow, int) {
	workflow, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if workflow == nil {
		return nil, errcode.CodeNotFound
	}
	if code := l.authorizeTarget(ctx, userID, workflow.ProjectID, workflow.ID, workflow.TargetEnvironmentID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	workflow.RequiresEnvironment = requiresEnvironment(workflow.Nodes)
	return workflow, errcode.CodeSuccess
}

func (l *Logic) Update(ctx context.Context, userID, id int64, req *UpdateWorkflowRequest) (*Workflow, int) {
	if req == nil || req.ExpectedVersion <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	workflow, err := l.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if workflow == nil {
		return nil, errcode.CodeNotFound
	}
	if code := l.authorizeTarget(ctx, userID, workflow.ProjectID, workflow.ID, workflow.TargetEnvironmentID, "update"); code != errcode.CodeSuccess {
		return nil, code
	}
	if req.Name != "" {
		workflow.Name = strings.TrimSpace(req.Name)
	}
	if !ValidateWorkflowIdentity(workflow.Name, workflow.Code) {
		return nil, errcode.CodeParamInvalid
	}
	if req.Definition.DeliveryType != workflow.DeliveryType {
		return nil, errcode.CodeParamInvalid
	}
	if issues := ValidateDefinition(req.Definition); len(issues) > 0 {
		return nil, errcode.CodeParamInvalid
	}
	workflow.Description = strings.TrimSpace(req.Description)
	if req.TargetEnvironmentID != nil {
		id := *req.TargetEnvironmentID
		// Once bound, a test workflow cannot be repurposed as a production one.
		if id <= 0 || (workflow.TargetEnvironmentID > 0 && workflow.TargetEnvironmentID != id) {
			return nil, errcode.CodeParamInvalid
		}
		if code := l.validateWorkflowEnvironment(ctx, userID, workflow.ProjectID, workflow.DeliveryType, id); code != 0 {
			return nil, code
		}
		if code := l.authorizeTarget(ctx, userID, workflow.ProjectID, workflow.ID, id, "update"); code != errcode.CodeSuccess {
			return nil, code
		}
		workflow.TargetEnvironmentID = id
	}
	if requiresEnvironment(req.Definition.Nodes) && workflow.TargetEnvironmentID <= 0 {
		return nil, errcode.CodeParamInvalid
	}
	workflow.Nodes = req.Definition.Nodes
	if ok, err := l.repo.Update(ctx, workflow, req.ExpectedVersion); err != nil {
		return nil, errcode.CodeDBError
	} else if !ok {
		return nil, errcode.CodeParamInvalid
	}
	return l.Get(ctx, userID, id)
}

func (l *Logic) Validate(ctx context.Context, userID, id int64, definition Definition) ([]ValidationIssue, int) {
	workflow, code := l.Get(ctx, userID, id)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	if definition.DeliveryType == "" {
		definition.DeliveryType = workflow.DeliveryType
	}
	return ValidateDefinition(definition), errcode.CodeSuccess
}

func (l *Logic) Run(ctx context.Context, userID, workflowID int64, req *RunRequest) (int64, int) {
	id, code, _ := l.RunWithValidation(ctx, userID, workflowID, req)
	return id, code
}

// RunWithValidation reports actionable deployment configuration errors before
// queuing expensive builds. Execute still renders and validates again at apply.
func (l *Logic) RunWithValidation(ctx context.Context, userID, workflowID int64, req *RunRequest) (int64, int, string) {
	var validationMessage string
	id, code := l.run(ctx, userID, workflowID, req, &validationMessage)
	return id, code, validationMessage
}

func (l *Logic) run(ctx context.Context, userID, workflowID int64, req *RunRequest, validationMessage *string) (int64, int) {
	if req == nil || req.EnvironmentID < 0 {
		return 0, errcode.CodeParamInvalid
	}
	mode := strings.TrimSpace(req.ExecutionMode)
	if mode == "" {
		mode = "build"
	}
	if mode != "build" && mode != "existing_image" && mode != "config_only" || mode != "existing_image" && req.SourceBuildID != 0 {
		*validationMessage = "请选择重新构建，或使用已有成功镜像"
		return 0, errcode.CodeParamInvalid
	}
	runner, ok := l.repo.(runRepository)
	if !ok {
		return 0, errcode.CodeDBError
	}
	workflow, code := l.Get(ctx, userID, workflowID)
	if code != errcode.CodeSuccess {
		return 0, code
	}
	needsEnvironment := requiresEnvironment(workflow.Nodes)
	if needsEnvironment && (workflow.TargetEnvironmentID <= 0 || workflow.TargetEnvironmentID != req.EnvironmentID) {
		*validationMessage = "工作流未绑定目标环境或请求环境不匹配；请在编排中绑定环境，不允许跨环境执行"
		return 0, errcode.CodeParamInvalid
	}
	if !needsEnvironment && req.EnvironmentID != workflow.TargetEnvironmentID {
		*validationMessage = "工作流已绑定环境，请使用绑定环境执行"
		return 0, errcode.CodeParamInvalid
	}
	// Environment-scoped policies are checked below after resolving the target.
	var allowed bool
	var err error
	nodes := workflow.Nodes
	if len(req.Services) > 0 {
		if len(req.Services) < 2 || len(req.Services) > 50 || req.ServiceID != 0 || req.ServiceRevisionID != 0 || req.SelectWorkflowService || req.Branch != "" || req.SourceBuildID != 0 || !req.UseEnvironmentConfig || mode == "config_only" {
			*validationMessage = "批量执行请指定至少两个服务、各自分支或镜像，使用目标环境配置"
			return 0, errcode.CodeParamInvalid
		}
		nodes, code, err = l.selectedRunNodes(ctx, userID, workflow, req.Services, mode)
		if err != nil {
			*validationMessage = err.Error()
			return 0, code
		}
	}
	if req.SelectWorkflowService {
		if req.ServiceID <= 0 || req.ServiceRevisionID <= 0 {
			return 0, errcode.CodeParamInvalid
		}
		nodes, err = selectWorkflowServiceNodes(nodes, req.ServiceID)
		if err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeParamInvalid
		}
	}
	for _, node := range nodes {
		if node.Type == NodeTest {
			issues := ValidateDefinition(Definition{DeliveryType: workflow.DeliveryType, Nodes: []Node{{Key: node.Key, Name: node.Name, Type: NodeTest, Config: node.Config}}})
			if len(issues) > 0 {
				*validationMessage = "测试节点配置不完整，请在编排中设置执行镜像和命令"
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	if req.ServiceID > 0 || req.ServiceRevisionID > 0 {
		if req.ServiceID <= 0 || req.ServiceRevisionID <= 0 {
			return 0, errcode.CodeParamInvalid
		}
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "service", Action: "view", ProjectID: workflow.ProjectID, ServiceID: req.ServiceID})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
		target, err := runner.FindReleaseTarget(ctx, req.ServiceID, req.ServiceRevisionID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if target == nil || target.ProjectID != workflow.ProjectID || target.DeliveryType != workflow.DeliveryType {
			return 0, errcode.CodeParamInvalid
		}
		nodes, err = applyReleaseOverride(nodes, req.ServiceID, req.ServiceRevisionID)
		if err != nil {
			return 0, errcode.CodeParamInvalid
		}
	}
	if branch := strings.TrimSpace(req.Branch); branch != "" {
		nodes, err = applyBranchOverride(nodes, branch)
		if err != nil {
			return 0, errcode.CodeParamInvalid
		}
	}
	var environment *EnvironmentTarget
	if req.EnvironmentID > 0 {
		environment, err = runner.FindEnvironmentTarget(ctx, req.EnvironmentID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if environment == nil {
			return 0, errcode.CodeEnvironmentNotFound
		}
		if environment.ProjectID != workflow.ProjectID || environment.Status != "active" || environment.BindingStatus != "validated" || !compatibleEnvironment(workflow.DeliveryType, environment.EnvironmentType) {
			return 0, errcode.CodeParamInvalid
		}
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "workflow", Action: "run", ProjectID: workflow.ProjectID, WorkflowID: workflow.ID, EnvironmentID: environment.ID, Environment: environment.Code, ClusterID: environment.ClusterID})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	} else {
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "workflow", Action: "run", ProjectID: workflow.ProjectID, WorkflowID: workflow.ID})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
	}
	if needsEnvironment && environment != nil && environment.IsProduction && (workflow.DeliveryType == DeliveryTypeK8sYAML || workflow.DeliveryType == DeliveryTypeHelm) && mode != "config_only" {
		if mode != "build" || req.ExecutionMode != "build" || !req.UseEnvironmentConfig {
			*validationMessage = "生产发布请选择已有镜像，或明确选择构建并部署并使用目标环境配置"
			return 0, errcode.CodeParamInvalid
		}
		if err = l.authorizeProductionBuild(ctx, userID, environment, workflow.ID); err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeNoPermission
		}
	}
	if needsEnvironment {
		if freezer, ok := l.repo.(interface {
			FreezeDeploymentConfigs(context.Context, int64, []Node, bool) ([]Node, error)
		}); ok {
			if environment == nil {
				return 0, errcode.CodeParamInvalid
			}
			nodes, err = freezer.FreezeDeploymentConfigs(ctx, environment.ID, nodes, req.ServiceRevisionID > 0 && !req.UseEnvironmentConfig)
			if err != nil {
				*validationMessage = "环境配置快照失败：" + err.Error()
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	var promoted *PromotionBuild
	if environment != nil && environment.IsProduction && mode == "build" {
		for i := range nodes {
			if nodes[i].Type == NodeYAMLDeploy {
				var cfg YAMLDeployConfig
				if json.Unmarshal(nodes[i].Config, &cfg) != nil || cfg.FrozenYAML == nil {
					*validationMessage = "生产构建缺少目标环境配置快照"
					return 0, errcode.CodeParamInvalid
				}
				copy := *cfg.FrozenYAML
				copy.ImageRef = ""
				copy.PreservedImages = nil
				cfg.FrozenYAML = &copy
				nodes[i].Config = mustJSON(cfg)
			}
		}
	}
	var configurationDeployment *CurrentDeployment
	if mode == "config_only" {
		if workflow.DeliveryType != DeliveryTypeK8sYAML || req.ServiceID <= 0 || req.SourceSnapshotID <= 0 || !req.UseEnvironmentConfig || req.Branch != "" || len(req.ExpectedConfigDigest) != 64 {
			*validationMessage = "仅配置发布需要固定当前环境、配置预览和上次成功部署"
			return 0, errcode.CodeParamInvalid
		}
		loader, ok := l.repo.(interface {
			LoadConfigurationDeployment(context.Context, int64, int64, int64) (*CurrentDeployment, error)
		})
		if !ok {
			return 0, errcode.CodeDBError
		}
		configurationDeployment, err = loader.LoadConfigurationDeployment(ctx, workflow.ProjectID, req.EnvironmentID, req.ServiceID)
		if err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeParamInvalid
		}
		if configurationDeployment == nil || configurationDeployment.SnapshotID != req.SourceSnapshotID {
			*validationMessage = "当前部署已变化或尚无可追溯镜像，请重新加载并预览"
			return 0, errcode.CodeParamInvalid
		}
		nodes, err = configurationOnlyNodes(nodes, req.ServiceID, configurationDeployment, req.ExpectedConfigDigest)
		if err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeParamInvalid
		}
	}
	if mode == "existing_image" && len(req.Services) == 0 {
		if (workflow.DeliveryType != DeliveryTypeK8sYAML && workflow.DeliveryType != DeliveryTypeHelm) || req.ServiceID <= 0 || req.SourceBuildID <= 0 || !req.UseEnvironmentConfig || strings.TrimSpace(req.Branch) != "" {
			*validationMessage = "复用镜像需要选择一个 K8s YAML/Helm 服务、成功构建和目标环境配置，不能同时指定构建分支"
			return 0, errcode.CodeParamInvalid
		}
		loader, ok := l.repo.(interface {
			LoadPromotionBuild(context.Context, int64) (*PromotionBuild, error)
		})
		if !ok {
			return 0, errcode.CodeDBError
		}
		promoted, err = loader.LoadPromotionBuild(ctx, req.SourceBuildID)
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if promoted == nil || promoted.ProjectID != workflow.ProjectID || promoted.ServiceID != req.ServiceID || promoted.Status != "success" || !validPromotionImage(promoted) {
			*validationMessage = "所选构建不属于当前项目和服务，或没有成功生成可追溯的镜像摘要"
			return 0, errcode.CodeParamInvalid
		}
		allowed, err = l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "build", Action: "view", ProjectID: promoted.ProjectID, ServiceID: promoted.ServiceID, BuildID: promoted.ID})
		if err != nil {
			return 0, errcode.CodeDBError
		}
		if !allowed {
			return 0, errcode.CodeNoPermission
		}
		nodes, err = promoteYAMLNodes(nodes, req.ServiceID, promoted)
		if err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeParamInvalid
		}
	}
	if mode == "existing_image" && len(req.Services) > 0 {
		nodes, err = l.promoteSelectedRun(ctx, userID, workflow, nodes, req.Services)
		if err != nil {
			*validationMessage = err.Error()
			return 0, errcode.CodeParamInvalid
		}
	}
	status := RunPending
	if environment != nil && (environment.IsProduction || environment.DeployApproval) {
		if !hasApprovalNode(nodes) {
			return 0, errcode.CodeParamInvalid
		}
		status = RunPendingApproval
	}
	// Approval is currently a run-level gate, including non-production workflows.
	if hasApprovalNode(nodes) {
		status = RunPendingApproval
	}
	triggerType := strings.TrimSpace(req.TriggerType)
	if triggerType == "" {
		triggerType = "manual"
	}
	if triggerType != "manual" && triggerType != "webhook" && triggerType != "schedule" && triggerType != "approval" && triggerType != TriggerInitialization {
		return 0, errcode.CodeParamInvalid
	}
	strategy := strings.TrimSpace(req.DeploymentStrategy)
	if strategy == "" {
		strategy = DeploymentStrategyRolling
	}
	// Only the deployment definition is executed today; do not acknowledge an
	// orchestration strategy that no executor actually implements.
	if strategy != DeploymentStrategyRolling {
		return 0, errcode.CodeParamInvalid
	}
	resolvedRevisionID := req.ServiceRevisionID
	if req.UseEnvironmentConfig && req.ServiceID > 0 {
		for _, node := range nodes {
			if node.Type != NodeYAMLDeploy && node.Type != NodeHelmDeploy {
				continue
			}
			var selected struct {
				ServiceID  int64 `json:"service_id"`
				RevisionID int64 `json:"service_revision_id"`
			}
			if json.Unmarshal(node.Config, &selected) == nil && selected.ServiceID == req.ServiceID {
				resolvedRevisionID = selected.RevisionID
				break
			}
		}
	}
	environmentID := int64(0)
	if environment != nil {
		environmentID = environment.ID
	}
	snapshot, err := json.Marshal(map[string]interface{}{
		"trigger_actor": authenticatedRunActor(ctx, userID),
		"workflow_id":   workflow.ID, "definition_version": workflow.DefinitionVersion,
		"environment_id": environmentID, "delivery_type": workflow.DeliveryType,
		"target":     environment,
		"service_id": req.ServiceID, "service_revision_id": resolvedRevisionID,
		"use_environment_config": req.UseEnvironmentConfig,
		"branch":                 strings.TrimSpace(req.Branch),
		"deployment_strategy":    strategy,
		"execution_mode":         mode, "source_build": promoted,
		"configuration_deployment": configurationDeployment,
		"expected_config_digest":   req.ExpectedConfigDigest,
		"services":                 req.Services,
	})
	if err != nil {
		return 0, errcode.CodeDBError
	}
	run := &WorkflowRun{WorkflowID: workflow.ID, ProjectID: workflow.ProjectID, EnvironmentID: environmentID, DefinitionVersion: workflow.DefinitionVersion, TriggerType: triggerType, TriggerUserID: userID, Status: status, ContextSnapshot: snapshot}
	tasks := make([]TaskRun, 0, len(nodes))
	for _, node := range nodes {
		nodeSnapshot, _ := json.Marshal(node)
		taskStatus := RunPending
		if status == RunPendingApproval && node.Type == NodeApproval {
			taskStatus = RunPendingApproval
		}
		tasks = append(tasks, TaskRun{NodeKey: node.Key, NodeType: node.Type, DependsOn: node.DependsOn, NodeSnapshot: nodeSnapshot, Status: taskStatus})
	}
	if environment != nil && environment.IsProduction && (workflow.DeliveryType == DeliveryTypeK8sYAML || workflow.DeliveryType == DeliveryTypeHelm) && mode == "build" {
		if repo, ok := l.repo.(*Repository); ok {
			if err = validateProductionBuildTasks(ctx, repo.db, run, tasks); err != nil {
				*validationMessage = err.Error()
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	if l.helmResolver != nil {
		for _, task := range tasks {
			if task.NodeType != NodeHelmDeploy {
				continue
			}
			cfg, e := helmDeployConfig(task.NodeSnapshot)
			if e == nil {
				var target HelmTarget
				if hasServiceBuildDependency(tasks, task, cfg.ServiceID) {
					target, e = l.helmResolver.RenderHelmConfig(ctx, *run, cfg)
				} else {
					target, e = l.helmResolver.RenderHelm(ctx, *run, cfg)
				}
				if e == nil {
					var manifest []byte
					manifest, e = renderHelmManifest(ctx, HelmRequest{Namespace: target.Namespace, ReleaseName: target.ReleaseName, Chart: target.Chart, Values: target.Values}, false)
					if e == nil {
						e = validateHelmBuiltImage(manifest, target.Namespace, target.ImageRef)
					}
				}
			}
			if e != nil {
				*validationMessage = fmt.Sprintf("Helm 部署配置预检失败（服务 %d，Revision %d）；请在服务配置中检查 Chart、values 和镜像映射", cfg.ServiceID, cfg.ServiceRevisionID)
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	if l.yamlResolver != nil {
		for _, task := range tasks {
			if task.NodeType != NodeYAMLDeploy {
				continue
			}
			cfg, validationErr := yamlDeployConfig(task.NodeSnapshot)
			if validationErr == nil {
				var target YAMLTarget
				// Only a build dependency for this service can supply a future image.
				if validator, ok := l.yamlResolver.(interface {
					RenderYAMLConfig(context.Context, WorkflowRun, YAMLDeployConfig) (YAMLTarget, error)
				}); ok && hasServiceBuildDependency(tasks, task, cfg.ServiceID) {
					target, validationErr = validator.RenderYAMLConfig(ctx, *run, cfg)
				} else {
					target, validationErr = l.yamlResolver.RenderYAML(ctx, *run, cfg)
				}
				if validationErr == nil {
					_, validationErr = ValidateYAMLManifest(target.Manifest, target.Namespace)
				}
			}
			if validationErr != nil {
				*validationMessage = fmt.Sprintf("部署配置预检失败（服务 %d，Revision %d）：%s", cfg.ServiceID, cfg.ServiceRevisionID, validationErr)
				return 0, errcode.CodeParamInvalid
			}
		}
	}
	id, err := runner.CreateRun(ctx, run, tasks)
	if err != nil {
		if errors.Is(err, projectstate.ErrDeleted) {
			return 0, errcode.CodeProjectDeleted
		}
		return 0, errcode.CodeDBError
	}
	return id, errcode.CodeSuccess
}

// Only independent service subgraphs can become separate workflow runs.
func selectWorkflowServiceNodes(nodes []Node, serviceID int64) ([]Node, error) {
	serviceTypes := map[string]bool{NodeBuild: true, NodeTest: true, NodeYAMLDeploy: true, NodeHelmDeploy: true, NodeHostDeploy: true}
	serviceByKey := make(map[string]int64, len(nodes))
	typeByKey := make(map[string]string, len(nodes))
	deployCount := make(map[int64]int)
	services := make(map[int64]bool)
	isDeploy := func(t string) bool { return t == NodeYAMLDeploy || t == NodeHelmDeploy || t == NodeHostDeploy }
	for _, node := range nodes {
		if !serviceTypes[node.Type] || node.Key == "" {
			return nil, fmt.Errorf("此工作流包含无法按服务拆分的节点，请按工作流配置执行")
		}
		var config struct {
			ServiceID int64 `json:"service_id"`
		}
		if json.Unmarshal(node.Config, &config) != nil || config.ServiceID <= 0 {
			return nil, fmt.Errorf("工作流节点缺少有效服务绑定")
		}
		if _, exists := serviceByKey[node.Key]; exists {
			return nil, fmt.Errorf("工作流节点名称重复")
		}
		serviceByKey[node.Key] = config.ServiceID
		typeByKey[node.Key] = node.Type
		services[config.ServiceID] = true
		if isDeploy(node.Type) {
			deployCount[config.ServiceID]++
		}
	}
	for _, node := range nodes {
		for _, dependency := range node.DependsOn {
			if serviceByKey[dependency] == serviceByKey[node.Key] {
				continue
			}
			if !isDeploy(node.Type) || !isDeploy(typeByKey[dependency]) {
				return nil, fmt.Errorf("工作流存在跨服务依赖，请按工作流配置执行")
			}
		}
	}
	if len(services) < 2 || deployCount[serviceID] != 1 {
		return nil, fmt.Errorf("所选服务没有独立的部署流程")
	}
	for id := range services {
		if deployCount[id] != 1 {
			return nil, fmt.Errorf("工作流中的服务包含多个部署节点，请按工作流配置执行")
		}
	}
	selected := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if serviceByKey[node.Key] == serviceID {
			deps := make([]string, 0, len(node.DependsOn))
			for _, dependency := range node.DependsOn {
				if serviceByKey[dependency] == serviceID {
					deps = append(deps, dependency)
				}
			}
			node.DependsOn = deps
			selected = append(selected, node)
		}
	}
	return selected, nil
}

// applyReleaseOverride creates a run-only snapshot; the persisted workflow
// definition remains unchanged and can be reused for the next promotion.
func applyReleaseOverride(nodes []Node, serviceID, revisionID int64) ([]Node, error) {
	result := append([]Node(nil), nodes...)
	deployNodes := 0
	for i := range result {
		if result[i].Type != NodeBuild && result[i].Type != NodeTest && result[i].Type != NodeYAMLDeploy && result[i].Type != NodeHelmDeploy && result[i].Type != NodeHostDeploy {
			continue
		}
		config := map[string]interface{}{}
		if len(result[i].Config) > 0 && json.Unmarshal(result[i].Config, &config) != nil {
			return nil, fmt.Errorf("invalid node config")
		}
		config["service_id"] = serviceID
		if result[i].Type == NodeYAMLDeploy || result[i].Type == NodeHelmDeploy || result[i].Type == NodeHostDeploy {
			config["service_revision_id"] = revisionID
			deployNodes++
		}
		encoded, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		result[i].Config = encoded
	}
	if deployNodes != 1 {
		return nil, fmt.Errorf("quick release requires exactly one deploy node")
	}
	return result, nil
}

// applyBranchOverride changes only the run snapshot. The saved workflow keeps
// its configured branch so a one-off execution cannot alter future runs.
func applyBranchOverride(nodes []Node, branch string) ([]Node, error) {
	result := append([]Node(nil), nodes...)
	for i := range result {
		if result[i].Type != NodeBuild && result[i].Type != NodeTest {
			continue
		}
		config := map[string]interface{}{}
		if len(result[i].Config) > 0 && json.Unmarshal(result[i].Config, &config) != nil {
			return nil, fmt.Errorf("invalid build node config")
		}
		config["branch"] = branch
		encoded, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}
		result[i].Config = encoded
	}
	return result, nil
}

func (l *Logic) GetRun(ctx context.Context, userID, runID int64) (*WorkflowRun, int) {
	runner, ok := l.repo.(runRepository)
	if !ok {
		return nil, errcode.CodeDBError
	}
	run, err := runner.FindRun(ctx, runID)
	if err != nil {
		return nil, errcode.CodeDBError
	}
	if run == nil {
		return nil, errcode.CodeNotFound
	}
	if code := l.authorizeTarget(ctx, userID, run.ProjectID, run.WorkflowID, run.EnvironmentID, "view"); code != errcode.CodeSuccess {
		return nil, code
	}
	actors := []WorkflowRun{*run}
	l.loadRunActors(ctx, actors)
	run.TriggerUsername = actors[0].TriggerUsername
	return run, errcode.CodeSuccess
}

// ListRuns returns recent immutable executions for one workflow.
func (l *Logic) ListRuns(ctx context.Context, userID, workflowID int64, page, pageSize int) ([]WorkflowRun, int64, int) {
	if workflowID <= 0 {
		return nil, 0, errcode.CodeParamInvalid
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	workflow, code := l.Get(ctx, userID, workflowID)
	if code != errcode.CodeSuccess {
		return nil, 0, code
	}
	runner, ok := l.repo.(runListRepository)
	if !ok {
		return nil, 0, errcode.CodeDBError
	}
	scope, err := l.authorizer.Scopes(ctx, userID, "workflow", "view")
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	items, total, err := runner.ListRuns(ctx, scope, workflow.ProjectID, workflow.ID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, errcode.CodeDBError
	}
	l.loadRunActors(ctx, items)
	if loader, ok := l.repo.(interface {
		LoadBuildCommits(context.Context, []WorkflowRun, authz.ScopeSet) error
	}); ok {
		buildScope, e := l.authorizer.Scopes(ctx, userID, "build", "view")
		if e != nil {
			return nil, 0, errcode.CodeDBError
		}
		if e = loader.LoadBuildCommits(ctx, items, buildScope); e != nil {
			return nil, 0, errcode.CodeDBError
		}
	}
	return items, total, errcode.CodeSuccess
}

func (l *Logic) Cancel(ctx context.Context, userID, runID int64) int {
	runner, ok := l.repo.(runRepository)
	if !ok {
		return errcode.CodeDBError
	}
	run, err := runner.FindRun(ctx, runID)
	if err != nil {
		return errcode.CodeDBError
	}
	if run == nil {
		return errcode.CodeNotFound
	}
	permission := authz.Request{Resource: "workflow", Action: "cancel", ProjectID: run.ProjectID, WorkflowID: run.WorkflowID, EnvironmentID: run.EnvironmentID}
	if run.EnvironmentID > 0 {
		env, err := runner.FindEnvironmentTarget(ctx, run.EnvironmentID)
		if err != nil {
			return errcode.CodeDBError
		}
		if env == nil || env.ID != run.EnvironmentID || env.ProjectID != run.ProjectID {
			return errcode.CodeParamInvalid
		}
		permission.Environment = env.Code
		permission.ClusterID = env.ClusterID
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, permission)
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	ok, err = runner.RequestCancel(ctx, runID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !ok {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeSuccess
}

func (l *Logic) Approve(ctx context.Context, userID, runID int64) int {
	runner, ok := l.repo.(runRepository)
	if !ok {
		return errcode.CodeDBError
	}
	run, err := runner.FindRun(ctx, runID)
	if err != nil {
		return errcode.CodeDBError
	}
	if run == nil {
		return errcode.CodeNotFound
	}
	env := &EnvironmentTarget{}
	if run.EnvironmentID > 0 {
		env, err = runner.FindEnvironmentTarget(ctx, run.EnvironmentID)
		if err != nil {
			return errcode.CodeDBError
		}
		if env == nil || env.ProjectID != run.ProjectID || env.Status != "active" || env.BindingStatus != "validated" {
			return errcode.CodeParamInvalid
		}
		var submitted struct {
			Target *EnvironmentTarget `json:"target"`
		}
		if len(run.ContextSnapshot) > 0 && json.Unmarshal(run.ContextSnapshot, &submitted) != nil {
			return errcode.CodeParamInvalid
		}
		if submitted.Target != nil && *submitted.Target != *env {
			return errcode.CodeParamInvalid
		}
	}
	allowed, err := l.authorizer.Authorize(ctx, userID, authz.Request{Resource: "workflow", Action: "approve", ProjectID: run.ProjectID, WorkflowID: run.WorkflowID, EnvironmentID: env.ID, Environment: env.Code, ClusterID: env.ClusterID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	if run.Status != RunPendingApproval {
		return errcode.CodeParamInvalid
	}
	ok, err = runner.ApproveRun(ctx, runID)
	if err != nil {
		return errcode.CodeDBError
	}
	if !ok {
		return errcode.CodeParamInvalid
	}
	return errcode.CodeSuccess
}

func compatibleEnvironment(deliveryType, environmentType string) bool {
	return (deliveryType == DeliveryTypeK8sYAML || deliveryType == DeliveryTypeHelm) && environmentType == "k8s_namespace" || deliveryType == DeliveryTypeHost && environmentType == "host_group"
}

// requiresEnvironment keeps build/test/notification workflows reusable across
// projects and environments. A deployment node is the point at which a
// concrete namespace or host group must be selected and frozen.
func requiresEnvironment(nodes []Node) bool {
	for _, node := range nodes {
		if node.Type == NodeYAMLDeploy || node.Type == NodeHelmDeploy || node.Type == NodeHostDeploy {
			return true
		}
	}
	return false
}

func (l *Logic) validateWorkflowEnvironment(ctx context.Context, uid, projectID int64, deliveryType string, id int64) int {
	runner, ok := l.repo.(runRepository)
	if !ok {
		return errcode.CodeDBError
	}
	env, err := runner.FindEnvironmentTarget(ctx, id)
	if err != nil {
		return errcode.CodeDBError
	}
	if env == nil || env.ProjectID != projectID || env.Status != "active" || env.BindingStatus != "validated" || !compatibleEnvironment(deliveryType, env.EnvironmentType) {
		return errcode.CodeParamInvalid
	}
	allowed, err := l.authorizer.Authorize(ctx, uid, authz.Request{Resource: "environment", Action: "view", ProjectID: projectID, EnvironmentID: id, Environment: env.Code, ClusterID: env.ClusterID})
	if err != nil {
		return errcode.CodeDBError
	}
	if !allowed {
		return errcode.CodeNoPermission
	}
	return errcode.CodeSuccess
}

func hasApprovalNode(nodes []Node) bool {
	for _, node := range nodes {
		if node.Type == NodeApproval {
			return true
		}
	}
	return false
}
