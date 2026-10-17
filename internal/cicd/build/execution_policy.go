package build

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ExecutionPolicy contains platform scheduling/storage choices. Queue leases,
// recovery and archival belong to the common worker, not to a Kubernetes Job.
type ExecutionPolicy struct {
	LogRetentionDays int               `json:"log_retention_days,omitempty"`
	NodeSelector     map[string]string `json:"node_selector,omitempty"`
	Tolerations      []BuildToleration `json:"tolerations,omitempty"`
	MaxBuildsPerNode int               `json:"max_builds_per_node,omitempty"`
	WorkspaceSize    string            `json:"workspace_size,omitempty"`
}

type BuildToleration struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

func (p ExecutionPolicy) Validate() error {
	if len(p.NodeSelector) > 20 || len(p.Tolerations) > 20 || p.MaxBuildsPerNode < 0 || p.MaxBuildsPerNode > 32 || p.LogRetentionDays < 0 || p.LogRetentionDays > 3650 {
		return fmt.Errorf("invalid build scheduling policy")
	}
	for key, value := range p.NodeSelector {
		if len(validation.IsQualifiedName(key)) != 0 || len(validation.IsValidLabelValue(value)) != 0 {
			return fmt.Errorf("invalid CI node label %q", key)
		}
	}
	for _, t := range p.Tolerations {
		if len(validation.IsQualifiedName(t.Key)) != 0 || len(validation.IsValidLabelValue(t.Value)) != 0 || (t.Effect != "NoSchedule" && t.Effect != "NoExecute" && t.Effect != "PreferNoSchedule") {
			return fmt.Errorf("invalid CI node toleration")
		}
	}
	if p.WorkspaceSize != "" {
		q, err := resource.ParseQuantity(p.WorkspaceSize)
		if err != nil || q.Cmp(resource.MustParse("100Mi")) < 0 {
			return fmt.Errorf("workspace storage must be at least 100Mi, for example 10Gi")
		}
	}
	return nil
}

func (p ExecutionPolicy) kubeTolerations() []corev1.Toleration {
	result := make([]corev1.Toleration, 0, len(p.Tolerations))
	for _, t := range p.Tolerations {
		result = append(result, corev1.Toleration{Key: t.Key, Value: t.Value, Operator: corev1.TolerationOpEqual, Effect: corev1.TaintEffect(t.Effect)})
	}
	return result
}
