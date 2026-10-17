package workflow

import "testing"

func TestPlanReturnsParallelReadyTasks(t *testing.T) {
	plan, err := Plan(WorkflowRun{Status: RunPending, Tasks: []TaskRun{
		{NodeKey: "build", Status: RunSucceeded},
		{NodeKey: "yaml", Status: RunPending, DependsOn: []string{"build"}},
		{NodeKey: "notify", Status: RunPending, DependsOn: []string{"build"}},
	}})
	if err != nil || len(plan.Runnable) != 2 || plan.Runnable[0] != "yaml" || plan.Runnable[1] != "notify" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanWaitsForRunningDependency(t *testing.T) {
	plan, err := Plan(WorkflowRun{Status: RunRunning, Tasks: []TaskRun{
		{NodeKey: "build", Status: RunRunning},
		{NodeKey: "deploy", Status: RunPending, DependsOn: []string{"build"}},
	}})
	if err != nil || len(plan.Waiting) != 1 || plan.Waiting[0] != "deploy" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanSkipsTaskAfterFailedDependency(t *testing.T) {
	plan, err := Plan(WorkflowRun{Status: RunRunning, Tasks: []TaskRun{
		{NodeKey: "build", Status: RunFailed},
		{NodeKey: "deploy", Status: RunPending, DependsOn: []string{"build"}},
	}})
	if err != nil || len(plan.Skippable) != 1 || plan.Skippable[0] != "deploy" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanRejectsUnknownDependency(t *testing.T) {
	if _, err := Plan(WorkflowRun{Status: RunPending, Tasks: []TaskRun{{NodeKey: "deploy", Status: RunPending, DependsOn: []string{"missing"}}}}); err == nil {
		t.Fatal("expected unknown dependency error")
	}
}

func TestPlanDeploysServicesInSavedOrder(t *testing.T) {
	tasks := []TaskRun{
		{NodeKey: "build-server", Status: RunSucceeded},
		{NodeKey: "deploy-server", NodeType: NodeYAMLDeploy, NodeSnapshot: mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 1})}), Status: RunRunning, DependsOn: []string{"build-server"}},
		{NodeKey: "build-gateway", Status: RunSucceeded},
		{NodeKey: "deploy-gateway", NodeType: NodeYAMLDeploy, NodeSnapshot: mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 2})}), Status: RunPending, DependsOn: []string{"build-gateway"}},
	}
	plan, err := Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Runnable) != 0 || len(plan.Waiting) != 1 || plan.Waiting[0] != "deploy-gateway" {
		t.Fatalf("before server submission: plan=%+v err=%v", plan, err)
	}
	tasks[1].ProviderRunID = "yaml:server"
	plan, err = Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Runnable) != 1 || plan.Runnable[0] != "deploy-gateway" {
		t.Fatalf("after server submission: plan=%+v err=%v", plan, err)
	}
	tasks[1].Status = RunFailed
	tasks[1].ProviderRunID = ""
	plan, err = Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Runnable) != 1 || plan.Runnable[0] != "deploy-gateway" {
		t.Fatalf("after server failure: plan=%+v err=%v", plan, err)
	}
}

func TestPlanBuildsAllServicesBeforeOrderedDeployment(t *testing.T) {
	server := mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 1})})
	gateway := mustJSON(Node{Config: mustJSON(map[string]int{"start_order": 2})})
	tasks := []TaskRun{
		{NodeKey: "build-server", NodeType: NodeBuild, Status: RunSucceeded},
		{NodeKey: "deploy-server", NodeType: NodeYAMLDeploy, NodeSnapshot: server, Status: RunPending},
		{NodeKey: "build-gateway", NodeType: NodeBuild, Status: RunRunning},
		{NodeKey: "deploy-gateway", NodeType: NodeYAMLDeploy, NodeSnapshot: gateway, Status: RunPending},
	}
	plan, err := Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Runnable) != 0 || len(plan.Waiting) != 2 {
		t.Fatalf("build barrier not applied: plan=%+v err=%v", plan, err)
	}
	tasks[2].Status = RunSucceeded
	plan, err = Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Runnable) != 1 || plan.Runnable[0] != "deploy-server" {
		t.Fatalf("first deployment should start after all builds: plan=%+v err=%v", plan, err)
	}
	tasks[2].Status = RunFailed
	plan, err = Plan(WorkflowRun{Status: RunRunning, Tasks: tasks})
	if err != nil || len(plan.Skippable) != 2 {
		t.Fatalf("failed build should block the deployment phase: plan=%+v err=%v", plan, err)
	}
}
