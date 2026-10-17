package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func jobManifest() []byte {
	return []byte("apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: migrate-r2\nspec:\n  template:\n    spec:\n      restartPolicy: Never\n      containers:\n      - name: task\n        image: registry.example/task@sha256:" + strings.Repeat("a", 64) + "\n")
}

func TestOrdinaryHelmJobsNeverEnableAutomaticReplay(t *testing.T) {
	application := []byte("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n")
	for _, test := range []struct {
		next, previous []byte
		atomic         bool
	}{{application, nil, true}, {jobManifest(), nil, false}, {jobManifest(), application, false}, {application, jobManifest(), false}} {
		atomic, err := helmAtomicPolicy(test.next, test.previous, "qa")
		if err != nil || atomic != test.atomic {
			t.Fatal("unsafe rollback policy", atomic, err)
		}
	}
	renderer := helmPostRenderer{namespace: "qa", rejectJobs: true}
	if _, err := renderer.Run(bytes.NewBuffer(jobManifest())); err == nil {
		t.Fatal("upgrade-only Job entered atomic release")
	}
	renderer.rejectJobs = false
	if _, err := renderer.Run(bytes.NewBuffer(jobManifest())); err != nil {
		t.Fatal("explicit Job release rejected", err)
	}
	if err := validateHelmRollbackHistory(HelmRollbackRequest{Namespace: "qa"}, jobManifest()); err == nil {
		t.Fatal("ordinary Job accepted as rollback target")
	}
}

type batchHelmTestProvider struct{ fakeHelmProvider }

func (p *batchHelmTestProvider) Render(context.Context, HelmRequest) ([]byte, error) {
	return jobManifest(), nil
}
func TestJobDeploymentWaitsWithoutAtomicReplayAndCannotBeRolledBack(t *testing.T) {
	provider := &batchHelmTestProvider{}
	snapshots := &fakeHelmSnapshotStore{}
	result, err := helmLogExecutor(provider, snapshots).Execute(context.Background(), helmLogRequest())
	if err != nil {
		t.Fatal(err)
	}
	if provider.request.Atomic || !provider.request.Wait || !strings.Contains(result.LogText, "不自动回滚") || snapshots.prepared == nil {
		t.Fatal("incorrect Job execution policy")
	}
	snapshot := snapshots.prepared
	snapshot.ID = 30
	snapshot.Status = "applied"
	snapshot.HelmRevision = 2
	request := ExecutionRequest{Run: WorkflowRun{ID: 40, ProjectID: 4, EnvironmentID: 5}, Task: TaskRun{NodeKey: "rollback", NodeSnapshot: mustJSON(Node{Type: NodeHelmRollback, Config: mustJSON(HelmRollbackConfig{SnapshotID: 30})})}}
	if _, err := NewHelmRollbackExecutor(fakeSnapshotReader{snapshot: snapshot}, provider).Execute(context.Background(), request); err == nil {
		t.Fatal("Job replay passed executor")
	}
	var inventory []helmResourceIdentity
	if err := json.Unmarshal(snapshot.RenderedManifest, &inventory); err != nil || len(inventory) != 1 || inventory[0].Kind != "Job" {
		t.Fatal("Job inventory not recorded", err)
	}
}
