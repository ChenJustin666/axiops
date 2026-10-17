package workflow

import "testing"

func TestProductionBatchRejectsCrossServiceResourceCollision(t *testing.T) {
	owners := map[string]int64{}
	preview := &ProductionPreview{Namespace: "prod", Candidate: ProductionCandidate{Verification: Verification{ServiceID: 1}}, TargetManifest: "apiVersion: v1\nkind: ConfigMap\nmetadata: {name: shared}\ndata: {key: value}\n"}
	if err := checkProductionBatchResources(owners, preview); err != nil {
		t.Fatal(err)
	}
	preview.Candidate.ServiceID = 2
	if err := checkProductionBatchResources(owners, preview); err == nil {
		t.Fatal("two services can overwrite the same resource")
	}
	preview.TargetManifest = "apiVersion: v1\nkind: ConfigMap\nmetadata: {name: independent}\ndata: {key: value}\n"
	if err := checkProductionBatchResources(owners, preview); err != nil {
		t.Fatal(err)
	}
}

func TestProductionServiceGraphKeepsPolicyAndRejectsUnsupportedNodes(t *testing.T) {
	nodes := []Node{
		{Key: "approval", Type: NodeApproval},
		{Key: "build-a", Type: NodeBuild, DependsOn: []string{"approval"}, Config: mustJSON(map[string]int{"service_id": 1})},
		{Key: "deploy-a", Type: NodeYAMLDeploy, DependsOn: []string{"build-a"}, Config: mustJSON(map[string]int{"service_id": 1})},
		{Key: "build-b", Type: NodeBuild, DependsOn: []string{"approval"}, Config: mustJSON(map[string]int{"service_id": 2})},
		{Key: "deploy-b", Type: NodeYAMLDeploy, DependsOn: []string{"build-b", "deploy-a"}, Config: mustJSON(map[string]int{"service_id": 2})},
		{Key: "notify", Type: NodeNotify, DependsOn: []string{"deploy-a", "deploy-b"}},
	}
	selected, err := productionServiceNodes(nodes, 2)
	if err != nil || len(selected) != 4 || selected[0].Type != NodeApproval || len(selected[2].DependsOn) != 1 || selected[3].DependsOn[0] != "deploy-b" {
		t.Fatalf("invalid selected graph: %+v %v", selected, err)
	}
	if _, err = productionServiceNodes(nodes, 3); err == nil {
		t.Fatal("missing service accepted")
	}
	withTest := append(append([]Node(nil), nodes...), Node{Key: "test", Type: NodeTest, Config: mustJSON(map[string]int{"service_id": 2})})
	if _, err = productionServiceNodes(withTest, 2); err == nil {
		t.Fatal("test silently removed")
	}
	nodes[3].DependsOn = []string{"build-a"}
	if _, err = productionServiceNodes(nodes, 2); err == nil {
		t.Fatal("cross-service build dependency silently removed")
	}
}
