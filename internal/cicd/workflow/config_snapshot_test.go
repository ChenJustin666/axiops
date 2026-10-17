package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPublicNodeSnapshotOmitsExecutionConfiguration(t *testing.T) {
	raw := json.RawMessage(`{"key":"deploy","config":{"service_id":1,"service_revision_id":2,"frozen_yaml":{"Variables":{"password":"secret"}},"frozen_helm":{},"frozen_helm_variables":[],"frozen_host":{"Definition":"secret"},"frozen_agent_ids":[1,2]}}`)
	public := string(publicNodeSnapshot(raw))
	if strings.Contains(public, "frozen") || strings.Contains(public, "secret") || !strings.Contains(public, `"service_revision_id":2`) {
		t.Fatal(public)
	}
	if !strings.Contains(string(raw), "secret") {
		t.Fatal("mutated persisted snapshot")
	}
}

func TestFrozenHelmValuesIgnoreLaterEnvironmentEdits(t *testing.T) {
	source := validHelmSource()
	resolver := NewDefaultHelmResolver(fakeHelmSourceStore{source: source, variables: []HelmVariable{{Key: "replicaCount", Value: "9"}}}, fakeCredentialDecryptor{})
	target, err := resolver.RenderHelm(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api", FrozenHelm: source, FrozenHelmVariables: []HelmVariable{{Key: "replicaCount", Value: "3"}}})
	if err != nil || target.Values["replicaCount"] != 3 {
		t.Fatalf("target=%+v err=%v", target, err)
	}
	changed := *source
	changed.Namespace = "other"
	_, err = resolver.RenderHelm(context.Background(), WorkflowRun{ProjectID: 7, EnvironmentID: 10}, HelmDeployConfig{ServiceID: 8, ServiceRevisionID: 9, ReleaseName: "api", FrozenHelm: &changed})
	if err == nil {
		t.Fatal("changed target accepted")
	}
}
