package build

import (
	"context"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestJarImageCompilesBeforePlatformBuilder(t *testing.T) {
	for _, custom := range []bool{false, true} {
		client := &fakeJobClient{}
		provider := NewKubeJobProvider("ci", client, client).WithDefaults(KubeJobDefaults{
			BuilderImage: "platform/buildkit:v1", BuildCommand: "exec /usr/local/bin/entrypoint.sh",
			CachePVCName: "dependencies", CacheDir: ".cache/maven",
		})
		req := ProviderBuildRequest{
			BuildRunID: 101, ProjectID: 1, ServiceID: 2, IdempotencyKey: "build:101",
			RepositoryURL: "https://git.example/app.git", BuildMode: "jar_image",
			CacheEnabled: true, CacheStrategy: "inherit", ResourceCPU: "500m", ResourceMemory: "2Gi",
			EnvVars: []EnvVar{{Key: "MAVEN_OPTS", Value: "-Xmx1g"}, {Key: "CACHE_DIR", Value: "/wrong"}},
		}
		if custom {
			req.BuilderImage, req.BuildCommand = "team/maven:17", "mvn -pl ad-gateway -am package"
		}
		if _, err := provider.Submit(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		pod := client.created.Spec.Template.Spec
		if len(pod.InitContainers) != 2 || pod.InitContainers[0].Name != "git-clone" || pod.InitContainers[1].Name != "jar-build" {
			t.Fatalf("JAR compilation must follow checkout: %+v", pod.InitContainers)
		}
		compile, builder := pod.InitContainers[1], pod.Containers[0]
		wantImage, wantCommand := DefaultJarCompileImage, DefaultJarCompileCommand
		if custom {
			wantImage, wantCommand = req.BuilderImage, req.BuildCommand
		}
		if compile.Image != wantImage || compile.Args[0] != "set -eu\n"+wantCommand || compile.WorkingDir != "/workspace" {
			t.Fatalf("compile template/custom command not applied: %+v", compile)
		}
		if builder.Image != "platform/buildkit:v1" || builder.Args[0] != "exec /usr/local/bin/entrypoint.sh" {
			t.Fatalf("image output contract lost: %+v", builder)
		}
		for _, env := range builder.Env {
			if env.Name == "AXIOPS_BUILD_MODE" && env.Value != "dockerfile" {
				t.Fatal("platform builder must receive the Dockerfile stage")
			}
		}
		for _, env := range compile.Env {
			if env.Name == "CACHE_DIR" && env.Value != "/workspace/.cache/maven" {
				t.Fatal("cache path override was not protected")
			}
		}
		if compile.VolumeMounts[0].Name != builder.VolumeMounts[0].Name || compile.VolumeMounts[1].SubPath != "project-1/service-2" {
			t.Fatal("compile step must share checkout and isolated cache")
		}
		if compile.Resources.Limits.Memory().String() != "2Gi" || compile.SecurityContext.Privileged != nil {
			t.Fatal("compile step resources/security incorrect")
		}
	}
}

type jarLogs struct{}

func (jarLogs) Read(_ context.Context, _, container string, _ int64) ([]byte, error) {
	return []byte(container + " output"), nil
}

func TestJarCompileFailureIsLoggedBeforeBuilderStarts(t *testing.T) {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{InitContainers: []corev1.Container{{Name: "git-clone"}, {Name: "jar-build"}}},
		Status: corev1.PodStatus{Phase: corev1.PodFailed,
			InitContainerStatuses: []corev1.ContainerStatus{{Name: "jar-build", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"}}}},
			ContainerStatuses:     []corev1.ContainerStatus{{Name: "builder", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "PodInitializing"}}}},
		},
	}
	pod.Name = "jar-pod"
	client := &fakeJobClient{existing: &batchv1.Job{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded"}}}}}
	state, err := NewKubeJobProvider("ci", client, client, diagnosticPods{pod: pod}).WithRuntimeData(nil, jarLogs{}, 1024).Status(context.Background(), ProviderExecution{JobName: "jar-job"})
	if err != nil || !strings.Contains(state.LogText, "[jar-build]\njar-build output") || strings.Contains(state.LogText, "[builder]") || !strings.Contains(state.ErrorMessage, "jar-build exited with code 1") {
		t.Fatalf("compile failure missing: %+v %v", state, err)
	}
}

func TestJarImageRequiresPlatformImageBuilder(t *testing.T) {
	client := &fakeJobClient{}
	_, err := NewKubeJobProvider("ci", client, client).Submit(context.Background(), ProviderBuildRequest{
		BuildMode: "jar_image", BuilderImage: "maven:3.9-eclipse-temurin-17", BuildCommand: "mvn package",
	})
	if err == nil || client.created != nil {
		t.Fatal("Maven image alone cannot produce platform image results")
	}
}
