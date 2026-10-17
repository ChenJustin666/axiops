package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
	"github.com/ChenJustin666/axiops/internal/shared/errcode"
)

func TestRunSelectedBatchCreatesOneRunWithOwnBranchesAndSubmissionOrder(t *testing.T) {
	nodes := []Node{
		{Key: "build-server", Type: NodeBuild, Config: mustJSON(map[string]int{"service_id": 47})},
		{Key: "deploy-server", Type: NodeYAMLDeploy, Config: mustJSON(map[string]int{"service_id": 47}), DependsOn: []string{"build-server"}},
		{Key: "build-gateway", Type: NodeBuild, Config: mustJSON(map[string]int{"service_id": 69})},
		{Key: "deploy-gateway", Type: NodeYAMLDeploy, Config: mustJSON(map[string]int{"service_id": 69}), DependsOn: []string{"build-gateway"}},
	}
	repo := &logicRepository{
		item:    &Workflow{ID: 9, ProjectID: 10, TargetEnvironmentID: 7, DeliveryType: DeliveryTypeK8sYAML, Status: "active", Nodes: nodes},
		env:     &EnvironmentTarget{ID: 7, ProjectID: 10, Code: "qa", EnvironmentType: "k8s_namespace", ClusterID: 2, Namespace: "qa", Status: "active", BindingStatus: "validated"},
		release: &ReleaseTarget{ProjectID: 10, DeliveryType: DeliveryTypeK8sYAML},
	}
	logic := &Logic{repo: repo, authorizer: authz.New(policyStore{
		{Resource: "workflow", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "workflow", Action: "run", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
		{Resource: "service", Action: "view", Effect: "allow", ScopeType: authz.ScopeProject, ScopeID: 10},
	})}
	req := &RunRequest{EnvironmentID: 7, UseEnvironmentConfig: true, Services: []RunServiceSelection{
		{ServiceID: 69, ServiceRevisionID: 12, Branch: "gateway-branch"},
		{ServiceID: 47, ServiceRevisionID: 11, Branch: "server-branch"},
	}}
	id, code := logic.Run(context.Background(), 7, 9, req)
	if code != errcode.CodeSuccess || id != 21 || len(repo.tasks) != 4 {
		t.Fatalf("run=%d code=%d tasks=%v", id, code, repo.tasks)
	}
	for _, task := range repo.tasks {
		var n Node
		_ = json.Unmarshal(task.NodeSnapshot, &n)
		var config struct {
			ServiceID  int64  `json:"service_id"`
			Branch     string `json:"branch"`
			StartOrder int    `json:"start_order"`
		}
		_ = json.Unmarshal(n.Config, &config)
		if task.NodeType == NodeBuild {
			expected := "gateway-branch"
			if config.ServiceID == 47 {
				expected = "server-branch"
			}
			if config.Branch != expected {
				t.Fatalf("wrong service branch: %s", n.Config)
			}
		} else {
			expected := 1
			if config.ServiceID == 47 {
				expected = 2
			}
			if config.StartOrder != expected || len(task.DependsOn) != 1 {
				t.Fatalf("wrong deployment order: %+v", task)
			}
		}
	}
	// Invalid selection never reaches CreateRun and never alters the definition.
	for _, selections := range [][]RunServiceSelection{
		{req.Services[0], req.Services[0]},
		{req.Services[0], {ServiceID: 999, ServiceRevisionID: 1, Branch: "main"}},
		{req.Services[0], {ServiceID: 47, ServiceRevisionID: 11}},
	} {
		repo.run = nil
		req.Services = selections
		_, code := logic.Run(context.Background(), 7, 9, req)
		if code != errcode.CodeParamInvalid || repo.run != nil {
			t.Fatalf("invalid batch queued: code=%d", code)
		}
	}
	if string(nodes[0].Config) != `{"service_id":47}` {
		t.Fatal("saved definition mutated")
	}
}

type selectedPromotionRepository struct {
	*promotionRepository
	builds map[int64]*PromotionBuild
}

func (r *selectedPromotionRepository) LoadPromotionBuild(_ context.Context, id int64) (*PromotionBuild, error) {
	return r.builds[id], nil
}

func TestSelectedBatchPromotionKeepsEnvironmentAndOrderWithoutBuilds(t *testing.T) {
	logic, base, _ := promotionFixture()
	base.env.IsProduction = false
	base.env.DeployApproval = false
	base.item.Nodes = []Node{
		{Key: "build-a", Type: NodeBuild, Config: mustJSON(map[string]int{"service_id": 1})},
		{Key: "deploy-a", Type: NodeYAMLDeploy, Config: mustJSON(map[string]int{"service_id": 1}), DependsOn: []string{"build-a"}},
		{Key: "build-b", Type: NodeBuild, Config: mustJSON(map[string]int{"service_id": 2})},
		{Key: "deploy-b", Type: NodeYAMLDeploy, Config: mustJSON(map[string]int{"service_id": 2}), DependsOn: []string{"build-b"}},
	}
	other := *base.build
	other.ID, other.ServiceID = 43, 2
	other.ImageDigest = "sha256:" + strings.Repeat("c", 64)
	repo := &selectedPromotionRepository{promotionRepository: base, builds: map[int64]*PromotionBuild{42: base.build, 43: &other}}
	logic.repo = repo
	request := &RunRequest{EnvironmentID: 7, UseEnvironmentConfig: true, ExecutionMode: "existing_image", Services: []RunServiceSelection{
		{ServiceID: 2, ServiceRevisionID: 12, SourceBuildID: 43},
		{ServiceID: 1, ServiceRevisionID: 11, SourceBuildID: 42},
	}}
	_, code, message := logic.RunWithValidation(context.Background(), 7, 9, request)
	if code != errcode.CodeSuccess || len(base.tasks) != 2 {
		t.Fatalf("promotion failed: %d %s", code, message)
	}
	for index, task := range base.tasks {
		var node Node
		_ = json.Unmarshal(task.NodeSnapshot, &node)
		var cfg YAMLDeployConfig
		_ = json.Unmarshal(node.Config, &cfg)
		want := repo.builds[request.Services[index].SourceBuildID]
		if task.NodeType != NodeYAMLDeploy || len(task.DependsOn) != 0 || cfg.StartOrder != index+1 || cfg.FrozenYAML.Variables["DB_HOST"] != "prod-db" || cfg.FrozenYAML.ImageRef != want.ImageURL+"@"+want.ImageDigest {
			t.Fatalf("wrong promotion snapshot: %s", task.NodeSnapshot)
		}
	}
	repo.run = nil
	request.Services[1].SourceBuildID = 43
	_, code, _ = logic.RunWithValidation(context.Background(), 7, 9, request)
	if code == errcode.CodeSuccess || repo.run != nil {
		t.Fatal("wrong-service image queued a partial run")
	}
}
