package build

import (
	"context"
	"strings"
	"testing"
)

type memoryLogArchive struct{ content []byte }

func (a *memoryLogArchive) Put(_ context.Context, _ string, content []byte) (string, error) {
	a.content = append([]byte(nil), content...)
	return "s3://logs/build.log", nil
}

func (a *memoryLogArchive) Get(context.Context, string) ([]byte, error) {
	return append([]byte(nil), a.content...), nil
}

type fakeRunningBuildStore struct {
	run          *BuildRun
	status       string
	podName      string
	nodeName     string
	logText      string
	logs         map[string]string
	errorMessage string
	digest       string
	verification OutputVerification
}

func TestReconcilerArchivesLogAndKeepsBoundedInlineTail(t *testing.T) {
	longLog := string(make([]byte, 70<<10))
	store := &fakeRunningBuildStore{run: &BuildRun{ID: 31, ProjectID: 7, Provider: "observable", Status: StatusRunning, JobName: "job-31"}}
	providers, _ := NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusRunning, LogText: longLog}})
	archive := &memoryLogArchive{}
	processed, err := NewReconciler(store, providers).WithLogArchive(archive).ReconcileOne(context.Background())
	if err != nil || !processed || len(archive.content) != len(longLog) || len(store.logText) != 64<<10 {
		t.Fatalf("processed=%v archived=%d inline=%d err=%v", processed, len(archive.content), len(store.logText), err)
	}
}

func (s *fakeRunningBuildStore) FindNextRunningRun(context.Context) (*BuildRun, error) {
	return s.run, nil
}
func (s *fakeRunningBuildStore) UpdateRunExecInfo(_ context.Context, _ int64, _, podName, nodeName string) error {
	s.podName, s.nodeName = podName, nodeName
	return nil
}
func (s *fakeRunningBuildStore) TransitionRunStatus(_ context.Context, _ int64, status, reason string) error {
	s.status = status
	s.errorMessage = reason
	return nil
}
func (s *fakeRunningBuildStore) UpdateRunOutputs(_ context.Context, _ int64, _, digest string, _ int64, _, _ string, _ int64) error {
	s.digest = digest
	return nil
}
func (s *fakeRunningBuildStore) UpdateRunVerification(_ context.Context, _ int64, verification OutputVerification) error {
	s.verification = verification
	return nil
}
func (s *fakeRunningBuildStore) CreateLog(_ context.Context, log *BuildLog) (int64, error) {
	if s.logs == nil {
		s.logs = map[string]string{}
	}
	s.logs[log.Stage] = log.LogText
	if log.Stage == "build" {
		s.logText = log.LogText
	}
	return 1, nil
}

func TestReconcilerPersistsSchedulingDiagnosticsAndRetainsWaitReasonOnTimeout(t *testing.T) {
	store := &fakeRunningBuildStore{run: &BuildRun{ID: 31, Provider: "observable", Status: StatusRunning, JobName: "job-31", PodName: "pod-31", ErrorMessage: "Insufficient cpu"}, podName: "pod-31"}
	providers, _ := NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusRunning, PodName: "pod-31", ErrorMessage: "Insufficient cpu"}})
	if _, err := NewReconciler(store, providers).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(store.logs["system"], "Insufficient cpu") || store.logText != "" {
		t.Fatalf("diagnostic missing or mixed into container output: %+v", store.logs)
	}
	providers, _ = NewProviderRegistry(observableProvider{state: ProviderState{Status: StatusTimeout, ErrorMessage: "DeadlineExceeded"}})
	if _, err := NewReconciler(store, providers).ReconcileOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.podName != "pod-31" || store.status != StatusTimeout || !strings.Contains(store.errorMessage, "Insufficient cpu") || !strings.Contains(store.logs["system"], "DeadlineExceeded") {
		t.Fatalf("timeout lost scheduling evidence: %+v", store)
	}
}

type observableProvider struct{ state ProviderState }

type fixedOutputVerifier struct{ result OutputVerification }

func (v fixedOutputVerifier) Verify(context.Context, BuildRun, ProviderState) (OutputVerification, error) {
	return v.result, nil
}

func (p observableProvider) Name() string { return "observable" }
func (p observableProvider) Submit(context.Context, ProviderBuildRequest) (ProviderExecution, error) {
	return ProviderExecution{}, nil
}
func (p observableProvider) Cancel(context.Context, ProviderExecution) error { return nil }
func (p observableProvider) Status(context.Context, ProviderExecution) (ProviderState, error) {
	return p.state, nil
}

func TestReconcilerCompletesObservedBuild(t *testing.T) {
	store := &fakeRunningBuildStore{run: &BuildRun{ID: 31, Provider: "observable", Status: StatusRunning, JobName: "job-31"}}
	providers, _ := NewProviderRegistry(observableProvider{state: ProviderState{
		Status: StatusSuccess, PodName: "pod-31", NodeName: "node-1", LogText: "build ok\n", ImageDigest: "sha256:abc",
	}})
	processed, err := NewReconciler(store, providers).ReconcileOne(context.Background())
	if err != nil || !processed || store.status != StatusSuccess || store.podName != "pod-31" || store.nodeName != "node-1" || store.logText != "build ok\n" || store.digest != "sha256:abc" {
		t.Fatalf("processed=%v status=%q pod=%q node=%q err=%v", processed, store.status, store.podName, store.nodeName, err)
	}
}

func TestReconcilerFailsBuildWhenOutputIntegrityFails(t *testing.T) {
	store := &fakeRunningBuildStore{run: &BuildRun{ID: 41, Provider: "observable", Status: StatusRunning, JobName: "job-41"}}
	providers, _ := NewProviderRegistry(observableProvider{state: ProviderState{
		Status: StatusSuccess, ArtifactURL: "https://artifacts.example/app.tgz", ArtifactSHA256: strings.Repeat("a", 64), ArtifactSize: 10,
	}})
	verification := OutputVerification{ImageStatus: VerificationNotApplicable, ArtifactStatus: VerificationFailed, Error: "artifact checksum mismatch"}
	processed, err := NewReconciler(store, providers).WithOutputVerifier(fixedOutputVerifier{result: verification}).ReconcileOne(context.Background())
	if err != nil || !processed || store.status != StatusFailed || store.verification.ArtifactStatus != VerificationFailed {
		t.Fatalf("processed=%v status=%q verification=%+v err=%v", processed, store.status, store.verification, err)
	}
}
