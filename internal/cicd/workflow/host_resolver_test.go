package workflow

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

type fakeHostSourceStore struct {
	source   *HostSource
	artifact *HostArtifact
	agents   []int64
	agentOS  string
}

func (f fakeHostSourceStore) LoadHostSource(context.Context, int64, int64, int64) (*HostSource, error) {
	return f.source, nil
}
func (f fakeHostSourceStore) LoadHostBuildArtifact(context.Context, int64) (*HostArtifact, error) {
	return f.artifact, nil
}

func (f fakeHostSourceStore) ListOnlineHostAgentIDs(_ context.Context, _ int64, _, osName string) ([]int64, error) {
	if f.agentOS != "" && f.agentOS != osName {
		return nil, nil
	}
	return f.agents, nil
}

func validHostSource() *HostSource {
	definition, _ := json.Marshal(HostSpec{OS: "linux", DeployScript: "./deploy", HealthScript: "./health", RollbackScript: "./rollback", TimeoutSeconds: 120})
	return &HostSource{ProjectID: 7, ServiceID: 8, ServiceRevisionID: 9, EnvironmentID: 10, DeliveryType: DeliveryTypeHost, DefinitionFormat: "host_spec", EnvironmentType: "host_group", BindingStatus: "validated", EnvironmentStatus: "active", HostGroupRef: "group-a", Definition: definition}
}

func TestFrozenHostTargetsDoNotExpandAndRejectMissingAgent(t *testing.T) {
	source := validHostSource()
	store := fakeHostSourceStore{source: source, artifact: &HostArtifact{URL: "https://artifact.example/app", SHA256: strings.Repeat("a", 64)}, agents: []int64{1, 2, 3}}
	run := WorkflowRun{ProjectID: 7, EnvironmentID: 10, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:19", NodeSnapshot: buildNodeSnapshot(8)}}}
	config := HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9, FrozenHost: source, FrozenAgentIDs: []int64{1, 2}}
	target, err := NewDefaultHostResolver(store).RenderHost(context.Background(), run, config)
	if err != nil || len(target.AgentIDs) != 2 {
		t.Fatalf("targets expanded: %+v %v", target, err)
	}
	store.agents = []int64{1, 3}
	if _, err := NewDefaultHostResolver(store).RenderHost(context.Background(), run, config); err == nil {
		t.Fatal("missing frozen agent replaced")
	}
}

func TestHostResolverUsesImmutableBuildArtifactAndOnlineAgents(t *testing.T) {
	store := fakeHostSourceStore{source: validHostSource(), artifact: &HostArtifact{URL: "https://artifacts.example.com/app.tgz", SHA256: strings.Repeat("a", 64), Size: 10}, agents: []int64{1, 2}, agentOS: "linux"}
	resolver := NewDefaultHostResolver(store)
	run := WorkflowRun{ID: 11, ProjectID: 7, EnvironmentID: 10, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:19", NodeSnapshot: buildNodeSnapshot(8)}}}
	target, err := resolver.RenderHost(context.Background(), run, HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9})
	if err != nil {
		t.Fatalf("render host: %v", err)
	}
	if target.ArtifactSHA256 != strings.Repeat("a", 64) || len(target.AgentIDs) != 2 || target.TimeoutSeconds != 120 || target.BatchSize != 1 {
		t.Fatalf("unexpected target: %+v", target)
	}
}

func TestHostResolverConvertsBatchPercentage(t *testing.T) {
	source := validHostSource()
	var spec HostSpec
	if err := json.Unmarshal(source.Definition, &spec); err != nil {
		t.Fatal(err)
	}
	spec.Rollout.BatchPercent = 40
	spec.Rollout.IntervalSeconds = 15
	source.Definition, _ = json.Marshal(spec)
	store := fakeHostSourceStore{
		source: source, artifact: &HostArtifact{URL: "https://artifacts.example.com/app.tgz", SHA256: strings.Repeat("a", 64), Size: 10},
		agents: []int64{1, 2, 3, 4, 5, 6}, agentOS: "linux",
	}
	run := WorkflowRun{ProjectID: 7, EnvironmentID: 10, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:19", NodeSnapshot: buildNodeSnapshot(8)}}}
	target, err := NewDefaultHostResolver(store).RenderHost(context.Background(), run, HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9})
	if err != nil || target.BatchSize != 3 || target.BatchIntervalSeconds != 15 {
		t.Fatalf("target=%+v err=%v", target, err)
	}
}

func TestHostResolverRejectsCrossProjectAndOfflineGroup(t *testing.T) {
	store := fakeHostSourceStore{source: validHostSource(), artifact: &HostArtifact{URL: "https://artifacts.example.com/app.tgz", SHA256: strings.Repeat("a", 64), Size: 10}}
	resolver := NewDefaultHostResolver(store)
	run := WorkflowRun{ProjectID: 99, EnvironmentID: 10}
	if _, err := resolver.RenderHost(context.Background(), run, HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9}); err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("expected project isolation error, got %v", err)
	}
	run.ProjectID = 7
	run.Tasks = []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:19", NodeSnapshot: buildNodeSnapshot(8)}}
	if _, err := resolver.RenderHost(context.Background(), run, HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9}); err == nil || !strings.Contains(err.Error(), "no online linux Agent") {
		t.Fatalf("expected offline group error, got %v", err)
	}
}

func TestHostResolverRejectsAgentWithDifferentOS(t *testing.T) {
	store := fakeHostSourceStore{source: validHostSource(), artifact: &HostArtifact{URL: "https://artifacts.example.com/app.tgz", SHA256: strings.Repeat("a", 64), Size: 10}, agents: []int64{1}, agentOS: "windows"}
	resolver := NewDefaultHostResolver(store)
	run := WorkflowRun{ProjectID: 7, EnvironmentID: 10, Tasks: []TaskRun{{NodeType: NodeBuild, Status: RunSucceeded, ProviderRunID: "build:19", NodeSnapshot: buildNodeSnapshot(8)}}}
	if _, err := resolver.RenderHost(context.Background(), run, HostDeployConfig{ServiceID: 8, ServiceRevisionID: 9}); err == nil || !strings.Contains(err.Error(), "no online linux Agent") {
		t.Fatalf("expected OS isolation error, got %v", err)
	}
}

func buildNodeSnapshot(serviceID int64) json.RawMessage {
	data, _ := json.Marshal(Node{Key: "build", Type: NodeBuild, Config: json.RawMessage(`{"service_id":` + strconv.FormatInt(serviceID, 10) + `}`)})
	return data
}
