package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

type rollbackLogicRepo struct {
	logicRepository
	preview *RollbackPreview
	prior   *WorkflowRun
}

func (r *rollbackLogicRepo) ListRollbackVersions(context.Context, int64, int64, int64, string) ([]DeploymentVersion, error) {
	return []DeploymentVersion{r.preview.Target}, nil
}
func (r *rollbackLogicRepo) PreviewRollback(context.Context, int64, int64, int64) (*RollbackPreview, error) {
	return r.preview, nil
}
func (r *rollbackLogicRepo) FindRollbackRequest(context.Context, string) (*WorkflowRun, error) {
	return r.prior, nil
}

func rollbackLogicFixture(t *testing.T, provider string) (*Logic, *rollbackLogicRepo, RollbackRequest) {
	t.Helper()
	env := &EnvironmentTarget{ID: 7, ProjectID: 10, ClusterID: 2, Namespace: "test", Status: "active", BindingStatus: "validated"}
	node := Node{Key: "deploy", Type: provider, Config: mustJSON(map[string]interface{}{"service_id": int64(9)})}
	snapshot := &DeploymentSnapshot{ID: 30, WorkflowRunID: 20, ServiceID: 9, ServiceRevisionID: 12, ProjectID: 10, EnvironmentID: 7, ClusterID: 2, Namespace: "test", Provider: provider, Status: "applied"}
	repo := &rollbackLogicRepo{logicRepository: logicRepository{env: env, run: &WorkflowRun{ID: 21, ProjectID: 10, WorkflowID: 2, EnvironmentID: 7, Tasks: []TaskRun{{NodeType: provider, NodeSnapshot: mustJSON(node)}}}}, preview: &RollbackPreview{Token: strings.Repeat("a", 64), WorkflowID: 2, EnvironmentID: 7, ClusterID: 2, Namespace: "test", ServiceID: 9, ServiceName: "api", Current: DeploymentVersion{SnapshotID: 31}, Target: DeploymentVersion{SnapshotID: 30, RunNumber: 1}, environment: env, snapshot: snapshot, workflowVersion: 4}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "deployment", Action: "rollback", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 9}})}
	return logic, repo, RollbackRequest{OriginRunID: 21, ServiceID: 9, ExpectedToken: repo.preview.Token}
}

func TestRollbackCreatesAuditedSingleServiceRun(t *testing.T) {
	for _, provider := range []string{NodeYAMLDeploy, NodeHelmDeploy} {
		t.Run(provider, func(t *testing.T) {
			logic, repo, req := rollbackLogicFixture(t, provider)
			id, code, msg := logic.SubmitRollback(actorContext(7, "rollback-account"), 7, 30, "rollback-request-0001", req)
			if code != 0 || id != 21 || msg != "" || len(repo.tasks) != 1 || repo.run.WorkflowID != 2 || repo.run.DefinitionVersion != 4 {
				t.Fatalf("run=%+v tasks=%+v code=%d msg=%s", repo.run, repo.tasks, code, msg)
			}
			if !snapshotRunActor(repo.run) || repo.run.TriggerUsername != "rollback-account" {
				t.Fatal("account not frozen")
			}
			var n Node
			var c struct {
				ServiceID  int64 `json:"service_id"`
				SnapshotID int64 `json:"snapshot_id"`
			}
			_ = json.Unmarshal(repo.tasks[0].NodeSnapshot, &n)
			_ = json.Unmarshal(n.Config, &c)
			if c.ServiceID != 9 || c.SnapshotID != 30 || repo.requestKey != "rollback-request-0001" {
				t.Fatal("rollback changed service or source")
			}
		})
	}
}

func TestRollbackPermissionIncludesEnvironmentCodeAndIdentity(t *testing.T) {
	for _, denied := range []bool{false, true} {
		l, repo, _ := rollbackLogicFixture(t, NodeYAMLDeploy)
		repo.env.Code = "qa"
		policies := policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "deployment", Action: "rollback", Effect: "allow", ScopeType: authz.ScopeService, ScopeID: 9, Environment: "qa"}}
		if denied {
			policies = append(policies, authz.Policy{Resource: "deployment", Action: "rollback", Effect: "deny", ScopeType: authz.ScopeEnvironment, ScopeID: 7, Environment: "qa"})
		}
		l.authorizer = authz.New(policies)
		_, code, _ := l.PreviewRollback(context.Background(), 7, 21, 9, 30)
		want := errcode.CodeSuccess
		if denied {
			want = errcode.CodeNoPermission
		}
		if code != want {
			t.Fatalf("rollback environment ignored: code=%d want=%d", code, want)
		}
		repo.env.ProjectID = 99
		if _, code, _ := l.PreviewRollback(context.Background(), 7, 21, 9, 30); code == 0 {
			t.Fatal("cross project rollback accepted")
		}
	}
}

func TestRollbackKeepsProductionAndEnvironmentApprovalPolicy(t *testing.T) {
	logic, repo, req := rollbackLogicFixture(t, NodeYAMLDeploy)
	repo.preview.RequiresApproval = true
	_, code, _ := logic.SubmitRollback(actorContext(7, "release-account"), 7, 30, "rollback-production-1", req)
	if code != 0 || repo.run.Status != RunPendingApproval || len(repo.tasks) != 2 || repo.tasks[0].NodeType != NodeApproval || len(repo.tasks[1].DependsOn) != 1 {
		t.Fatalf("approval bypassed: %+v", repo.tasks)
	}
}

func TestRollbackRejectsWrongServicePermissionAndStalePreview(t *testing.T) {
	logic, repo, req := rollbackLogicFixture(t, NodeYAMLDeploy)
	req.ServiceID = 99
	if _, code, _ := logic.SubmitRollback(context.Background(), 7, 30, "rollback-invalid-1", req); code != errcode.CodeParamInvalid {
		t.Fatal("unselected service accepted")
	}
	req.ServiceID = 9
	logic.authorizer = authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})
	if _, code, _ := logic.SubmitRollback(context.Background(), 7, 30, "rollback-invalid-1", req); code != errcode.CodeNoPermission {
		t.Fatal("permission bypassed")
	}
	logic, repo, req = rollbackLogicFixture(t, NodeYAMLDeploy)
	req.ExpectedToken = strings.Repeat("b", 64)
	if _, code, _ := logic.SubmitRollback(context.Background(), 7, 30, "rollback-invalid-1", req); code == 0 || repo.requestKey != "" {
		t.Fatal("stale preview created run")
	}
	req.ExpectedToken = repo.preview.Token
	repo.preview.BlockedReason = "same version"
	if _, code, msg := logic.SubmitRollback(context.Background(), 7, 30, "rollback-invalid-1", req); code == 0 || msg != "same version" {
		t.Fatal("blocked preview submitted")
	}
}

func TestRollbackRetryReturnsOriginalRunAfterVersionChanged(t *testing.T) {
	logic, repo, req := rollbackLogicFixture(t, NodeYAMLDeploy)
	repo.prior = &WorkflowRun{ID: 99, TriggerUserID: 7, SourceSnapshotID: 30, ContextSnapshot: mustJSON(map[string]interface{}{"rollback_request": req})}
	repo.preview.Token = "changed"
	if id, code, _ := logic.SubmitRollback(context.Background(), 7, 30, "rollback-retry-0001", req); code != 0 || id != 99 {
		t.Fatal("idempotent retry did not return original")
	}
	if _, code, _ := logic.SubmitRollback(context.Background(), 8, 30, "rollback-retry-0001", req); code == 0 {
		t.Fatal("another account reused request")
	}
}
