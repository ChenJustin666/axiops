package build

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const KubeJobProviderName = "kube_job"

const DefaultJarCompileImage = "maven:3.9-eclipse-temurin-17"
const DefaultJarCompileCommand = `mvn -B -ntp -Dmaven.repo.local="${CACHE_DIR:-/workspace/.cache/maven}" clean package`

//go:embed git_clone.sh
var gitCloneScript string

//go:embed git_changes.sh
var gitChangesScript string

// JobCreator and JobDeleter keep the provider independent from a concrete
// clientset, making manifest and retry behavior unit-testable.
type JobCreator interface {
	Create(context.Context, *batchv1.Job, metav1.CreateOptions) (*batchv1.Job, error)
}

type JobDeleter interface {
	Delete(context.Context, string, metav1.DeleteOptions) error
}

type JobGetter interface {
	Get(context.Context, string, metav1.GetOptions) (*batchv1.Job, error)
}

type PodLister interface {
	List(context.Context, metav1.ListOptions) (*corev1.PodList, error)
}

type BuildCredentialSecretStore interface {
	Ensure(context.Context, string, string, map[string]string) error
	Delete(context.Context, string) error
}

type PodLogReader interface {
	Read(context.Context, string, string, int64) ([]byte, error)
}

type KubeJobProvider struct {
	defaults  KubeJobDefaults
	namespace string
	creator   JobCreator
	deleter   JobDeleter
	pods      PodLister
	secrets   BuildCredentialSecretStore
	logs      PodLogReader
	logLimit  int64
}

type KubeJobDefaults struct {
	ExecutionPolicy     ExecutionPolicy
	BuildCommand        string
	TimeoutSeconds      int
	ResourceCPU         string
	ResourceMemory      string
	BuilderImage        string
	CachePVCName        string
	CacheDir            string
	JobRetentionSeconds int32
}

func (p *KubeJobProvider) WithDefaults(defaults KubeJobDefaults) *KubeJobProvider {
	p.defaults = defaults
	return p
}

func (p *KubeJobProvider) WithExecutionPolicy(policy ExecutionPolicy) *KubeJobProvider {
	p.defaults.ExecutionPolicy = policy
	return p
}

var imageDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var artifactSHA256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func NewKubeJobProvider(namespace string, creator JobCreator, deleter JobDeleter, podLister ...PodLister) *KubeJobProvider {
	provider := &KubeJobProvider{namespace: namespace, creator: creator, deleter: deleter, logLimit: 1 << 20}
	if len(podLister) > 0 {
		provider.pods = podLister[0]
	}
	return provider
}

func (p *KubeJobProvider) WithRuntimeData(secrets BuildCredentialSecretStore, logs PodLogReader, logLimit int64) *KubeJobProvider {
	p.secrets, p.logs = secrets, logs
	if logLimit > 0 {
		p.logLimit = logLimit
	}
	return p
}

func (p *KubeJobProvider) Name() string { return KubeJobProviderName }

func (p *KubeJobProvider) Submit(ctx context.Context, req ProviderBuildRequest) (ProviderExecution, error) {
	if execution, found, err := p.Recover(ctx, req.BuildRunID, req.IdempotencyKey); err != nil || found {
		return execution, err
	}
	req.ExecutionPolicy = p.defaults.ExecutionPolicy
	if req.BuildMode == "jar_image" {
		// Saved build_image/build_command belong to the Java compilation step.
		req.CompileImage = defaultString(req.BuilderImage, DefaultJarCompileImage)
		req.CompileCommand = defaultString(req.BuildCommand, DefaultJarCompileCommand)
		req.BuilderImage, req.BuildCommand = p.defaults.BuilderImage, p.defaults.BuildCommand
	}
	if strings.TrimSpace(req.BuildCommand) == "" {
		req.BuildCommand = p.defaults.BuildCommand
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = p.defaults.TimeoutSeconds
	}
	req.ResourceCPU = strings.TrimSpace(defaultString(req.ResourceCPU, p.defaults.ResourceCPU))
	req.ResourceMemory = strings.TrimSpace(defaultString(req.ResourceMemory, p.defaults.ResourceMemory))
	if strings.TrimSpace(req.BuilderImage) == "" {
		req.BuilderImage = p.defaults.BuilderImage
	}
	if strings.TrimSpace(req.BuilderImage) == "" || strings.TrimSpace(req.BuildCommand) == "" {
		return ProviderExecution{}, fmt.Errorf("configure a builder image and build command in the service or platform Build.Defaults before running a build")
	}
	if req.CacheEnabled && strings.TrimSpace(req.CacheDir) == "" {
		req.CacheDir = p.defaults.CacheDir
	}
	if req.CacheEnabled && (req.CacheStrategy == "inherit" || req.CacheStrategy == "") {
		req.CacheStrategy = "emptydir"
		if p.defaults.CachePVCName != "" {
			req.CacheStrategy = "pvc"
			req.CachePVCName = p.defaults.CachePVCName
		}
	}
	if req.CacheEnabled && req.CacheStrategy == "pvc" && req.CachePVCName == "" {
		return ProviderExecution{}, fmt.Errorf("PVC cache requires a configured claim in the CI namespace")
	}
	if p.creator == nil {
		return ProviderExecution{}, fmt.Errorf("kube job provider is not configured")
	}
	if req.BuildRunID <= 0 || strings.TrimSpace(req.IdempotencyKey) == "" {
		return ProviderExecution{}, fmt.Errorf("build run id and idempotency key are required")
	}
	if err := validateRepositoryURL(req.RepositoryURL); err != nil {
		return ProviderExecution{}, err
	}
	job, err := BuildKubeJob(p.namespace, req)
	if err != nil {
		return ProviderExecution{}, err
	}
	secretName := buildCredentialSecretName(req.BuildRunID)
	credentialData := map[string]string{}
	if req.GitCredential != "" {
		credentialData["token"] = req.GitCredential
	}
	if req.RegistryDockerConfig != "" {
		credentialData[corev1.DockerConfigJsonKey] = req.RegistryDockerConfig
	}
	if len(credentialData) > 0 {
		if p.secrets == nil {
			return ProviderExecution{}, fmt.Errorf("build credential secret store is not configured")
		}
		if err := p.secrets.Ensure(ctx, secretName, req.IdempotencyKey, credentialData); err != nil {
			return ProviderExecution{}, fmt.Errorf("prepare build credential: %w", err)
		}
	}
	job.Annotations = map[string]string{"axiops.io/retention-seconds": strconv.Itoa(int(p.defaults.JobRetentionSeconds)), "axiops.io/archive-before-cleanup": "true"}
	created, err := p.creator.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			// A timeout can arrive after the API server committed the Job. Keep
			// its Secret and recover the same identity; never submit a new run.
			if !apierrors.IsInvalid(err) && !apierrors.IsForbidden(err) && !apierrors.IsUnauthorized(err) && !apierrors.IsBadRequest(err) {
				return ProviderExecution{}, &ExecutionUncertainError{Cause: err}
			}
			if len(credentialData) > 0 {
				_ = p.deleteCredentials(ctx, secretName, req.IdempotencyKey)
			}
			return ProviderExecution{}, err
		}
		getter, ok := p.creator.(JobGetter)
		if !ok {
			return ProviderExecution{}, fmt.Errorf("get existing kube job after create conflict: %w", err)
		}
		created, err = getter.Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			return ProviderExecution{}, fmt.Errorf("get existing kube job: %w", err)
		}
		if created.Labels["axiops.io/build-run"] != strconv.FormatInt(req.BuildRunID, 10) ||
			created.Labels["axiops.io/idempotency"] != labelValue(req.IdempotencyKey) {
			return ProviderExecution{}, fmt.Errorf("existing kube job %q belongs to another build request", job.Name)
		}
	}
	execution := ProviderExecution{BuildRunID: req.BuildRunID, ExternalID: created.Name, JobName: created.Name}
	if created.UID != "" {
		execution.ExternalID = string(created.UID)
	}
	return execution, nil
}

func (p *KubeJobProvider) Cancel(ctx context.Context, execution ProviderExecution) error {
	if p.deleter == nil {
		return fmt.Errorf("kube job provider is not configured")
	}
	if strings.TrimSpace(execution.JobName) == "" {
		return fmt.Errorf("kube job name is required for cancellation")
	}
	options := metav1.DeleteOptions{PropagationPolicy: func() *metav1.DeletionPropagation {
		policy := metav1.DeletePropagationForeground
		return &policy
	}()}
	if execution.BuildRunID > 0 {
		if getter, ok := p.creator.(JobGetter); ok {
			job, err := getter.Get(ctx, execution.JobName, metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			if apierrors.IsNotFound(err) {
				if p.secrets != nil {
					return p.deleteCredentials(ctx, buildCredentialSecretNameFromJob(execution.JobName), executionKey(execution))
				}
				return nil
			}
			if err == nil {
				if err = validateExecutionOwner(job, execution.BuildRunID); err != nil {
					return err
				}
				if job.UID != "" {
					uid := job.UID
					options.Preconditions = &metav1.Preconditions{UID: &uid}
				}
			}
		}
	}
	err := p.deleter.Delete(ctx, execution.JobName, options)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if getter, ok := p.creator.(JobGetter); ok {
		_, getErr := getter.Get(ctx, execution.JobName, metav1.GetOptions{})
		if getErr == nil {
			return &ExecutionUncertainError{Cause: errors.New("cancellation requested; waiting for Job and containers to stop")}
		}
		if !apierrors.IsNotFound(getErr) {
			return &ExecutionUncertainError{Cause: getErr}
		}
	}
	if p.secrets != nil {
		return p.deleteCredentials(ctx, buildCredentialSecretNameFromJob(execution.JobName), executionKey(execution))
	}
	return nil
}

func (p *KubeJobProvider) Status(ctx context.Context, execution ProviderExecution) (ProviderState, error) {
	getter, ok := p.creator.(JobGetter)
	if !ok || strings.TrimSpace(execution.JobName) == "" {
		return ProviderState{}, fmt.Errorf("kube job status provider is not configured")
	}
	job, err := getter.Get(ctx, execution.JobName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ProviderState{}, fmt.Errorf("%w: %s; Job was removed before result collection", ErrExecutionMissing, execution.JobName)
		}
		return ProviderState{}, err
	}
	state := ProviderState{Status: StatusRunning}
	var observedPod *corev1.Pod
	for _, condition := range job.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			continue
		}
		switch condition.Type {
		case batchv1.JobComplete:
			state.Status = StatusSuccess
		case batchv1.JobFailed:
			state.Status = StatusFailed
			if condition.Reason == "DeadlineExceeded" {
				state.Status = StatusTimeout
			}
			state.ErrorMessage = strings.TrimSpace(condition.Reason + ": " + condition.Message)
		}
	}
	if p.pods != nil {
		pods, err := p.pods.List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + execution.JobName})
		// Pod metadata enriches logs but must not block a terminal Job state.
		if err != nil {
			state.LogsIncomplete = true
			state.DiagnosticText += "[system] Pod status query failed: " + err.Error() + "\n"
		} else if pods == nil || len(pods.Items) == 0 {
			state.DiagnosticText += "[system] No Pod found for Job " + execution.JobName + "; container logs are not available.\n"
		} else {
			observedPod = &pods.Items[0]
			state.PodName = pods.Items[0].Name
			state.NodeName = pods.Items[0].Spec.NodeName
			state.DiagnosticText += fmt.Sprintf("[system] Job=%s Pod=%s phase=%s node=%s\n", execution.JobName, state.PodName, observedPod.Status.Phase, state.NodeName)
			if reason := strings.TrimSpace(observedPod.Status.Reason + " " + observedPod.Status.Message); reason != "" {
				state.DiagnosticText += "[system] Pod reason: " + reason + "\n"
				if observedPod.Status.Phase == corev1.PodFailed {
					if state.ErrorMessage != "" {
						state.ErrorMessage += "; "
					}
					state.ErrorMessage += "Pod: " + reason
				}
			}
			if pods.Items[0].Status.Phase == corev1.PodPending && !IsTerminalStatus(state.Status) {
				for _, condition := range pods.Items[0].Status.Conditions {
					if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
						state.ErrorMessage = "waiting for cluster resources: " + strings.TrimSpace(condition.Reason+" "+condition.Message)
						break
					}
				}
				if state.ErrorMessage == "" {
					statuses := append(append([]corev1.ContainerStatus{}, pods.Items[0].Status.InitContainerStatuses...), pods.Items[0].Status.ContainerStatuses...)
					for _, status := range statuses {
						if waiting := status.State.Waiting; waiting != nil && waiting.Reason != "PodInitializing" && waiting.Reason != "ContainerCreating" {
							state.ErrorMessage = status.Name + ": " + strings.TrimSpace(waiting.Reason+" "+waiting.Message)
							break
						}
					}
				}
			}
			// Init-container failures outlive Pending. Preserve the failing stage
			// even after the Job controller adds a generic BackoffLimitExceeded.
			statuses := append(append([]corev1.ContainerStatus{}, observedPod.Status.InitContainerStatuses...), observedPod.Status.ContainerStatuses...)
			for _, status := range statuses {
				if status.Name == "git-clone" && status.State.Terminated != nil && status.State.Terminated.ExitCode == 0 {
					commit := strings.TrimSpace(status.State.Terminated.Message)
					if validSourceCommit(commit) {
						state.SourceCommit = commit
					}
				}
				if stopped := status.State.Terminated; stopped != nil && stopped.ExitCode != 0 {
					reason := fmt.Sprintf("%s exited with code %d: %s (see [%s] container logs)", status.Name, stopped.ExitCode, stopped.Reason, status.Name)
					if message := strings.TrimSpace(stopped.Message); strings.HasPrefix(message, "AXIOPS_BUILD_ERROR:") {
						if len(message) > 2048 {
							message = message[:2048]
						}
						reason += "; " + message
					}
					if state.ErrorMessage == "" {
						state.ErrorMessage = reason
					} else {
						state.ErrorMessage += "; " + reason
					}
				}
			}
			if job.Labels["axiops.io/execution-kind"] != "test" {
				state.readTerminationResult(pods.Items[0])
			}
		}
	}
	if state.PodName != "" && p.logs != nil {
		containers := []string{"git-clone", "builder"}
		if observedPod != nil {
			for _, container := range observedPod.Spec.InitContainers {
				if container.Name == "jar-build" {
					containers = []string{"git-clone", "jar-build", "builder"}
					break
				}
			}
		}
		containerLogLimit := p.logLimit / int64(len(containers))
		for _, container := range containers {
			// Waiting containers have no stdout yet. Describe their actual state
			// instead of issuing log requests that Kubernetes must reject.
			if observedPod != nil {
				statuses := append(append([]corev1.ContainerStatus{}, observedPod.Status.InitContainerStatuses...), observedPod.Status.ContainerStatuses...)
				var status *corev1.ContainerStatus
				for i := range statuses {
					if statuses[i].Name == container {
						status = &statuses[i]
						break
					}
				}
				if status != nil && status.State.Waiting != nil {
					waiting := status.State.Waiting
					state.DiagnosticText += fmt.Sprintf("[system] %s waiting: %s %s\n", container, waiting.Reason, waiting.Message)
					continue
				}
				if status == nil && observedPod.Status.Phase == corev1.PodPending {
					state.DiagnosticText += "[system] " + container + " has not started; no container output yet.\n"
					continue
				}
			}
			content, err := p.logs.Read(ctx, state.PodName, container, containerLogLimit)
			if err != nil {
				state.LogsIncomplete = true
				state.DiagnosticText += fmt.Sprintf("[system] Log collection failed for %s/%s: %s\n", state.PodName, container, err)
			} else if len(content) > 0 {
				state.LogTruncated = state.LogTruncated || int64(len(content)) >= containerLogLimit
				if container == "git-clone" {
					state.GitChanges = parseGitChanges(content)
					if state.GitChanges != nil && (state.SourceCommit == "" || state.GitChanges.Head != state.SourceCommit) {
						state.GitChanges = nil
					}
					// Machine-readable evidence is shown in the change panel, not as base64 in logs.
					var lines []string
					for _, line := range strings.Split(string(content), "\n") {
						if !strings.HasPrefix(line, gitChangesPrefix) {
							lines = append(lines, line)
						}
					}
					content = []byte(strings.Join(lines, "\n"))
				}
				state.LogText += "[" + container + "]\n" + strings.ToValidUTF8(string(content), "\uFFFD") + "\n"
			}
		}
	}
	if state.ErrorMessage != "" {
		state.DiagnosticText += "[system] " + state.ErrorMessage + "\n"
	}
	if state.LogsIncomplete {
		state.LogTruncated = true
	}
	return state, nil
}

func (p *KubeJobProvider) Recover(ctx context.Context, id int64, key string) (ProviderExecution, bool, error) {
	getter, ok := p.creator.(JobGetter)
	if !ok {
		return ProviderExecution{}, false, nil
	}
	job, err := getter.Get(ctx, fmt.Sprintf("axiops-build-%d", id), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return ProviderExecution{}, false, nil
	}
	if err != nil {
		return ProviderExecution{}, false, &ExecutionUncertainError{Cause: err}
	}
	if job.Labels["axiops.io/build-run"] != strconv.FormatInt(id, 10) || job.Labels["axiops.io/idempotency"] != labelValue(key) {
		return ProviderExecution{}, false, fmt.Errorf("existing Job %q belongs to another build request", job.Name)
	}
	return ProviderExecution{BuildRunID: id, JobName: job.Name, ExternalID: string(job.UID)}, true, nil
}

func (p *KubeJobProvider) Finalize(ctx context.Context, execution ProviderExecution) error {
	if execution.JobName == "" && execution.BuildRunID > 0 {
		execution.JobName = fmt.Sprintf("axiops-build-%d", execution.BuildRunID)
	}
	getter, ok := p.creator.(JobGetter)
	if !ok {
		return errors.New("Job finalization requires a lookup client")
	}
	job, err := getter.Get(ctx, execution.JobName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		if execution.BuildRunID > 0 {
			if err := validateExecutionOwner(job, execution.BuildRunID); err != nil {
				return err
			}
		}
		if job.Labels["axiops.io/component"] != "build" {
			return errors.New("refusing to finalize an unmanaged Job")
		}
		terminal := false
		for _, c := range job.Status.Conditions {
			if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
				terminal = true
			}
		}
		if !terminal {
			return errors.New("execution is still active; cleanup deferred")
		}
		retention := int64(p.defaults.JobRetentionSeconds)
		if value, parseErr := strconv.ParseInt(job.Annotations["axiops.io/retention-seconds"], 10, 32); parseErr == nil {
			retention = value
		}
		if retention == 0 {
			// Logs/results are already durable. Delete explicitly and confirm
			// foreground GC, rather than depending on a TTL controller existing.
			return p.Cancel(ctx, execution)
		}
		// Retained Pods are evidence, not occupied build slots. Explicitly
		// release the scheduling label after durable completion.
		if pods, ok := p.pods.(interface {
			List(context.Context, metav1.ListOptions) (*corev1.PodList, error)
			Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.Pod, error)
		}); ok {
			list, listErr := pods.List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + job.Name})
			if listErr != nil {
				return listErr
			}
			for _, pod := range list.Items {
				if _, has := pod.Labels["axiops.io/build-slot"]; has {
					if _, patchErr := pods.Patch(ctx, pod.Name, types.MergePatchType, []byte(`{"metadata":{"labels":{"axiops.io/build-slot":null}}}`), metav1.PatchOptions{}); patchErr != nil && !apierrors.IsNotFound(patchErr) {
						return patchErr
					}
				}
			}
		}
		patcher, ok := p.creator.(interface {
			Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*batchv1.Job, error)
		})
		if !ok {
			return errors.New("Job finalization requires a patch client")
		}
		ack, parseErr := time.Parse(time.RFC3339, job.Annotations["axiops.io/archive-ack-at"])
		if parseErr != nil {
			ack = time.Now().UTC().Truncate(time.Second)
		}
		finished := job.CreationTimestamp.Time
		for _, c := range job.Status.Conditions {
			if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
				finished = c.LastTransitionTime.Time
			}
		}
		if job.Status.CompletionTime != nil {
			finished = job.Status.CompletionTime.Time
		}
		if finished.IsZero() {
			finished = ack
		}
		finished = finished.UTC().Truncate(time.Second)
		ttl := retention + int64(ack.Sub(finished).Seconds())
		if ttl < 0 {
			ttl = 0
		}
		if ttl > 2147483647 {
			ttl = 2147483647
		}
		body, _ := json.Marshal(map[string]interface{}{"metadata": map[string]interface{}{"resourceVersion": job.ResourceVersion, "annotations": map[string]string{"axiops.io/archive-ack-at": ack.Format(time.RFC3339)}}, "spec": map[string]int64{"ttlSecondsAfterFinished": ttl}})
		if _, err = patcher.Patch(ctx, job.Name, types.MergePatchType, body, metav1.PatchOptions{}); err != nil {
			return err
		}
	}
	if p.secrets != nil {
		return p.deleteCredentials(ctx, buildCredentialSecretNameFromJob(execution.JobName), executionKey(execution))
	}
	return nil
}

func validateExecutionOwner(job *batchv1.Job, id int64) error {
	if job.Labels["axiops.io/build-run"] != strconv.FormatInt(id, 10) || job.Labels["axiops.io/idempotency"] != labelValue(fmt.Sprintf("build:%d:submit", id)) {
		return fmt.Errorf("refusing to mutate Job %q owned by another build request", job.Name)
	}
	return nil
}

func executionKey(execution ProviderExecution) string {
	id := execution.BuildRunID
	if id <= 0 {
		id, _ = strconv.ParseInt(strings.TrimPrefix(execution.JobName, "axiops-build-"), 10, 64)
	}
	return fmt.Sprintf("build:%d:submit", id)
}

func (p *KubeJobProvider) deleteCredentials(ctx context.Context, name, key string) error {
	if owner, ok := p.secrets.(interface {
		DeleteOwned(context.Context, string, string) error
	}); ok {
		return owner.DeleteOwned(ctx, name, key)
	}
	return p.secrets.Delete(ctx, name)
}

func (s *ProviderState) readTerminationResult(pod corev1.Pod) {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != "builder" || status.State.Terminated == nil || strings.TrimSpace(status.State.Terminated.Message) == "" {
			continue
		}
		var result BuildResult
		if json.Unmarshal([]byte(status.State.Terminated.Message), &result) == nil &&
			(result.Schema == "" || result.Schema == "axiops.build/v1") &&
			validProviderResult(result.ImageDigest, result.ImageSize, result.ArtifactURL, result.ArtifactSHA256, result.ArtifactSize) {
			s.ImageDigest, s.ImageSize = result.ImageDigest, result.ImageSize
			s.ArtifactURL, s.ArtifactSHA256, s.ArtifactSize = result.ArtifactURL, result.ArtifactSHA256, result.ArtifactSize
		}
	}
}

func validProviderResult(imageDigest string, imageSize int64, artifactURL, artifactSHA256 string, artifactSize int64) bool {
	if imageSize < 0 || artifactSize < 0 || (imageDigest != "" && !imageDigestPattern.MatchString(imageDigest)) {
		return false
	}
	if artifactURL == "" {
		return artifactSHA256 == ""
	}
	if !artifactSHA256Pattern.MatchString(artifactSHA256) {
		return false
	}
	parsed, err := url.Parse(artifactURL)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	switch parsed.Scheme {
	case "https", "s3", "oss", "cos":
		return true
	default:
		return false
	}
}

func BuildKubeJob(namespace string, req ProviderBuildRequest) (*batchv1.Job, error) {
	if err := req.ExecutionPolicy.Validate(); err != nil {
		return nil, err
	}
	if req.BuildMode == "jar_image" && (strings.TrimSpace(req.CompileImage) == "" || strings.TrimSpace(req.CompileCommand) == "") {
		return nil, fmt.Errorf("JAR image build requires a resolved Java compilation image and command")
	}
	name := fmt.Sprintf("axiops-build-%d", req.BuildRunID)
	if namespace == "" {
		namespace = "axiops-cicd"
	}
	builderImage := req.BuilderImage
	if builderImage == "" {
		builderImage = "axiops/builder:stable"
	}
	command := req.BuildCommand
	if strings.TrimSpace(command) == "" {
		command = "echo 'build command is not configured' && exit 1"
	}
	deadline := int64(req.TimeoutSeconds)
	if deadline <= 0 {
		deadline = 1800
	}
	backoff := int32(0)
	runID := strconv.FormatInt(req.BuildRunID, 10)
	serviceID := strconv.FormatInt(req.ServiceID, 10)
	projectID := strconv.FormatInt(req.ProjectID, 10)
	automountToken := false
	enableServiceLinks := false
	resources, err := buildResources(req.ResourceCPU, req.ResourceMemory)
	if err != nil {
		return nil, err
	}
	workspaceMount := corev1.VolumeMount{Name: "workspace", MountPath: "/workspace"}
	volumeMounts := []corev1.VolumeMount{workspaceMount}
	volumes := []corev1.Volume{{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	cacheStrategy := normalizeCacheStrategy(req.CacheStrategy)
	if req.CacheEnabled && cacheStrategy == "pvc" && strings.TrimSpace(req.CachePVCName) == "" {
		// A PVC must be explicitly provisioned and selected. Falling back to an
		// ephemeral cache is safer than mounting an arbitrary claim.
		cacheStrategy = "emptydir"
	}
	if req.CacheEnabled && cacheStrategy == "emptydir" {
		cachePath := normalizeCachePath(req.CacheDir)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "build-cache", MountPath: cachePath})
		volumes = append(volumes, corev1.Volume{Name: "build-cache", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
	} else if req.CacheEnabled && cacheStrategy == "pvc" {
		cachePath := normalizeCachePath(req.CacheDir)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "build-cache", MountPath: cachePath, SubPath: fmt.Sprintf("project-%d/service-%d", req.ProjectID, req.ServiceID)})
		volumes = append(volumes, corev1.Volume{Name: "build-cache", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: strings.TrimSpace(req.CachePVCName)}}})
	}
	gitEnv := []corev1.EnvVar{
		{Name: "GIT_URL", Value: req.RepositoryURL},
		{Name: "GIT_BRANCH", Value: req.Branch},
		{Name: "GIT_COMMIT", Value: req.Commit},
		{Name: "GIT_CHANGE_BASE", Value: req.ChangeBaseCommit},
	}
	if req.GitCredential != "" {
		gitEnv = append(gitEnv, corev1.EnvVar{Name: "GIT_TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: buildCredentialSecretName(req.BuildRunID)}, Key: "token",
		}}})
	}

	imageBuildMode := defaultString(req.BuildMode, "dockerfile")
	if req.BuildMode == "jar_image" {
		// Compilation has already completed before the platform builder starts.
		imageBuildMode = "dockerfile"
	}
	builderEnv := []corev1.EnvVar{
		{Name: "GIT_BRANCH", Value: req.Branch},
		{Name: "GIT_COMMIT", Value: req.Commit},
		{Name: "IMAGE_URL", Value: req.ImageURL},
		{Name: "IMAGE_TAG", Value: imageTag(req.ImageURL)},
		{Name: "IMAGE_REPO", Value: req.ImageRepo},
		{Name: "DOCKERFILE_PATH", Value: defaultString(req.DockerfilePath, "Dockerfile")},
		{Name: "BUILD_CONTEXT", Value: defaultString(req.BuildContext, ".")},
		{Name: "CACHE_ENABLED", Value: strconv.FormatBool(req.CacheEnabled)},
		{Name: "CACHE_DIR", Value: normalizeCachePath(req.CacheDir)},
		{Name: "CACHE_STRATEGY", Value: cacheStrategy},
		{Name: "CACHE_PVC_NAME", Value: strings.TrimSpace(req.CachePVCName)},
		{Name: "AXIOPS_RESULT_FILE", Value: "/dev/termination-log"},
		{Name: "AXIOPS_BUILD_MODE", Value: imageBuildMode},
		{Name: "AXIOPS_RESULT_SCHEMA", Value: "axiops.build/v1"},
		{Name: "LANG", Value: "C.UTF-8"},
		{Name: "LC_ALL", Value: "C.UTF-8"},
	}
	// User-defined variables are injected into the builder container while
	// platform variables above remain authoritative when names collide.
	var imagePullSecrets []corev1.LocalObjectReference
	if req.RegistryDockerConfig != "" {
		secretName := buildCredentialSecretName(req.BuildRunID)
		builderEnv = append(builderEnv, corev1.EnvVar{Name: "DOCKER_CONFIG", Value: "/run/axiops-registry"})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "registry-auth", MountPath: "/run/axiops-registry", ReadOnly: true})
		volumes = append(volumes, corev1.Volume{Name: "registry-auth", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
			SecretName: secretName, Items: []corev1.KeyToPath{{Key: corev1.DockerConfigJsonKey, Path: "config.json"}},
		}}})
		imagePullSecrets = []corev1.LocalObjectReference{{Name: secretName}}
	}
	customEnv := make([]corev1.EnvVar, 0, len(req.EnvVars))
	reserved := make(map[string]struct{}, len(builderEnv))
	for _, env := range builderEnv {
		reserved[env.Name] = struct{}{}
	}
	for _, env := range req.EnvVars {
		name := strings.TrimSpace(env.Key)
		if name == "" {
			continue
		}
		if _, exists := reserved[name]; exists {
			continue
		}
		customEnv = append(customEnv, corev1.EnvVar{Name: name, Value: env.Value})
	}
	builderEnv = append(customEnv, builderEnv...)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"axiops.io/component":   "build",
				"axiops.io/build-run":   runID,
				"axiops.io/service":     serviceID,
				"axiops.io/project":     projectID,
				"axiops.io/provider":    KubeJobProviderName,
				"axiops.io/idempotency": labelValue(req.IdempotencyKey),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoff,
			ActiveDeadlineSeconds: &deadline,
			// No TTL until logs and terminal state have been acknowledged in SQL.
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"axiops.io/component": "build",
					"axiops.io/build-run": runID,
				}},
				Spec: corev1.PodSpec{
					NodeSelector:                 req.ExecutionPolicy.NodeSelector,
					Tolerations:                  req.ExecutionPolicy.kubeTolerations(),
					ImagePullSecrets:             imagePullSecrets,
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: &automountToken,
					EnableServiceLinks:           &enableServiceLinks,
					InitContainers: []corev1.Container{{
						Name:         "git-clone",
						Image:        "alpine/git:v2.47.2",
						Command:      []string{"/bin/sh", "-c"},
						Args:         []string{gitCloneScript + "\n(" + gitChangesScript + ") || echo '[changes] Git history is unavailable; source checkout remains valid.'"},
						Env:          gitEnv,
						Resources:    resources,
						VolumeMounts: []corev1.VolumeMount{workspaceMount},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: func() *bool { value := false; return &value }(),
							SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
						},
					}},
					Containers: []corev1.Container{{
						Name:         "builder",
						Image:        builderImage,
						Command:      []string{"/bin/sh", "-c"},
						Args:         []string{command},
						WorkingDir:   "/workspace",
						Env:          builderEnv,
						Resources:    resources,
						VolumeMounts: volumeMounts,
						SecurityContext: &corev1.SecurityContext{
							// Current builder requires a trusted, dedicated CI pool.
							// RootlessKit does not remove this privileged boundary.
							Privileged:     func() *bool { value := true; return &value }(),
							SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
						},
					}},
					Volumes: volumes,
				},
			},
		},
	}
	if req.BuildMode == "test" {
		job.Labels["axiops.io/execution-kind"] = "test"
		job.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: &automountToken,
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		}
	}
	if req.BuildMode == "jar_image" {
		compileMounts := []corev1.VolumeMount{}
		for _, mount := range volumeMounts {
			// Compilation needs source/cache, not registry push credentials.
			if mount.Name != "registry-auth" {
				compileMounts = append(compileMounts, mount)
			}
		}
		compileEnv := append([]corev1.EnvVar{}, customEnv...)
		compileEnv = append(compileEnv,
			corev1.EnvVar{Name: "CACHE_DIR", Value: normalizeCachePath(req.CacheDir)},
			corev1.EnvVar{Name: "BUILD_CONTEXT", Value: defaultString(req.BuildContext, ".")},
			corev1.EnvVar{Name: "GIT_BRANCH", Value: req.Branch},
			corev1.EnvVar{Name: "GIT_COMMIT", Value: req.Commit},
		)
		job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers, corev1.Container{
			Name: "jar-build", Image: req.CompileImage,
			WorkingDir: "/workspace", Command: []string{"/bin/sh", "-c"},
			Args: []string{"set -eu\n" + req.CompileCommand},
			Env:  compileEnv, VolumeMounts: compileMounts, Resources: resources,
			SecurityContext: &corev1.SecurityContext{
				// Checkout is root-owned; both Maven and Gradle need write access.
				RunAsUser:                func() *int64 { value := int64(0); return &value }(),
				AllowPrivilegeEscalation: func() *bool { value := false; return &value }(),
				SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
		})
	}
	if err := mountInlineDockerfile(job, req); err != nil {
		return nil, err
	}
	if slots := req.ExecutionPolicy.MaxBuildsPerNode; slots > 0 {
		slot := strconv.FormatInt(req.BuildRunID%int64(slots), 10)
		job.Spec.Template.Labels["axiops.io/build-slot"] = slot
		job.Spec.Template.Spec.Affinity = &corev1.Affinity{PodAntiAffinity: &corev1.PodAntiAffinity{RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{{TopologyKey: "kubernetes.io/hostname", LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"axiops.io/component": "build", "axiops.io/build-slot": slot}}}}}}
	}
	if size := req.ExecutionPolicy.WorkspaceSize; size != "" {
		quantity := resource.MustParse(size)
		for i := range job.Spec.Template.Spec.Volumes {
			if job.Spec.Template.Spec.Volumes[i].EmptyDir != nil {
				job.Spec.Template.Spec.Volumes[i].EmptyDir.SizeLimit = &quantity
			}
		}
		// Container writable layers, source and emptyDir cache share this cap.
		for i := range job.Spec.Template.Spec.Containers {
			job.Spec.Template.Spec.Containers[i].Resources.Limits[corev1.ResourceEphemeralStorage] = quantity
		}
		for i := range job.Spec.Template.Spec.InitContainers {
			c := &job.Spec.Template.Spec.InitContainers[i]
			if c.Resources.Limits == nil {
				c.Resources.Limits = corev1.ResourceList{}
			}
			c.Resources.Limits[corev1.ResourceEphemeralStorage] = quantity
		}
	}
	return job, nil
}

func imageTag(imageURL string) string {
	value := strings.TrimSpace(imageURL)
	if at := strings.IndexByte(value, '@'); at >= 0 {
		return ""
	}
	lastSlash := strings.LastIndexByte(value, '/')
	colon := strings.LastIndexByte(value, ':')
	if colon <= lastSlash || colon < 0 || colon == len(value)-1 {
		return ""
	}
	return value[colon+1:]
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeCachePath(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return "/workspace/.axiops-cache"
	}
	clean = strings.ReplaceAll(clean, "\\", "/")
	clean = path.Clean(clean)
	clean = strings.TrimPrefix(clean, "/workspace/")
	clean = strings.TrimPrefix(path.Clean("/"+clean), "/")
	if clean == "" || clean == "." || clean == "workspace" {
		clean = ".axiops-cache"
	}
	return "/workspace/" + clean
}

func buildCredentialSecretName(buildRunID int64) string {
	return fmt.Sprintf("axiops-build-%d-git", buildRunID)
}

func buildCredentialSecretNameFromJob(jobName string) string {
	return strings.TrimSpace(jobName) + "-git"
}

func buildResources(cpu, memory string) (corev1.ResourceRequirements, error) {
	resources := corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}}
	for _, item := range []struct {
		name  corev1.ResourceName
		value string
	}{
		{corev1.ResourceCPU, cpu}, {corev1.ResourceMemory, memory},
	} {
		value := strings.TrimSpace(item.value)
		if value == "" {
			// Empty means no platform request or limit for this resource.
			// Kubernetes admission policies can still apply namespace defaults.
			continue
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil || quantity.Sign() <= 0 {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid build %s %q: use a positive Kubernetes quantity or leave empty", item.name, value)
		}
		if item.name == corev1.ResourceMemory && quantity.Cmp(resource.MustParse("1Mi")) < 0 {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid build memory %q: use at least 1Mi, for example 2Gi, or leave empty", value)
		}
		resources.Requests[item.name] = quantity.DeepCopy()
		resources.Limits[item.name] = quantity.DeepCopy()
	}
	return resources, nil
}

func labelValue(value string) string {
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, value)
	value = strings.Trim(value, "-_.")
	if len(value) > 63 {
		value = value[:63]
	}
	value = strings.Trim(value, "-_.")
	if value == "" {
		return "none"
	}
	return value
}

func validateRepositoryURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("repository URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}
	if parsed.User != nil || parsed.RawQuery != "" {
		return fmt.Errorf("repository URL must not contain embedded credentials or query parameters")
	}
	switch parsed.Scheme {
	case "http", "https", "ssh", "git":
		if parsed.Host == "" {
			return fmt.Errorf("repository URL host is required")
		}
	default:
		return fmt.Errorf("repository URL scheme %q is not allowed", parsed.Scheme)
	}
	return nil
}
