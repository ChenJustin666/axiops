package envworkload

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func imagesOf(containers []corev1.Container) []string {
	out := make([]string, 0, len(containers))
	for _, c := range containers {
		out = append(out, c.Image)
	}
	return out
}

func deploymentToItem(d *appsv1.Deployment) WorkloadItem {
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	ready := d.Status.ReadyReplicas
	status := replicaStatus(desired, ready, d.Status.ObservedGeneration >= d.Generation && d.Status.UpdatedReplicas == desired && d.Status.AvailableReplicas == desired)
	if d.Spec.Paused {
		status = "Paused"
	}
	for _, condition := range d.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse && condition.Reason == "ProgressDeadlineExceeded" {
			status = condition.Reason
		}
	}
	return WorkloadItem{
		UID:             d.UID,
		Status:          status,
		Name:            d.Name,
		Namespace:       d.Namespace,
		Kind:            "Deployment",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          imagesOf(d.Spec.Template.Spec.Containers),
		Labels:          d.Labels,
		CreatedAt:       d.CreationTimestamp.UnixMilli(),
	}
}

func statefulSetToItem(s *appsv1.StatefulSet) WorkloadItem {
	desired := int32(1)
	if s.Spec.Replicas != nil {
		desired = *s.Spec.Replicas
	}
	ready := s.Status.ReadyReplicas
	return WorkloadItem{
		UID:             s.UID,
		Status:          replicaStatus(desired, ready, s.Status.ObservedGeneration >= s.Generation && (s.Spec.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType || s.Status.CurrentRevision == s.Status.UpdateRevision)),
		Name:            s.Name,
		Namespace:       s.Namespace,
		Kind:            "StatefulSet",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          imagesOf(s.Spec.Template.Spec.Containers),
		Labels:          s.Labels,
		CreatedAt:       s.CreationTimestamp.UnixMilli(),
	}
}

func daemonSetToItem(d *appsv1.DaemonSet) WorkloadItem {
	desired := d.Status.DesiredNumberScheduled
	ready := d.Status.NumberReady
	return WorkloadItem{
		UID:             d.UID,
		Status:          replicaStatus(desired, ready, d.Status.ObservedGeneration >= d.Generation && d.Status.UpdatedNumberScheduled == desired),
		Name:            d.Name,
		Namespace:       d.Namespace,
		Kind:            "DaemonSet",
		Ready:           fmt.Sprintf("%d/%d", ready, desired),
		DesiredReplicas: desired,
		ReadyReplicas:   ready,
		Images:          imagesOf(d.Spec.Template.Spec.Containers),
		Labels:          d.Labels,
		CreatedAt:       d.CreationTimestamp.UnixMilli(),
	}
}

func cronJobToItem(c *batchv1.CronJob) WorkloadItem {
	suspend := false
	if c.Spec.Suspend != nil {
		suspend = *c.Spec.Suspend
	}
	status := "Active"
	if suspend {
		status = "Suspended"
	}
	return WorkloadItem{
		Name:      c.Name,
		Namespace: c.Namespace,
		Kind:      "CronJob",
		Schedule:  c.Spec.Schedule,
		Suspend:   suspend,
		Status:    status,
		Images:    imagesOf(c.Spec.JobTemplate.Spec.Template.Spec.Containers),
		Labels:    c.Labels,
		CreatedAt: c.CreationTimestamp.UnixMilli(),
	}
}

func jobToItem(j *batchv1.Job) WorkloadItem {
	completions := int32(1)
	if j.Spec.Completions != nil {
		completions = *j.Spec.Completions
	}
	status := "Running"
	if j.Status.Active == 0 {
		status = "Pending"
	}
	for _, condition := range j.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			continue
		}
		if condition.Type == batchv1.JobComplete {
			status = "Completed"
		}
		if condition.Type == batchv1.JobFailed {
			status = "Failed"
		}
	}
	return WorkloadItem{
		Name:        j.Name,
		Namespace:   j.Namespace,
		Kind:        "Job",
		Status:      status,
		Completions: fmt.Sprintf("%d/%d", j.Status.Succeeded, completions),
		Images:      imagesOf(j.Spec.Template.Spec.Containers),
		Labels:      j.Labels,
		CreatedAt:   j.CreationTimestamp.UnixMilli(),
	}
}

func replicaStatus(desired, ready int32, observed bool) string {
	if !observed {
		return "Progressing"
	}
	if desired == 0 {
		return "ScaledToZero"
	}
	if ready == desired {
		return "Ready"
	}
	return "NotReady"
}
