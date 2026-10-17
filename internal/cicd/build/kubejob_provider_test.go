package build

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type fakeJobClient struct {
	created   *batchv1.Job
	existing  *batchv1.Job
	deleted   string
	createErr error
	deleteErr error
}

type logTestPods struct{}

type diagnosticPods struct {
	pod corev1.Pod
	err error
}

func (p diagnosticPods) List(context.Context, metav1.ListOptions) (*corev1.PodList, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &corev1.PodList{Items: []corev1.Pod{p.pod}}, nil
}

func TestUnschedulableBuildHasDiagnosticsWithoutContainerOutput(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{}}
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pending-pod"}, Status: corev1.PodStatus{
		Phase:      corev1.PodPending,
		Conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: "Unschedulable", Message: "0/3 nodes available: Insufficient cpu"}},
	}}
	provider := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).WithRuntimeData(nil, logTestReader{}, 1024)
	state, err := provider.Status(context.Background(), ProviderExecution{JobName: "pending-job"})
	if err != nil || state.Status != StatusRunning || state.LogText != "" || !strings.Contains(state.DiagnosticText, "Insufficient cpu") || !strings.Contains(state.DiagnosticText, "has not started") {
		t.Fatalf("missing truthful pre-start diagnostic: state=%+v err=%v", state, err)
	}
}

func TestObservationFailuresAreVisibleWithoutChangingJobOutcome(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}}}}
	provider := NewKubeJobProvider("ci", client, client, diagnosticPods{err: errors.New("pods is forbidden")})
	state, err := provider.Status(context.Background(), ProviderExecution{JobName: "job"})
	if err != nil || state.Status != StatusSuccess || !strings.Contains(state.DiagnosticText, "pods is forbidden") {
		t.Fatalf("pod query failure hidden or outcome changed: %+v %v", state, err)
	}
	provider = NewKubeJobProvider("ci", client, client, logTestPods{}).WithRuntimeData(nil, logTestReader{}, 1024)
	state, err = provider.Status(context.Background(), ProviderExecution{JobName: "job"})
	if err != nil || !strings.Contains(state.DiagnosticText, "Log collection failed") || !strings.Contains(state.LogText, "拉取代码失败") {
		t.Fatalf("partial log collection failure hidden: %+v %v", state, err)
	}
}

func TestPendingPodDoesNotOverwriteJobDeadline(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "DeadlineExceeded"}}}}}
	pod := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: "Unschedulable"}}}}
	state, err := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).Status(context.Background(), ProviderExecution{JobName: "job"})
	if err != nil || state.Status != StatusTimeout || !strings.Contains(state.ErrorMessage, "DeadlineExceeded") {
		t.Fatalf("deadline lost: %+v %v", state, err)
	}
}

func TestCloneFailureReasonSurvivesPodFailureAndJobBackoff(t *testing.T) {
	pod := corev1.Pod{Status: corev1.PodStatus{
		Phase:                 corev1.PodFailed,
		InitContainerStatuses: []corev1.ContainerStatus{{Name: "git-clone", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 128, Reason: "Error"}}}},
	}}
	for _, jobStatus := range []batchv1.JobStatus{{}, {Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}}} {
		client := &fakeJobClient{existing: &batchv1.Job{Status: jobStatus}}
		state, err := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).Status(context.Background(), ProviderExecution{JobName: "job"})
		if err != nil || !strings.Contains(state.ErrorMessage, "git-clone exited with code 128") || !strings.Contains(state.DiagnosticText, "[git-clone]") {
			t.Fatalf("init container failure lost: %+v %v", state, err)
		}
		if len(jobStatus.Conditions) > 0 && (state.Status != StatusFailed || !strings.Contains(state.ErrorMessage, "BackoffLimitExceeded")) {
			t.Fatalf("job failure must retain container cause: %+v", state)
		}
	}
}

func TestBuilderStartupDiagnosticSurvivesJobBackoff(t *testing.T) {
	message := "AXIOPS_BUILD_ERROR: ROOTLESS_USER_NAMESPACE_DISABLED: user.max_user_namespaces=0"
	pod := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed,
		ContainerStatuses: []corev1.ContainerStatus{{Name: "builder", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error", Message: message}}}},
	}}
	client := &fakeJobClient{existing: &batchv1.Job{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}}}}
	state, err := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).Status(context.Background(), ProviderExecution{JobName: "job"})
	if err != nil || state.Status != StatusFailed || !strings.Contains(state.ErrorMessage, message) || !strings.Contains(state.DiagnosticText, message) {
		t.Fatalf("builder startup cause lost: %+v err=%v", state, err)
	}
}

func (logTestPods) List(context.Context, metav1.ListOptions) (*corev1.PodList, error) {
	return &corev1.PodList{Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pod-8"}}}}, nil
}

type logTestReader struct{}

func (logTestReader) Read(_ context.Context, _ string, container string, _ int64) ([]byte, error) {
	if container == "git-clone" {
		return []byte("拉取代码失败"), nil
	}
	return nil, errors.New("builder not started")
}
func TestCloneFailureLogVisibleBeforeBuilderStarts(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{}}
	p := NewKubeJobProvider("ci", client, client, logTestPods{}).WithRuntimeData(nil, logTestReader{}, 1024)
	state, err := p.Status(context.Background(), ProviderExecution{JobName: "job-8"})
	if err != nil || !strings.Contains(state.LogText, "[git-clone]\n拉取代码失败") {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func (f *fakeJobClient) Create(_ context.Context, job *batchv1.Job, _ metav1.CreateOptions) (*batchv1.Job, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = job.DeepCopy()
	created := job.DeepCopy()
	created.UID = types.UID("uid-1")
	f.existing = created.DeepCopy()
	return created, nil
}

func (f *fakeJobClient) Get(_ context.Context, _ string, _ metav1.GetOptions) (*batchv1.Job, error) {
	if f.existing == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, "job")
	}
	return f.existing.DeepCopy(), nil
}

func (f *fakeJobClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	f.deleted = name
	if f.deleteErr == nil {
		f.existing = nil
	}
	return f.deleteErr
}

func TestKubeJobProviderSubmitIsIdempotent(t *testing.T) {
	req := ProviderBuildRequest{
		BuilderImage: "registry/ci:v1",
		BuildRunID:   9, BuildCommand: "true", RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:9:submit",
	}
	existing, err := BuildKubeJob("axiops-cicd", req)
	if err != nil {
		t.Fatal(err)
	}
	existing.UID = types.UID("existing-uid")
	client := &fakeJobClient{
		createErr: apierrors.NewAlreadyExists(schema.GroupResource{Group: "batch", Resource: "jobs"}, existing.Name),
		existing:  existing,
	}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	execution, err := provider.Submit(context.Background(), req)
	if err != nil || execution.ExternalID != "existing-uid" || execution.JobName != existing.Name {
		t.Fatalf("idempotent Submit() execution=%+v error=%v", execution, err)
	}
}

func TestKubeJobProviderRejectsJobNameCollision(t *testing.T) {
	req := ProviderBuildRequest{
		BuilderImage: "registry/ci:v1",
		BuildRunID:   9, BuildCommand: "true", RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:9:submit",
	}
	existing, err := BuildKubeJob("axiops-cicd", req)
	if err != nil {
		t.Fatal(err)
	}
	existing.Labels["axiops.io/idempotency"] = "another-request"
	client := &fakeJobClient{
		createErr: apierrors.NewAlreadyExists(schema.GroupResource{Group: "batch", Resource: "jobs"}, existing.Name),
		existing:  existing,
	}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	if _, err := provider.Submit(context.Background(), req); err == nil {
		t.Fatal("provider accepted an existing job owned by another request")
	}
}

func TestKubeJobProviderCancelIsIdempotent(t *testing.T) {
	client := &fakeJobClient{deleteErr: apierrors.NewNotFound(schema.GroupResource{Group: "batch", Resource: "jobs"}, "axiops-build-9")}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	if err := provider.Cancel(context.Background(), ProviderExecution{JobName: "axiops-build-9"}); err != nil {
		t.Fatalf("idempotent Cancel() error = %v", err)
	}
}

func TestKubeJobProviderObservesSuccessfulJob(t *testing.T) {
	client := &fakeJobClient{existing: &batchv1.Job{
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: "True"}}},
	}}
	state, err := NewKubeJobProvider("axiops-cicd", client, client).Status(context.Background(), ProviderExecution{JobName: "axiops-build-9"})
	if err != nil || state.Status != StatusSuccess {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func TestProviderResultValidation(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	artifactDigest := strings.Repeat("b", 64)
	if !validProviderResult(digest, 100, "https://artifacts.example.com/app.tgz", artifactDigest, 200) {
		t.Fatal("valid provider result was rejected")
	}
	if validProviderResult("sha256:bad", 100, "https://artifacts.example.com/app.tgz", artifactDigest, 200) ||
		validProviderResult(digest, 100, "file:///tmp/app.tgz", artifactDigest, 200) ||
		validProviderResult(digest, 100, "https://artifacts.example.com/app.tgz", "bad", 200) {
		t.Fatal("invalid provider result was accepted")
	}
}

func TestCIPlatformDefaultsAndIsolatedCache(t *testing.T) {
	client := &fakeJobClient{}
	p := NewKubeJobProvider("ci-only", client, client).WithDefaults(KubeJobDefaults{BuilderImage: "registry/ci:v1", BuildCommand: "build-and-report", CachePVCName: "shared-cache", CacheDir: "/workspace/.cache", JobRetentionSeconds: 86400})
	_, err := p.Submit(context.Background(), ProviderBuildRequest{BuildRunID: 8, ProjectID: 2, ServiceID: 3, RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:8", CacheEnabled: true, CacheStrategy: "inherit"})
	if err != nil {
		t.Fatal(err)
	}
	job := client.created
	if job.Namespace != "ci-only" || job.Spec.Template.Spec.Containers[0].Image != "registry/ci:v1" {
		t.Fatal("CI defaults not applied")
	}
	mount := job.Spec.Template.Spec.Containers[0].VolumeMounts[1]
	if mount.SubPath != "project-2/service-3" {
		t.Fatal("cache not isolated")
	}
	if mount.MountPath != "/workspace/.cache" {
		t.Fatalf("absolute cache path changed: %s", mount.MountPath)
	}
	if job.Spec.Template.Spec.Volumes[0].EmptyDir == nil {
		t.Fatal("source workspace must be private")
	}
	if job.Spec.TTLSecondsAfterFinished != nil || job.Annotations["axiops.io/retention-seconds"] != "86400" {
		t.Fatal("log recovery retention missing")
	}
}

func TestBuildKubeJobManifest(t *testing.T) {
	job, err := BuildKubeJob("", ProviderBuildRequest{
		BuildRunID: 42, ServiceID: 7, ProjectID: 3, Branch: "main", Commit: "abc",
		RepositoryURL: "https://git.example/app.git", BuildCommand: "go test ./...", IdempotencyKey: "build:42:submit",
		DockerfilePath: "docker/Dockerfile", BuildContext: "services/api", ImageRepo: "registry.example/api", ImageURL: "registry.example/api:main-42",
		CacheEnabled: true, CacheDir: " .cache/go ", EnvVars: []EnvVar{{Key: "GOFLAGS", Value: "-mod=readonly"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Namespace != "axiops-cicd" || job.Name != "axiops-build-42" {
		t.Fatalf("unexpected job identity: %s/%s", job.Namespace, job.Name)
	}
	if job.Labels["axiops.io/build-run"] != "42" || job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 0 {
		t.Fatal("build identity or retry policy missing")
	}
	if job.Labels["axiops.io/idempotency"] != "build-42-submit" {
		t.Fatalf("idempotency label was not normalized: %q", job.Labels["axiops.io/idempotency"])
	}
	container := job.Spec.Template.Spec.Containers[0]
	if container.Image == "" || len(container.Args) != 1 || container.Args[0] != "go test ./..." {
		t.Fatal("builder container was not configured")
	}
	if container.WorkingDir != "/workspace" || len(job.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatal("source checkout workspace was not configured")
	}
	if job.Spec.Template.Spec.AutomountServiceAccountToken == nil || *job.Spec.Template.Spec.AutomountServiceAccountToken {
		t.Fatal("build pod must not receive a Kubernetes service account token")
	}
	if len(job.Spec.Template.Spec.Volumes) != 2 {
		t.Fatalf("expected workspace and cache volumes, got %d", len(job.Spec.Template.Spec.Volumes))
	}
	if len(container.VolumeMounts) != 2 || container.VolumeMounts[1].MountPath != "/workspace/.cache/go" {
		t.Fatalf("cache mount not configured: %+v", container.VolumeMounts)
	}
	env := map[string]string{}
	for _, item := range container.Env {
		env[item.Name] = item.Value
	}
	for key, value := range map[string]string{
		"IMAGE_REPO": "registry.example/api", "IMAGE_TAG": "main-42", "DOCKERFILE_PATH": "docker/Dockerfile",
		"BUILD_CONTEXT": "services/api", "CACHE_ENABLED": "true", "CACHE_DIR": "/workspace/.cache/go", "GOFLAGS": "-mod=readonly",
	} {
		if env[key] != value {
			t.Fatalf("builder env %s=%q, want %q", key, env[key], value)
		}
	}
}

func TestBuildKubeJobReferencesCredentialSecretWithoutPlaintext(t *testing.T) {
	job, err := BuildKubeJob("axiops-cicd", ProviderBuildRequest{
		BuildRunID: 42, RepositoryURL: "https://git.example/app.git", GitCredential: "plain-token-must-not-appear",
		BuildCommand: "true", IdempotencyKey: "build:42:submit",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "plain-token-must-not-appear") {
		t.Fatal("git token leaked into Job manifest")
	}
	var tokenEnv *corev1.EnvVar
	for i := range job.Spec.Template.Spec.InitContainers[0].Env {
		if job.Spec.Template.Spec.InitContainers[0].Env[i].Name == "GIT_TOKEN" {
			tokenEnv = &job.Spec.Template.Spec.InitContainers[0].Env[i]
		}
	}
	if tokenEnv == nil || tokenEnv.ValueFrom == nil || tokenEnv.ValueFrom.SecretKeyRef == nil || tokenEnv.ValueFrom.SecretKeyRef.Name != "axiops-build-42-git" {
		t.Fatalf("token env=%+v", tokenEnv)
	}
}

func TestKubeJobProviderSubmitCancel(t *testing.T) {
	client := &fakeJobClient{}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	execution, err := provider.Submit(context.Background(), ProviderBuildRequest{
		BuilderImage: "registry/ci:v1",
		BuildRunID:   9, BuildCommand: "true", RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:9:submit",
	})
	if err != nil || execution.JobName != "axiops-build-9" || client.created == nil {
		t.Fatalf("Submit() execution=%+v error=%v", execution, err)
	}
	if err := provider.Cancel(context.Background(), execution); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if client.deleted != "axiops-build-9" {
		t.Fatalf("deleted job = %q", client.deleted)
	}
}

func TestKubeJobProviderRejectsCredentialsInRepositoryURL(t *testing.T) {
	client := &fakeJobClient{}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	_, err := provider.Submit(context.Background(), ProviderBuildRequest{
		BuilderImage: "registry/ci:v1",
		BuildRunID:   9, BuildCommand: "true", RepositoryURL: "https://token@git.example/app.git", IdempotencyKey: "build:9:submit",
	})
	if err == nil || client.created != nil {
		t.Fatal("provider accepted repository credentials embedded in the URL")
	}
}

func TestKubeJobProviderRejectsLocalRepositoryURL(t *testing.T) {
	client := &fakeJobClient{}
	provider := NewKubeJobProvider("axiops-cicd", client, client)
	_, err := provider.Submit(context.Background(), ProviderBuildRequest{
		BuilderImage: "registry/ci:v1",
		BuildRunID:   9, BuildCommand: "true", RepositoryURL: "file:///etc", IdempotencyKey: "build:9:submit",
	})
	if err == nil || client.created != nil {
		t.Fatal("provider accepted a local file repository URL")
	}
}

func TestKubeJobRejectsMissingToolchainBeforeCreatingResources(t *testing.T) {
	for _, defaults := range []KubeJobDefaults{{}, {BuilderImage: "registry/ci:v1"}, {BuildCommand: "build-and-report"}} {
		client := &fakeJobClient{}
		provider := NewKubeJobProvider("ci", client, client).WithDefaults(defaults)
		_, err := provider.Submit(context.Background(), ProviderBuildRequest{BuildRunID: 9, RepositoryURL: "https://git.example/app.git", IdempotencyKey: "build:9"})
		if err == nil || !strings.Contains(err.Error(), "Build.Defaults") || client.created != nil {
			t.Fatalf("missing toolchain should fail before Job creation: %v", err)
		}
	}
}

func TestCachePathsStayWithinWorkspaceAndAreIdempotent(t *testing.T) {
	for _, raw := range []string{"", ".cache", "/workspace/.cache", "/workspace", "../../cache", `cache\maven`} {
		got := normalizeCachePath(raw)
		if !strings.HasPrefix(got, "/workspace/") || normalizeCachePath(got) != got {
			t.Fatalf("cache path %q normalized to unstable or unsafe path %q", raw, got)
		}
	}
}
