package workflow

import (
	"context"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func TestWorkflowReadsAndEditsUseBoundEnvironment(t *testing.T) {
	for _, action := range []string{"view", "update"} {
		for _, environmentCode := range []string{"qa", "prod"} {
			t.Run(action+"/"+environmentCode, func(t *testing.T) {
				repo := &logicRepository{
					item: &Workflow{ID: 9, ProjectID: 10, Name: "Release", Code: "release", TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1},
					env:  &EnvironmentTarget{ID: 7, ProjectID: 10, Code: environmentCode, ClusterID: 2},
					run:  &WorkflowRun{ID: 21, WorkflowID: 9, ProjectID: 10, EnvironmentID: 7},
				}
				logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
					{Resource: "workflow", Action: "*", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
					{Resource: "workflow", Action: "*", Effect: "deny", ScopeType: authz.ScopeProject, ScopeID: 10, Environment: "prod"},
				})}
				want := errcode.CodeSuccess
				if environmentCode == "prod" {
					want = errcode.CodeNoPermission
				}
				if action == "view" {
					if _, code := logic.Get(context.Background(), 7, 9); code != want {
						t.Fatalf("workflow read: %d want %d", code, want)
					}
					if _, code := logic.GetRun(context.Background(), 7, 21); code != want {
						t.Fatalf("run read: %d want %d", code, want)
					}
				} else {
					_, code := logic.Update(context.Background(), 7, 9, &UpdateWorkflowRequest{ExpectedVersion: 1, Definition: Definition{DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{{Key: "notice", Name: "Notice", Type: NodeNotify}}}})
					if code != want || repo.updated != (want == errcode.CodeSuccess) {
						t.Fatalf("workflow update: %d want %d; written=%v", code, want, repo.updated)
					}
				}
			})
		}
	}
}

func TestWorkflowEnvironmentScopedReaderAndCrossProjectTarget(t *testing.T) {
	repo := &logicRepository{item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7}, env: &EnvironmentTarget{ID: 7, ProjectID: 10, Code: "qa"}}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeEnvironment, ScopeID: 7, Environment: "qa"}})}
	if _, code := logic.Get(context.Background(), 7, 9); code != errcode.CodeSuccess {
		t.Fatalf("own environment reader rejected: %d", code)
	}
	repo.env.ProjectID = 20
	if _, code := logic.Get(context.Background(), 7, 9); code != errcode.CodeNoPermission {
		t.Fatalf("cross-project target not rejected: %d", code)
	}
	repo.env.ProjectID, repo.env.ID = 10, 8
	if _, code := logic.Get(context.Background(), 7, 9); code != errcode.CodeNoPermission {
		t.Fatalf("wrong target identity not rejected: %d", code)
	}
}
