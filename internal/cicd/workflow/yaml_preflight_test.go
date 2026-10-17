package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func TestRunPreflightRejectsMissingVariablesBeforeCreatingBuildTasks(t *testing.T) {
	source := &YAMLSource{ProjectID: 10, ServiceID: 9, ServiceRevisionID: 12, DeliveryType: DeliveryTypeK8sYAML, DefinitionFormat: DeliveryTypeK8sYAML, EnvironmentID: 7, EnvironmentType: "k8s_namespace", BindingStatus: "validated", EnvironmentStatus: "active", ClusterID: 2, Namespace: "qa", Manifest: []byte("apiVersion: v1\nkind: Service\nmetadata:\n  name: api\nspec:\n  ports:\n  - port: {{.port}}\n")}
	repo := &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{
			{Key: "build", Type: NodeBuild, Config: mustJSON(BuildNodeConfig{ServiceID: 9})},
			{Key: "deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}, Config: mustJSON(YAMLDeployConfig{ServiceID: 9, ServiceRevisionID: 12})},
		}},
		env: &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active"},
	}
	logic := &Logic{repo: repo, yamlResolver: NewDefaultYAMLResolver(fakeYAMLSourceStore{source: source}), authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	id, code, message := logic.RunWithValidation(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if id != 0 || code != errcode.CodeParamInvalid || repo.run != nil || !strings.Contains(message, "undefined variables: port") {
		t.Fatalf("id=%d code=%d message=%s run=%+v", id, code, message, repo.run)
	}
	source.Variables = map[string]string{"port": "8001"}
	id, code, message = logic.RunWithValidation(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if id != 21 || code != errcode.CodeSuccess || message != "" || len(repo.tasks) != 2 {
		t.Fatalf("id=%d code=%d message=%s", id, code, message)
	}
	// An upstream build supplies Image later, without persisting validation data.
	source.Manifest = []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\ndata:\n  image: {{.Image}}\n")
	id, code, message = logic.RunWithValidation(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if id != 21 || code != errcode.CodeSuccess || message != "" || source.ImageName != "" {
		t.Fatalf("pending build image rejected: id=%d code=%d message=%s", id, code, message)
	}
	// The same build running in parallel cannot satisfy this dependency.
	repo.item.Nodes[1].DependsOn = nil
	id, code, message = logic.RunWithValidation(context.Background(), 7, 9, &RunRequest{EnvironmentID: 7})
	if id != 0 || code != errcode.CodeParamInvalid || !strings.Contains(message, "Image") {
		t.Fatalf("unrelated build bypassed image validation: id=%d code=%d message=%s", id, code, message)
	}
}
