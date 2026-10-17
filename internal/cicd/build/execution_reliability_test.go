package build

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type unavailableArchive struct{}

func TestZeroRetentionDeletesCompletedJobsWithoutTTLController(t *testing.T) {
	for _, condition := range []batchv1.JobConditionType{batchv1.JobComplete, batchv1.JobFailed} {
		t.Run(string(condition), func(t *testing.T) {
			ctx := context.Background()
			job, _ := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 34, IdempotencyKey: "build:34:submit"})
			job.Annotations = map[string]string{"axiops.io/retention-seconds": "0"}
			job.Status.Conditions = []batchv1.JobCondition{{Type: condition, Status: corev1.ConditionTrue}}
			client := fake.NewSimpleClientset(job)
			jobs := client.BatchV1().Jobs("ci")
			secrets := &recordingBuildSecrets{}
			provider := NewKubeJobProvider("ci", jobs, jobs).WithRuntimeData(secrets, nil, 1024)
			if err := provider.Finalize(ctx, ProviderExecution{BuildRunID: 34, JobName: job.Name}); err != nil {
				t.Fatal(err)
			}
			if _, err := jobs.Get(ctx, job.Name, metav1.GetOptions{}); !apierrors.IsNotFound(err) {
				t.Fatalf("zero retention only scheduled a TTL instead of deleting: %v", err)
			}
			if secrets.deleted != buildCredentialSecretName(34) {
				t.Fatal("build credentials not cleaned")
			}
		})
	}
}

func TestZeroRetentionRetriesUntilDeletionIsConfirmed(t *testing.T) {
	job, _ := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 35, IdempotencyKey: "build:35:submit"})
	job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}
	client := &terminatingJobClient{fakeJobClient: fakeJobClient{existing: job}}
	secrets := &recordingBuildSecrets{}
	err := NewKubeJobProvider("ci", client, client).WithRuntimeData(secrets, nil, 1024).Finalize(context.Background(), ProviderExecution{BuildRunID: 35, JobName: job.Name})
	var uncertain *ExecutionUncertainError
	if !errors.As(err, &uncertain) || client.deleted != job.Name || secrets.deleted != "" {
		t.Fatalf("unfinished deletion was acknowledged: %v", err)
	}
}

func TestEvictionReasonSurvivesJobCleanup(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}}}}
	pod := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed, Reason: "Evicted", Message: "The node was low on ephemeral-storage"}}
	state, err := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).Status(context.Background(), ProviderExecution{JobName: "job"})
	if err != nil || !strings.Contains(state.DiagnosticText, "ephemeral-storage") || !strings.Contains(state.ErrorMessage, "Evicted") {
		t.Fatalf("eviction evidence lost: %+v %v", state, err)
	}
}

func (unavailableArchive) Put(context.Context, string, []byte) (string, error) {
	return "", errors.New("storage offline")
}
func (unavailableArchive) Get(context.Context, string) ([]byte, error) {
	return nil, errors.New("storage offline")
}

func TestArchiveOutagePersistsDatabaseLogAndCompletes(t *testing.T) {
	s := &fakeRunningBuildStore{run: &BuildRun{ID: 1, Provider: "observable", Status: StatusRunning, JobName: "job"}}
	p, _ := NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusSuccess, LogText: "final output\n"}})
	if _, err := NewReconciler(s, p).WithLogArchive(unavailableArchive{}).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.status != StatusSuccess || s.logText != "final output\n" || !strings.Contains(s.logs["archive"], "database") {
		t.Fatalf("outage lost result: %+v", s)
	}
}

func TestMissingExecutionTerminatesInsteadOfRunningForever(t *testing.T) {
	client := &fakeJobClient{}
	s := &fakeRunningBuildStore{run: &BuildRun{ID: 1, Provider: KubeJobProviderName, Status: StatusRunning, JobName: "removed"}}
	p, _ := NewProviderRegistry(NewKubeJobProvider("ci", client, client))
	if _, err := NewReconciler(s, p).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.status != StatusFailed || !strings.Contains(s.errorMessage, "removed") {
		t.Fatalf("missing execution remained active: %+v", s)
	}
}

type terminalRetryStore struct {
	fakeRunningBuildStore
	first time.Time
}

func (s *terminalRetryStore) ObserveTerminal(context.Context, int64) (time.Time, error) {
	return s.first, nil
}
func TestFinalLogRetrySurvivesReconcilerRestartAndIsBounded(t *testing.T) {
	s := &terminalRetryStore{fakeRunningBuildStore: fakeRunningBuildStore{run: &BuildRun{ID: 1, Provider: "observable", Status: StatusRunning, JobName: "job"}}, first: time.Now()}
	p, _ := NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusFailed, LogsIncomplete: true, DiagnosticText: "log transport timed out", LogText: "partial"}})
	if _, err := NewReconciler(s, p).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.status != "" || s.logText != "partial" {
		t.Fatal("final logs not retried before completion")
	}
	s.first = time.Now().Add(-6 * time.Minute)
	if _, err := NewReconciler(s, p).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.status != StatusFailed || !strings.Contains(s.logs["system"], "timed out") {
		t.Fatal("final log outage stuck forever or lost diagnostic")
	}
}

func TestJobCreationTimeoutDoesNotDeletePossiblyMountedSecret(t *testing.T) {
	client := &fakeJobClient{createErr: context.DeadlineExceeded}
	secrets := &recordingBuildSecrets{}
	p := NewKubeJobProvider("ci", client, client).WithRuntimeData(secrets, nil, 1024)
	_, err := p.Submit(context.Background(), ProviderBuildRequest{BuildRunID: 1, IdempotencyKey: "build:1:submit", RepositoryURL: "https://git.example/test.git", BuilderImage: "builder:v1", BuildCommand: "build", GitCredential: "test-token"})
	var uncertain *ExecutionUncertainError
	if !errors.As(err, &uncertain) || secrets.deleted != "" || secrets.data["token"] == "" {
		t.Fatalf("uncertain submission damaged execution: %v", err)
	}
}

func TestFinalizeDefersTTLUntilAcknowledgedAndReleasesRetainedPodSlot(t *testing.T) {
	ctx := context.Background()
	job, err := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 12, IdempotencyKey: "build:12:submit", ExecutionPolicy: ExecutionPolicy{MaxBuildsPerNode: 2, NodeSelector: map[string]string{"axiops.io/ci-pool": "ci"}, WorkspaceSize: "10Gi"}})
	if err != nil {
		t.Fatal(err)
	}
	if job.Spec.TTLSecondsAfterFinished != nil || job.Spec.Template.Spec.NodeSelector["axiops.io/ci-pool"] != "ci" || job.Spec.Template.Spec.Affinity.PodAntiAffinity == nil || job.Spec.Template.Spec.Volumes[0].EmptyDir.SizeLimit.String() != "10Gi" {
		t.Fatal("unsafe scheduling or premature TTL")
	}
	job.Annotations = map[string]string{"axiops.io/retention-seconds": "3600"}
	job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(time.Now().Add(-48 * time.Hour))}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-12", Namespace: "ci", Labels: map[string]string{"job-name": job.Name, "axiops.io/build-slot": "0"}}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}
	client := fake.NewSimpleClientset(job, pod)
	jobs := client.BatchV1().Jobs("ci")
	pods := client.CoreV1().Pods("ci")
	secrets := &recordingBuildSecrets{}
	p := NewKubeJobProvider("ci", jobs, jobs, pods).WithRuntimeData(secrets, nil, 1024)
	if _, err := p.Status(ctx, ProviderExecution{JobName: job.Name}); err != nil {
		t.Fatal(err)
	}
	if secrets.deleted != "" {
		t.Fatal("observation cleaned up before persistence")
	}
	if err := p.Finalize(ctx, ProviderExecution{JobName: job.Name}); err != nil {
		t.Fatal(err)
	}
	saved, _ := jobs.Get(ctx, job.Name, metav1.GetOptions{})
	savedPod, _ := pods.Get(ctx, pod.Name, metav1.GetOptions{})
	if *saved.Spec.TTLSecondsAfterFinished < 48*3600+3598 || savedPod.Labels["axiops.io/build-slot"] != "" || secrets.deleted == "" {
		t.Fatalf("retention ignored offline interval or slot not released: %+v", saved.Spec)
	}
	first := *saved.Spec.TTLSecondsAfterFinished
	if err := p.Finalize(ctx, ProviderExecution{JobName: job.Name}); err != nil {
		t.Fatal(err)
	}
	saved, _ = jobs.Get(ctx, job.Name, metav1.GetOptions{})
	if *saved.Spec.TTLSecondsAfterFinished != first {
		t.Fatal("retry extended retention")
	}
}

type brokenCredentialResolver struct{ fakeResolver }

func (*brokenCredentialResolver) BuildRequest(context.Context, int64) (ProviderBuildRequest, error) {
	return ProviderBuildRequest{}, errors.New("credentials rotated")
}
func TestWorkerAdoptsExecutionBeforeReadingCredentials(t *testing.T) {
	job, _ := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 1, IdempotencyKey: "build:1:submit"})
	client := &fakeJobClient{existing: job}
	providers, _ := NewProviderRegistry(NewKubeJobProvider("ci", client, client))
	store := &fakeCommandStore{command: &BuildCommand{ID: 1, BuildRunID: 1, CommandType: CommandSubmit, Provider: KubeJobProviderName, IdempotencyKey: "build:1:submit", Attempts: 3}}
	resolver := &brokenCredentialResolver{}
	if _, err := NewWorker(store, providers, resolver, 3, time.Second).ProcessOne(context.Background(), "replacement"); err != nil {
		t.Fatal(err)
	}
	if !store.completed || !resolver.submitted || store.failed || client.created != nil {
		t.Fatal("restart did not adopt the original execution")
	}
}

func TestCancelDiscoversUnacknowledgedSubmission(t *testing.T) {
	job, _ := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 1, IdempotencyKey: "build:1:submit"})
	client := &fakeJobClient{existing: job}
	providers, _ := NewProviderRegistry(NewKubeJobProvider("ci", client, client))
	store := &fakeCommandStore{command: &BuildCommand{ID: 2, BuildRunID: 1, CommandType: CommandCancel, Provider: KubeJobProviderName}}
	resolver := &fakeResolver{emptyExecution: true}
	if _, err := NewWorker(store, providers, resolver, 3, time.Second).ProcessOne(context.Background(), "replacement"); err != nil {
		t.Fatal(err)
	}
	if client.deleted != job.Name || !resolver.cancelled || !store.completed {
		t.Fatal("unacknowledged Job escaped cancellation")
	}
}

type terminatingJobClient struct{ fakeJobClient }

func (c *terminatingJobClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	c.deleted = name
	return nil
}
func TestCancellationDoesNotFinishBeforeContainersStop(t *testing.T) {
	client := &terminatingJobClient{fakeJobClient: fakeJobClient{existing: &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "job"}}}}
	secrets := &recordingBuildSecrets{}
	p := NewKubeJobProvider("ci", client, client).WithRuntimeData(secrets, nil, 1024)
	err := p.Cancel(context.Background(), ProviderExecution{JobName: "job"})
	var uncertain *ExecutionUncertainError
	if !errors.As(err, &uncertain) || client.deleted != "job" || secrets.deleted != "" {
		t.Fatalf("cancellation falsely completed: %v", err)
	}
}

func TestCleanupNeverMutatesForeignJobOrCredential(t *testing.T) {
	ctx := context.Background()
	job, _ := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 33, IdempotencyKey: "foreign-request"})
	job.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: buildCredentialSecretName(33), Namespace: "ci", Labels: map[string]string{"axiops.io/component": "build-credential", "axiops.io/idempotency": "foreign-request"}}}
	client := fake.NewSimpleClientset(job, secret)
	jobs := client.BatchV1().Jobs("ci")
	secrets := client.CoreV1().Secrets("ci")
	p := NewKubeJobProvider("ci", jobs, jobs).WithRuntimeData(NewKubernetesSecretStore(secrets), nil, 1024)
	execution := ProviderExecution{BuildRunID: 33, JobName: job.Name}
	if err := p.Cancel(ctx, execution); err == nil {
		t.Fatal("foreign Job cancellation allowed")
	}
	if err := p.Finalize(ctx, execution); err == nil {
		t.Fatal("foreign Job cleanup allowed")
	}
	saved, err := jobs.Get(ctx, job.Name, metav1.GetOptions{})
	if err != nil || saved.Spec.TTLSecondsAfterFinished != nil {
		t.Fatal("foreign Job mutated")
	}
	if err := jobs.Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := p.Finalize(ctx, execution); err == nil {
		t.Fatal("foreign orphan Secret cleanup allowed")
	}
	if _, err := secrets.Get(ctx, secret.Name, metav1.GetOptions{}); err != nil {
		t.Fatal("foreign Secret deleted")
	}
}
