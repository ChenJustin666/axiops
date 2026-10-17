package workflow

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Validate the actual SDK render, including upgrades whose .Release.IsUpgrade
// branch differs from the configuration preview. Never mutate manifests after
// Helm has stored its release: use the official post-renderer instead.
type helmPostRenderer struct {
	namespace  string
	prepare    func([]byte) ([]byte, error)
	rejectJobs bool
}

func (p helmPostRenderer) Run(in *bytes.Buffer) (*bytes.Buffer, error) {
	data := in.Bytes()
	if _, err := parseHelmManifestResources(data, p.namespace); err != nil {
		return nil, err
	}
	if p.rejectJobs {
		if hasJob, err := helmManifestHasJob(data, p.namespace); err != nil {
			return nil, err
		} else if hasJob {
			return nil, fmt.Errorf("Chart 升级分支新增了 Job，无法安全使用原子回滚；请将任务显式建模为独立 Job 服务或 Helm hook")
		}
	}
	if p.prepare != nil {
		var err error
		data, err = p.prepare(data)
		if err != nil {
			return nil, err
		}
	}
	if _, err := parseHelmManifestResources(data, p.namespace); err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func validateHelmBuiltImage(manifest []byte, namespace, expected string) error {
	if expected == "" {
		return nil
	}
	resources, err := parseHelmManifestResources(manifest, namespace)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		for _, path := range [][]string{workloadPodSpec(resource)} {
			for _, field := range []string{"containers", "initContainers"} {
				containers, _, _ := unstructured.NestedSlice(resource.Object, append(path, field)...)
				for _, raw := range containers {
					if c, ok := raw.(map[string]interface{}); ok && c["image"] == expected {
						return nil
					}
				}
			}
		}
	}
	return fmt.Errorf("Chart 没有使用本次构建的镜像，请检查当前环境的镜像 values 路径映射")
}
