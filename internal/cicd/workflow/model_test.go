package workflow

import "testing"

func TestValidateDefinitionAcceptsParallelBuildAndDeploy(t *testing.T) {
	issues := ValidateDefinition(Definition{DeliveryType: DeliveryTypeHelm, Nodes: []Node{
		{Key: "build", Name: "Build", Type: NodeBuild},
		{Key: "deploy", Name: "Deploy", Type: NodeHelmDeploy, DependsOn: []string{"build"}},
		{Key: "notify", Name: "Notify", Type: NodeNotify, DependsOn: []string{"deploy"}},
	}})
	if len(issues) != 0 {
		t.Fatalf("unexpected validation issues: %+v", issues)
	}
}

func TestValidateDefinitionRejectsCycleAndWrongDeployType(t *testing.T) {
	issues := ValidateDefinition(Definition{DeliveryType: DeliveryTypeK8sYAML, Nodes: []Node{
		{Key: "a", Name: "A", Type: NodeBuild, DependsOn: []string{"b"}},
		{Key: "b", Name: "B", Type: NodeHelmDeploy, DependsOn: []string{"a"}},
	}})
	seen := map[string]bool{}
	for _, item := range issues {
		seen[item.Code] = true
	}
	if !seen["cycle_detected"] || !seen["incompatible_node_type"] {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateDefinitionRejectsMissingDependencyAndLimits(t *testing.T) {
	issues := ValidateDefinition(Definition{DeliveryType: DeliveryTypeHost, Nodes: []Node{
		{Key: "deploy", Name: "Deploy", Type: NodeHostDeploy, DependsOn: []string{"missing"}, TimeoutSeconds: 86401, RetryLimit: 11},
	}})
	seen := map[string]bool{}
	for _, item := range issues {
		seen[item.Code] = true
	}
	for _, code := range []string{"missing_dependency", "invalid_timeout", "invalid_retry"} {
		if !seen[code] {
			t.Fatalf("missing %s in issues=%+v", code, issues)
		}
	}
}

func TestValidateDefinitionAcceptsSubmitOrderWithoutSuccessDependency(t *testing.T) {
	server := mustJSON(map[string]int{"service_id": 47, "start_order": 1})
	gateway := mustJSON(map[string]int{"service_id": 69, "start_order": 2})
	nodes := []Node{
		{Key: "server", Name: "Server", Type: NodeYAMLDeploy, Config: server},
		{Key: "gateway", Name: "Gateway", Type: NodeYAMLDeploy, Config: gateway},
	}
	if issues := ValidateDefinition(Definition{DeliveryType: DeliveryTypeK8sYAML, Nodes: nodes}); len(issues) != 0 {
		t.Fatalf("unexpected validation issues: %+v", issues)
	}
	nodes[1].DependsOn = []string{"server"}
	issues := ValidateDefinition(Definition{DeliveryType: DeliveryTypeK8sYAML, Nodes: nodes})
	if len(issues) == 0 || issues[0].Code != "ordered_cross_service_dependency" {
		t.Fatalf("cross-service success dependency should be rejected: %+v", issues)
	}
}
