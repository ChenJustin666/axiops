package workflow

import (
	"context"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"testing"
)

func TestDeploymentRequiresExplicitEnvironmentBinding(t *testing.T) {
	repo := &logicRepository{item: &Workflow{ID: 9, ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "deploy", Name: "deploy", Type: NodeYAMLDeploy}}}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})}
	if _, code, message := logic.RunWithValidation(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7}); code != errcode.CodeParamInvalid || message == "" || repo.run != nil {
		t.Fatalf("code=%d message=%s", code, message)
	}
}

func TestApprovalNodeGatesBuildOnlyAndCanBeApproved(t *testing.T) {
	repo := &logicRepository{item: &Workflow{ID: 9, ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "approval", Name: "approval", Type: NodeApproval}, {Key: "build", Name: "build", Type: NodeBuild}}}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})}
	if _, code := logic.Run(context.Background(), 7, 9, &RunRequest{}); code != 0 || repo.run.Status != RunPendingApproval {
		t.Fatalf("code=%d run=%+v", code, repo.run)
	}
	if code := logic.Approve(context.Background(), 7, repo.run.ID); code != 0 || !repo.approved {
		t.Fatalf("approve=%d", code)
	}
}

type environmentTaskStore struct {
	fakeTaskStore
	target *EnvironmentTarget
}

func (s *environmentTaskStore) FindEnvironmentTarget(context.Context, int64) (*EnvironmentTarget, error) {
	return s.target, nil
}

func TestWorkerRejectsChangedDeploymentTargets(t *testing.T) {
	original := EnvironmentTarget{ID: 7, ProjectID: 10, Code: "qa", ClusterID: 2, Namespace: "risk-qa", HostGroupRef: "qa-hosts", Status: "active", BindingStatus: "validated"}
	for _, kind := range []string{NodeYAMLDeploy, NodeHelmDeploy, NodeHostDeploy, NodeYAMLRollback, NodeHelmRollback} {
		t.Run(kind, func(t *testing.T) {
			current := original
			store := &environmentTaskStore{target: &current}
			worker := NewWorker(store, nil, 0, 0)
			run := WorkflowRun{ProjectID: 10, EnvironmentID: 7, ContextSnapshot: mustJSON(map[string]interface{}{"target": original})}
			if err := worker.validateEnvironmentTarget(context.Background(), run, TaskRun{NodeType: kind}); err != nil {
				t.Fatal(err)
			}
			current.Namespace = "risk-prod"
			if err := worker.validateEnvironmentTarget(context.Background(), run, TaskRun{NodeType: kind}); err == nil {
				t.Fatal("accepted changed namespace")
			}
			current = original
			current.HostGroupRef = "production-hosts"
			if err := worker.validateEnvironmentTarget(context.Background(), run, TaskRun{NodeType: kind}); err == nil {
				t.Fatal("accepted changed host group")
			}
		})
	}
}
