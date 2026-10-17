package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type policyStore []authz.Policy

func (p policyStore) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) { return p, nil }

type logicRepository struct {
	profile    *DeliveryProfile
	item       *Workflow
	created    *Workflow
	updated    bool
	env        *EnvironmentTarget
	run        *WorkflowRun
	tasks      []TaskRun
	cancel     bool
	approved   bool
	snapshot   *DeploymentSnapshot
	release    *ReleaseTarget
	requestKey string
	listed     []Workflow
	runs       []WorkflowRun
	runTotal   int64
}

func (r *logicRepository) FindProjectDeliveryProfile(context.Context, int64) (*DeliveryProfile, error) {
	return r.profile, nil
}
func (r *logicRepository) List(context.Context, authz.ScopeSet, int64) ([]Workflow, error) {
	return r.listed, nil
}
func (r *logicRepository) FindByID(context.Context, int64) (*Workflow, error) { return r.item, nil }
func (r *logicRepository) Create(_ context.Context, item *Workflow) (int64, error) {
	r.created = item
	return 9, nil
}
func (r *logicRepository) Update(_ context.Context, item *Workflow, _ int64) (bool, error) {
	r.updated = true
	r.item = item
	return true, nil
}
func (r *logicRepository) FindEnvironmentTarget(context.Context, int64) (*EnvironmentTarget, error) {
	return r.env, nil
}
func (r *logicRepository) FindReleaseTarget(context.Context, int64, int64) (*ReleaseTarget, error) {
	return r.release, nil
}
func (r *logicRepository) FindAppliedDeploymentSnapshot(context.Context, int64) (*DeploymentSnapshot, error) {
	return r.snapshot, nil
}
func (r *logicRepository) CreateRun(_ context.Context, run *WorkflowRun, tasks []TaskRun) (int64, error) {
	r.run = run
	r.tasks = tasks
	r.run.ID = 21
	return 21, nil
}
func (r *logicRepository) CreateRollbackRun(_ context.Context, run *WorkflowRun, tasks []TaskRun, _ int64, requestKey string) (int64, error) {
	r.run = run
	r.tasks = tasks
	r.requestKey = requestKey
	r.run.ID = 21
	return 21, nil
}
func (r *logicRepository) FindRun(context.Context, int64) (*WorkflowRun, error) { return r.run, nil }
func (r *logicRepository) ListRuns(context.Context, authz.ScopeSet, int64, int64, int, int) ([]WorkflowRun, int64, error) {
	return r.runs, r.runTotal, nil
}
func (r *logicRepository) RequestCancel(context.Context, int64) (bool, error) {
	r.cancel = true
	return true, nil
}
func (r *logicRepository) ApproveRun(context.Context, int64) (bool, error) {
	r.approved = true
	return true, nil
}

func TestListUsesWorkflowViewScope(t *testing.T) {
	repo := &logicRepository{listed: []Workflow{{ID: 9, ProjectID: 10, Name: "Deploy", NodeCount: 2}}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}})}
	items, code := logic.List(context.Background(), 7, 10)
	if code != errcode.CodeSuccess || len(items) != 1 || items[0].NodeCount != 2 {
		t.Fatalf("items=%+v code=%d", items, code)
	}
}

func TestCancelHonorsEnvironmentDenial(t *testing.T) {
	repo := &logicRepository{run: &WorkflowRun{ID: 1, ProjectID: 10, WorkflowID: 2, EnvironmentID: 7}, env: &EnvironmentTarget{ID: 7, ProjectID: 10, Code: "prod"}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "cancel", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "cancel", Effect: "deny", ScopeType: authz.ScopeEnvironment, ScopeID: 7},
	})}
	if code := logic.Cancel(context.Background(), 9, 1); code != errcode.CodeNoPermission || repo.cancel {
		t.Fatalf("environment denial bypassed: %d", code)
	}
}

func TestCreateRejectsDeliveryTypeMismatch(t *testing.T) {
	repo := &logicRepository{profile: &DeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "workflow", Action: "create", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}})}
	_, code := logic.Create(context.Background(), 7, 10, &CreateWorkflowRequest{
		Name: "Deploy", Code: "deploy", Definition: Definition{DeliveryType: DeliveryTypeHelm, Nodes: []Node{{Key: "build", Name: "Build", Type: NodeBuild}}},
	})
	if code != errcode.CodeParamInvalid || repo.created != nil {
		t.Fatalf("code=%d created=%+v", code, repo.created)
	}
}

func TestUpdateUsesDefinitionVersionAndProjectPermission(t *testing.T) {
	repo := &logicRepository{
		profile: &DeliveryProfile{DeliveryType: DeliveryTypeK8sYAML, DeliveryStatus: "active"},
		item:    &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, Name: "Deploy", Code: "deploy", DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 2, Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "update", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	updated, code := logic.Update(context.Background(), 7, 9, &UpdateWorkflowRequest{
		ExpectedVersion: 2,
		Name:            "Deploy test",
		Description:     "validated",
		Definition:      Definition{DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "build", Name: "Build", Type: NodeBuild}}},
	})
	if code != errcode.CodeSuccess || updated == nil || !repo.updated {
		t.Fatalf("updated=%+v code=%d updated=%v", updated, code, repo.updated)
	}
}

func TestRunCreatesImmutableSnapshotForNonProduction(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 4, Status: "active", Nodes: []Node{
			{Key: "build", Name: "Build", Type: NodeBuild},
			{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}},
		}},
		env: &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	id, code := logic.Run(actorContext(7, "release-account"), 7, 9, &RunRequest{EnvironmentID: 7})
	if code != errcode.CodeSuccess || id != 21 || repo.run == nil || repo.run.DefinitionVersion != 4 || len(repo.tasks) != 2 {
		t.Fatalf("id=%d code=%d run=%+v tasks=%d", id, code, repo.run, len(repo.tasks))
	}
	if !snapshotRunActor(repo.run) || repo.run.TriggerUsername != "release-account" {
		t.Fatal("manual execution did not freeze its authenticated account")
	}
}

func TestRunRejectsEnvironmentDifferentFromWorkflowBinding(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{{Key: "build", Type: NodeBuild}}},
		env:  &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})}
	if _, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 8}); code != errcode.CodeParamInvalid || repo.run != nil {
		t.Fatalf("code=%d run=%+v", code, repo.run)
	}
}

func TestBuildOnlyWorkflowRunsWithoutEnvironment(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 11, ProjectID: 10, TargetEnvironmentID: 0, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{{Key: "build", Name: "Build", Type: NodeBuild, Config: mustJSON(BuildNodeConfig{ServiceID: 3})}}},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	id, code := logic.Run(context.Background(), 7, 11, &RunRequest{})
	if code != errcode.CodeSuccess || id != 21 || repo.run == nil || repo.run.EnvironmentID != 0 {
		t.Fatalf("id=%d code=%d run=%+v", id, code, repo.run)
	}
}

func TestRunStoresDeploymentStrategyInContextSnapshot(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 4, Status: "active", Nodes: []Node{
			{Key: "build", Name: "Build", Type: NodeBuild},
			{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}},
		}},
		env: &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7, DeploymentStrategy: DeploymentStrategyRolling})
	if code != errcode.CodeSuccess || repo.run == nil {
		t.Fatalf("code=%d run=%+v", code, repo.run)
	}
	var snapshot map[string]interface{}
	if err := json.Unmarshal(repo.run.ContextSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot["deployment_strategy"] != DeploymentStrategyRolling {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestRunAcceptsInitializationTrigger(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy}}},
		env:  &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	id, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7, TriggerType: TriggerInitialization})
	if code != errcode.CodeSuccess || id != 21 || repo.run == nil || repo.run.TriggerType != TriggerInitialization {
		t.Fatalf("id=%d code=%d run=%+v", id, code, repo.run)
	}
}

func TestRunRejectsUnknownDeploymentStrategy(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{{Key: "build", Name: "Build", Type: NodeBuild}}},
		env:  &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	for _, strategy := range []string{"unsafe", DeploymentStrategyCanary, DeploymentStrategyBlueGreen} {
		_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7, DeploymentStrategy: strategy})
		if code != errcode.CodeParamInvalid || repo.run != nil {
			t.Fatalf("strategy=%s code=%d run=%+v", strategy, code, repo.run)
		}
	}
}

func TestRunAppliesReleaseOverrideOnlyToRunSnapshot(t *testing.T) {
	originalConfig := []byte(`{"service_id":8,"service_revision_id":11}`)
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 4, Status: "active", Nodes: []Node{
			{Key: "build", Name: "Build", Type: NodeBuild, Config: []byte(`{"service_id":8,"branch":"main"}`)},
			{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, Config: originalConfig, DependsOn: []string{"build"}},
		}},
		env:     &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
		release: &ReleaseTarget{ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 12},
	})}
	_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7, ServiceID: 12, ServiceRevisionID: 13})
	if code != errcode.CodeSuccess || len(repo.tasks) != 2 {
		t.Fatalf("code=%d tasks=%+v", code, repo.tasks)
	}
	var snapshot Node
	if err := json.Unmarshal(repo.tasks[1].NodeSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	var config YAMLDeployConfig
	if err := json.Unmarshal(snapshot.Config, &config); err != nil || config.ServiceID != 12 || config.ServiceRevisionID != 13 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
	if string(repo.item.Nodes[1].Config) != string(originalConfig) {
		t.Fatal("persisted workflow config was mutated")
	}
}

func TestRunSelectsOnlyOneIndependentWorkflowService(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 4, Status: "active", Nodes: []Node{
			{Key: "build-a", Type: NodeBuild, Config: []byte(`{"service_id":8,"branch":"main"}`)},
			{Key: "deploy-a", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":8,"service_revision_id":11}`), DependsOn: []string{"build-a"}},
			{Key: "build-b", Type: NodeBuild, Config: []byte(`{"service_id":12,"branch":"main"}`)},
			{Key: "deploy-b", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":12,"service_revision_id":13}`), DependsOn: []string{"build-b"}},
		}},
		env:     &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
		release: &ReleaseTarget{ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 12},
	})}
	_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7, ServiceID: 12, ServiceRevisionID: 15, SelectWorkflowService: true, Branch: "test"})
	if code != errcode.CodeSuccess || len(repo.tasks) != 2 || repo.tasks[0].NodeKey != "build-b" || repo.tasks[1].NodeKey != "deploy-b" || repo.tasks[1].DependsOn[0] != "build-b" {
		t.Fatalf("code=%d tasks=%+v", code, repo.tasks)
	}
	var build, deploy Node
	if err := json.Unmarshal(repo.tasks[0].NodeSnapshot, &build); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(repo.tasks[1].NodeSnapshot, &deploy); err != nil {
		t.Fatal(err)
	}
	if string(build.Config) != `{"branch":"test","service_id":12}` || string(deploy.Config) != `{"service_id":12,"service_revision_id":15}` {
		t.Fatalf("build=%s deploy=%s", build.Config, deploy.Config)
	}
	if len(repo.item.Nodes) != 4 {
		t.Fatal("saved workflow changed")
	}
}

func TestRunRejectsCrossServiceDependency(t *testing.T) {
	nodes := []Node{
		{Key: "build-a", Type: NodeBuild, Config: []byte(`{"service_id":8}`)},
		{Key: "deploy-a", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":8,"service_revision_id":11}`), DependsOn: []string{"build-a"}},
		{Key: "deploy-b", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":12,"service_revision_id":13}`), DependsOn: []string{"build-a"}},
	}
	if _, err := selectWorkflowServiceNodes(nodes, 12); err == nil {
		t.Fatal("cross-service dependency was accepted")
	}
	nodes[2].DependsOn = nil
	nodes[2].Type = NodeApproval
	if _, err := selectWorkflowServiceNodes(nodes, 12); err == nil {
		t.Fatal("unscoped approval node was accepted")
	}
	nodes[2].Type = NodeBuild
	if _, err := selectWorkflowServiceNodes(nodes, 8); err == nil {
		t.Fatal("service without a deploy node was accepted")
	}
}

func TestOrderedWorkflowStillAllowsOneServiceRun(t *testing.T) {
	nodes := []Node{
		{Key: "build-server", Type: NodeBuild, Config: []byte(`{"service_id":47}`)},
		{Key: "deploy-server", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":47}`), DependsOn: []string{"build-server"}},
		{Key: "build-gateway", Type: NodeBuild, Config: []byte(`{"service_id":69}`)},
		{Key: "deploy-gateway", Type: NodeYAMLDeploy, Config: []byte(`{"service_id":69}`), DependsOn: []string{"build-gateway", "deploy-server"}},
	}
	selected, err := selectWorkflowServiceNodes(nodes, 69)
	if err != nil || len(selected) != 2 || len(selected[1].DependsOn) != 1 || selected[1].DependsOn[0] != "build-gateway" {
		t.Fatalf("selected=%+v err=%v", selected, err)
	}
	if len(nodes[3].DependsOn) != 2 {
		t.Fatal("saved workflow dependencies changed")
	}
}

func TestRunHonorsEnvironmentApprovalFlag(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{
			{Key: "approval", Name: "Approval", Type: NodeApproval},
			{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy, DependsOn: []string{"approval"}},
		}},
		env: &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", DeployApproval: true, Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if code != errcode.CodeSuccess || repo.run.Status != RunPendingApproval {
		t.Fatalf("code=%d run=%+v", code, repo.run)
	}
}

func TestProductionRunRequiresApprovalNode(t *testing.T) {
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{{Key: "deploy", Name: "Deploy", Type: NodeYAMLDeploy}}},
		env:  &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", IsProduction: true, Status: "active"},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	_, code := logic.Run(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if code != errcode.CodeParamInvalid || repo.run != nil {
		t.Fatalf("code=%d run=%+v", code, repo.run)
	}
}

func TestApproveMovesPendingProductionRunForward(t *testing.T) {
	repo := &logicRepository{env: &EnvironmentTarget{ID: 7, ProjectID: 10, Status: "active", BindingStatus: "validated"}, run: &WorkflowRun{ID: 21, ProjectID: 10, EnvironmentID: 7, Status: RunPendingApproval}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{
		Resource: "workflow", Action: "approve", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10,
	}})}
	code := logic.Approve(context.Background(), 7, 21)
	if code != errcode.CodeSuccess || !repo.approved {
		t.Fatalf("code=%d approved=%v", code, repo.approved)
	}
}

func TestListRunsUsesWorkflowViewScope(t *testing.T) {
	repo := &logicRepository{
		item:     &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 2},
		runs:     []WorkflowRun{{ID: 21, WorkflowID: 9, ProjectID: 10, Status: RunSucceeded}},
		runTotal: 1,
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeWorkflow, ScopeID: 9},
	})}
	items, total, code := logic.ListRuns(context.Background(), 7, 9, 1, 20)
	if code != errcode.CodeSuccess || total != 1 || len(items) != 1 || items[0].ID != 21 {
		t.Fatalf("items=%+v total=%d code=%d", items, total, code)
	}
}
