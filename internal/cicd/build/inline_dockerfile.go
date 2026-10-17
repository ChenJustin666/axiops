package build

import (
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"path"
	"regexp"
	"strings"
)

var dockerfileFrom = regexp.MustCompile(`(?im)^\s*FROM\s+\S+`)

func validateDockerfileInput(source, content, context string) error {
	if source == "" || source == "repository" {
		return nil
	}
	if source != "inline" {
		return fmt.Errorf("Dockerfile 来源无效")
	}
	if len(content) > 64*1024 || strings.ContainsRune(content, 0) || !dockerfileFrom.MatchString(content) {
		return fmt.Errorf("平台 Dockerfile 不能为空，必须包含 FROM，且不能超过 64 KiB")
	}
	context = defaultString(context, ".")
	if path.IsAbs(context) || strings.Contains(context, "\\") || path.Clean(context) != context || context == ".." || strings.HasPrefix(context, "../") {
		return fmt.Errorf("构建上下文必须是仓库内的相对路径")
	}
	return nil
}

// Mount the frozen text independently of checkout. No shell interpolation of
// Dockerfile content, and no change to any tracked file in the source repository.
func mountInlineDockerfile(job *batchv1.Job, req ProviderBuildRequest) error {
	if err := validateDockerfileInput(req.DockerfileSource, req.DockerfileContent, req.BuildContext); err != nil {
		return err
	}
	if req.DockerfileSource != "inline" {
		return nil
	}
	name := ".axiops-generated.Dockerfile"
	pod := &job.Spec.Template.Spec
	pod.Volumes = append(pod.Volumes, corev1.Volume{Name: "inline-dockerfile", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
	checkout := &pod.InitContainers[0]
	checkout.VolumeMounts = append(checkout.VolumeMounts, corev1.VolumeMount{Name: "inline-dockerfile", MountPath: "/run/axiops-dockerfile"})
	checkout.Env = append(checkout.Env, corev1.EnvVar{Name: "AXIOPS_DOCKERFILE_CONTENT", Value: req.DockerfileContent})
	checkout.Args[0] += "\nprintf '%s\\n' \"$AXIOPS_DOCKERFILE_CONTENT\" > /run/axiops-dockerfile/Dockerfile\nchmod 644 /run/axiops-dockerfile/Dockerfile\necho '[prepare] Using saved platform Dockerfile'\n"
	builder := &pod.Containers[0]
	builder.VolumeMounts = append(builder.VolumeMounts, corev1.VolumeMount{Name: "inline-dockerfile", MountPath: path.Join("/workspace", defaultString(req.BuildContext, "."), name), SubPath: "Dockerfile", ReadOnly: true})
	for i := range builder.Env {
		if builder.Env[i].Name == "DOCKERFILE_PATH" {
			builder.Env[i].Value = name
		}
	}
	return nil
}
