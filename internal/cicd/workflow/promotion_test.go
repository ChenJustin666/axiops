package workflow

import (
	"context"
	"encoding/json"
	"github.com/ChenJustin666/axiops/internal/cicd/serviceassets"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"strings"
	"testing"
)

type promotionRepository struct {
	*logicRepository
	build *PromotionBuild
}

func (r *promotionRepository) LoadPromotionBuild(context.Context, int64) (*PromotionBuild, error) {
	return r.build, nil
}
func (r *promotionRepository) FreezeDeploymentConfigs(_ context.Context, eid int64, nodes []Node, _ bool) ([]Node, error) {
	result := append([]Node(nil), nodes...)
	for i, n := range result {
		if n.Type == NodeYAMLDeploy {
			var c YAMLDeployConfig
			json.Unmarshal(n.Config, &c)
			c.FrozenYAML = &YAMLSource{ServiceID: c.ServiceID, EnvironmentID: eid, Namespace: "app-prod", Variables: map[string]string{"DB_HOST": "prod-db"}, Assets: serviceassets.Settings{Resources: &serviceassets.RuntimeResources{MemoryLimit: "4Gi"}}}
			result[i].Config = mustJSON(c)
		}
	}
	return result, nil
}
func promotionFixture() (*Logic, *promotionRepository, *RunRequest) {
	repo := &promotionRepository{logicRepository: &logicRepository{
		item: &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, DefinitionVersion: 1, Status: "active", Nodes: []Node{
			{Key: "approval", Type: NodeApproval},
			{Key: "build", Type: NodeBuild, DependsOn: []string{"approval"}, Config: mustJSON(map[string]interface{}{"service_id": 1})},
			{Key: "deploy", Type: NodeYAMLDeploy, DependsOn: []string{"build"}, Config: mustJSON(map[string]interface{}{"service_id": 1, "service_revision_id": 11})},
		}}, env: &EnvironmentTarget{ID: 7, ProjectID: 10, EnvironmentType: "k8s_namespace", BindingStatus: "validated", Status: "active", DeployApproval: true}, release: &ReleaseTarget{ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML}},
		build: &PromotionBuild{ID: 42, ProjectID: 10, ServiceID: 1, BuildNumber: 1, Status: "success", ImageURL: "registry.example/team/app:test-1", ImageDigest: "sha256:" + strings.Repeat("a", 64), Commit: strings.Repeat("b", 40)},
	}
	policies := policyStore{}
	for _, p := range [][2]string{{"workflow", "view"}, {"workflow", "run"}, {"service", "view"}, {"build", "view"}} {
		policies = append(policies, authz.Policy{Resource: p[0], Action: p[1], Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10})
	}
	return &Logic{repo: repo, authorizer: authz.New(policies)}, repo, &RunRequest{EnvironmentID: 7, ServiceID: 1, ServiceRevisionID: 11, UseEnvironmentConfig: true, ExecutionMode: "existing_image", SourceBuildID: 42}
}
func TestPromotionUsesChosenDigestAndOnlyTargetEnvironmentConfig(t *testing.T) {
	logic, repo, req := promotionFixture()
	_, code, message := logic.RunWithValidation(context.Background(), 7, 9, req)
	if code != 0 {
		t.Fatalf("code=%d message=%s", code, message)
	}
	if repo.run.Status != RunPendingApproval || len(repo.tasks) != 2 {
		t.Fatal("promotion bypassed approval or created a build")
	}
	var node Node
	json.Unmarshal(repo.tasks[1].NodeSnapshot, &node)
	var c YAMLDeployConfig
	json.Unmarshal(node.Config, &c)
	if len(node.DependsOn) != 1 || node.DependsOn[0] != "approval" || c.FrozenYAML.Namespace != "app-prod" || c.FrozenYAML.Variables["DB_HOST"] != "prod-db" || c.FrozenYAML.Assets.Resources.MemoryLimit != "4Gi" || c.FrozenYAML.ImageRef != repo.build.ImageURL+"@"+repo.build.ImageDigest {
		t.Fatalf("incorrect promotion: %s", node.Config)
	}
	if len(repo.item.Nodes) != 3 {
		t.Fatal("promotion changed the saved workflow")
	}
	if !strings.Contains(string(repo.run.ContextSnapshot), `"source_build"`) {
		t.Fatal("source build provenance was not saved")
	}
}

func TestProductionOrdinaryExecutionRequiresExplicitBuildOrImageEntry(t *testing.T) {
	for _, mode := range []string{"", "build", "existing_image"} {
		t.Run(mode, func(t *testing.T) {
			logic, repo, req := promotionFixture()
			repo.env.IsProduction = true
			req.ExecutionMode = mode
			if mode != "existing_image" {
				req.SourceBuildID = 0
				req.Branch = "main"
			}
			_, code, message := logic.RunWithValidation(context.Background(), 7, 9, req)
			if code == 0 || repo.run != nil || message == "" {
				t.Fatalf("ordinary production execution bypassed verification: code=%d message=%s", code, message)
			}
		})
	}
}
func TestPromotionRejectsWrongArtifactAndIncompatibleRequests(t *testing.T) {
	for _, modify := range []func(*promotionRepository, *RunRequest){
		func(r *promotionRepository, q *RunRequest) { r.build.ProjectID = 99 },
		func(r *promotionRepository, q *RunRequest) { r.build.ServiceID = 99 },
		func(r *promotionRepository, q *RunRequest) { r.build.Status = "failed" },
		func(r *promotionRepository, q *RunRequest) { r.build.ImageDigest = "" },
		func(r *promotionRepository, q *RunRequest) { q.Branch = "main" },
		func(r *promotionRepository, q *RunRequest) { q.UseEnvironmentConfig = false },
		func(r *promotionRepository, q *RunRequest) { q.EnvironmentID = 9 },
		func(r *promotionRepository, q *RunRequest) {
			r.item.Nodes = append(r.item.Nodes, Node{Key: "test", Type: NodeTest})
		},
	} {
		logic, r, q := promotionFixture()
		modify(r, q)
		_, code, _ := logic.RunWithValidation(context.Background(), 7, 9, q)
		if code == 0 || r.run != nil {
			t.Fatal("unsafe promotion accepted")
		}
	}
	logic, r, q := promotionFixture()
	logic.authorizer = authz.New(policyStore{{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}, {Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10}})
	_, code, _ := logic.RunWithValidation(context.Background(), 7, 9, q)
	if code == 0 || r.run != nil {
		t.Fatal("missing artifact permission accepted")
	}
}
