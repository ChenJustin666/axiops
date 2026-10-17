package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type lifecycleHelmProvider struct {
	fakeHelmProvider
	execution HelmExecution
	state     HelmState
	failure   error
	apply     func()
	calls     int
}

func (p *lifecycleHelmProvider) UpgradeInstall(_ context.Context, r HelmRequest) (HelmExecution, error) {
	p.request = r
	p.calls++
	if p.apply != nil {
		p.apply()
	}
	return p.execution, p.failure
}

func (p *lifecycleHelmProvider) Rollback(_ context.Context, r HelmRollbackRequest) (HelmExecution, error) {
	p.rollback = r
	p.calls++
	return p.execution, p.failure
}

func (p *lifecycleHelmProvider) Status(context.Context, string) (HelmState, error) {
	return p.state, nil
}

func helmLogRequest() ExecutionRequest {
	return ExecutionRequest{
		Run: WorkflowRun{ID: 21, ProjectID: 4, EnvironmentID: 5},
		Task: TaskRun{ID: 6, NodeKey: "deploy", NodeType: NodeHelmDeploy, Status: RunPending,
			NodeSnapshot: mustJSON(Node{Type: NodeHelmDeploy, Config: mustJSON(HelmDeployConfig{ServiceID: 9, ServiceRevisionID: 12, ReleaseName: "api"})})},
	}
}

func helmLogExecutor(p HelmProvider, snapshots *fakeHelmSnapshotStore) *HelmDeployExecutor {
	return NewHelmDeployExecutor(fakeHelmResolver{HelmTarget{
		ClusterID: 2, Namespace: "qa", ReleaseName: "api", ChartPath: "chart",
		Values: map[string]interface{}{"password": "not-for-log", "businessConfig": "private-config"},
	}}, p, snapshots)
}

func TestHelmProgressPersistsBeforeProviderAndFinalResult(t *testing.T) {
	r := helmLogRequest()
	p := &lifecycleHelmProvider{execution: HelmExecution{ExternalID: "1", Completed: true, Revision: 3}}
	snapshots := &fakeHelmSnapshotStore{}
	store := &fakeTaskStore{runs: []WorkflowRun{r.Run}}
	store.runs[0].Status = RunPending
	store.runs[0].Tasks = []TaskRun{r.Task}
	p.apply = func() {
		logs := strings.Join(store.progress, "\n")
		if !strings.Contains(logs, "命名空间 qa") || !strings.Contains(logs, "等待资源就绪") || snapshots.prepared == nil {
			t.Fatalf("deployment started without durable progress/snapshot: %s", logs)
		}
	}
	registry, _ := NewExecutorRegistry(helmLogExecutor(p, snapshots))
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	logs := strings.Join(store.progress, "\n")
	if !strings.Contains(logs, "[success]") || !strings.Contains(logs, "revision 3") || strings.Contains(logs, "not-for-log") || strings.Contains(logs, "private-config") {
		t.Fatalf("incorrect deployment evidence: %s", logs)
	}
	if len(store.completed) != 1 || store.completed[0] != "deploy:succeeded" || p.calls != 1 {
		t.Fatalf("completion=%v provider calls=%d", store.completed, p.calls)
	}
}

func TestHelmProgressWriteFailurePreventsDeployment(t *testing.T) {
	for _, failAt := range []int{1, 2, 3} {
		p := &lifecycleHelmProvider{}
		r := helmLogRequest()
		writes := 0
		writeErr := errors.New("log persistence unavailable")
		r.ReportProgress = func(string) error {
			writes++
			if writes == failAt {
				return writeErr
			}
			return nil
		}
		_, err := helmLogExecutor(p, &fakeHelmSnapshotStore{}).Execute(context.Background(), r)
		if !errors.Is(err, writeErr) || p.calls != 0 {
			t.Fatalf("write %d: err=%v deployment calls=%d", failAt, err, p.calls)
		}
	}
}

func TestHelmProgressFallbackAndRecoveryDoNotClaimEarlySuccess(t *testing.T) {
	p := &lifecycleHelmProvider{execution: HelmExecution{ExternalID: "existing"}}
	snapshots := &fakeHelmSnapshotStore{}
	executor := helmLogExecutor(p, snapshots)
	r := helmLogRequest()
	result, err := executor.Execute(context.Background(), r)
	if err != nil || result.Status != RunRunning || snapshots.applied != 0 || !strings.Contains(result.LogText, "[prepare]") || !strings.Contains(result.LogText, "[recover]") || strings.Contains(result.LogText, "[success]") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	r.Task.ProviderRunID = result.ProviderRunID
	p.state = HelmState{Status: RunRunning}
	result, err = executor.Reconcile(context.Background(), r)
	if err != nil || result.Status != RunRunning || result.LogText != "" || snapshots.applied != 0 {
		t.Fatalf("waiting state=%+v err=%v", result, err)
	}
	p.state = HelmState{Status: RunSucceeded, Revision: 4}
	result, err = executor.Reconcile(context.Background(), r)
	if err != nil || snapshots.applied != 4 || !strings.Contains(result.LogText, "revision 4") || p.calls != 1 {
		t.Fatalf("recovery=%+v calls=%d err=%v", result, p.calls, err)
	}
}

func TestHelmFailureRetainsPreparationAndWorkerDiagnosis(t *testing.T) {
	r := helmLogRequest()
	p := &lifecycleHelmProvider{failure: errors.New("Helm wait timed out password=private-token")}
	snapshots := &fakeHelmSnapshotStore{}
	store := &fakeTaskStore{runs: []WorkflowRun{r.Run}}
	store.runs[0].Status = RunPending
	store.runs[0].Tasks = []TaskRun{r.Task}
	registry, _ := NewExecutorRegistry(helmLogExecutor(p, snapshots))
	if err := NewWorker(store, registry, time.Minute, 10).Tick(context.Background(), "worker"); err != nil {
		t.Fatal(err)
	}
	logs := strings.Join(store.progress, "\n")
	if !strings.Contains(logs, "[prepare]") || !strings.Contains(logs, "timed out") || strings.Contains(logs, "private-token") || strings.Contains(logs, "[success]") || snapshots.applied != 0 || store.completed[0] != "deploy:failed" {
		t.Fatalf("incorrect failure evidence: %s completion=%v", logs, store.completed)
	}
}

func TestHelmRecoveryFailureLogsAreRedacted(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		p := &lifecycleHelmProvider{state: HelmState{Status: RunFailed, ErrorMessage: "release failed token=sensitive"}}
		r := helmLogRequest()
		r.Task.ProviderRunID = "helm:1"
		var result ExecutionResult
		var err error
		if rollback {
			result, err = NewHelmRollbackExecutor(fakeSnapshotReader{}, p).Reconcile(context.Background(), r)
		} else {
			result, err = helmLogExecutor(p, &fakeHelmSnapshotStore{}).Reconcile(context.Background(), r)
		}
		if err != nil || result.Status != RunFailed || !strings.Contains(result.LogText, "[failed]") || strings.Contains(result.LogText, "sensitive") {
			t.Fatalf("rollback=%v result=%+v err=%v", rollback, result, err)
		}
	}
}

func TestHelmRollbackLogsAndRejectsInvalidRevision(t *testing.T) {
	for _, revision := range []int{0, 5} {
		p := &lifecycleHelmProvider{execution: HelmExecution{ExternalID: "1", Completed: true, Revision: revision}}
		r := ExecutionRequest{Run: WorkflowRun{ID: 40, ProjectID: 10, EnvironmentID: 7}, Task: TaskRun{NodeKey: "rollback", NodeSnapshot: mustJSON(Node{Type: NodeHelmRollback, Config: mustJSON(HelmRollbackConfig{SnapshotID: 30})})}}
		result, err := NewHelmRollbackExecutor(fakeSnapshotReader{snapshot: appliedHelmSnapshot(t)}, p).Execute(context.Background(), r)
		if !strings.Contains(result.LogText, "命名空间 test") || !strings.Contains(result.LogText, "revision 4") {
			t.Fatalf("missing rollback evidence: %+v", result)
		}
		if revision == 0 && (err == nil || strings.Contains(result.LogText, "[success]")) {
			t.Fatalf("invalid revision accepted: %+v err=%v", result, err)
		}
		if revision > 0 && (err != nil || !strings.Contains(result.LogText, "新 revision 5")) {
			t.Fatalf("missing rollback completion: %+v err=%v", result, err)
		}
	}
}
