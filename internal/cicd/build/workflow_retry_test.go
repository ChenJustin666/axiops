package build

import (
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/cicd/workflow"
)

func TestWorkflowRetryFreezesInputAndChangesOutputIdentity(t *testing.T) {
	snapshot := `{"repository_url":"https://git.example/api","config":{"build_image":"builder:1"}}`
	previous := BuildRun{ServiceID: 9, ProjectID: 4, Status: StatusFailed, TriggerType: "workflow", TriggerKey: "workflow:3:build", GitCommit: strings.Repeat("a", 40), GitBranch: "qa", ConfigSnapshot: &snapshot, ImageURL: "registry.example:5000/team/api:old"}
	request := workflow.BuildTriggerRequest{WorkflowRunID: 3, NodeKey: "build", ServiceID: 9, TriggerKey: "workflow:3:build:retry:1"}
	run, err := workflowRetryRun(previous, request, 2, 7)
	if err != nil || run.GitCommit != previous.GitCommit || *run.ConfigSnapshot != snapshot || run.ImageURL == previous.ImageURL || !strings.HasPrefix(run.ImageURL, "registry.example:5000/team/api:") {
		t.Fatalf("%+v %v", run, err)
	}
	previous.GitCommit = ""
	if _, err := workflowRetryRun(previous, request, 2, 7); err == nil {
		t.Fatal("moving branch allowed for retry")
	}
	previous.GitCommit = strings.Repeat("a", 40)
	previous.TriggerKey = "workflow:99:build"
	if _, err := workflowRetryRun(previous, request, 2, 7); err == nil {
		t.Fatal("cross-workflow retry accepted")
	}
}
func TestTestJobHasIsolatedWorkspaceAndNoPrivilegedBuilder(t *testing.T) {
	job, err := BuildKubeJob("ci", ProviderBuildRequest{BuildRunID: 42, ServiceID: 9, BuildMode: "test", BuilderImage: "golang:1.24", BuildCommand: "go test ./...", TimeoutSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	pod := job.Spec.Template.Spec
	container := pod.Containers[0]
	if container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
		t.Fatal("privileged test")
	}
	if *pod.AutomountServiceAccountToken || *container.SecurityContext.AllowPrivilegeEscalation || len(container.SecurityContext.Capabilities.Drop) != 1 || container.WorkingDir != "/workspace" || *job.Spec.ActiveDeadlineSeconds != 60 {
		t.Fatal("test isolation or deadline missing")
	}
	if job.Labels["axiops.io/execution-kind"] != "test" || !strings.Contains(pod.InitContainers[0].Args[0], "git rev-parse HEAD >/dev/termination-log") {
		t.Fatal("source provenance missing")
	}
}
