package envworkload

import (
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Read once per namespace, never once per workload or replica. Ownership uses
// controller UIDs rather than names/labels, which may overlap or be reused.
func (c *k8sClient) enrichWorkloadStatuses(ctx context.Context, ns string, items []WorkloadItem) error {
	wanted := false
	deployments := false
	for _, item := range items {
		if needsPodStatus(item) {
			wanted = true
			deployments = deployments || item.Kind == "Deployment"
		}
	}
	if !wanted {
		return nil
	}
	pods, err := c.listPods(ctx, ns)
	if err != nil {
		return err
	}
	var replicas []appsv1.ReplicaSet
	if deployments {
		list, err := c.cs.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		replicas = list.Items
	}
	applyPodStatuses(items, pods, replicas)
	return nil
}

func needsPodStatus(item WorkloadItem) bool {
	return item.UID != "" && item.DesiredReplicas > 0 && item.Status != "Ready" &&
		(item.Kind == "Deployment" || item.Kind == "StatefulSet" || item.Kind == "DaemonSet")
}

func applyPodStatuses(items []WorkloadItem, pods []corev1.Pod, replicas []appsv1.ReplicaSet) {
	type ownerKey struct {
		namespace, kind string
		uid             types.UID
	}
	owners := map[ownerKey]int{}
	for i := range items {
		if needsPodStatus(items[i]) {
			owners[ownerKey{items[i].Namespace, items[i].Kind, items[i].UID}] = i
		}
	}
	for _, rs := range replicas {
		owner := metav1.GetControllerOf(&rs)
		if owner == nil || owner.Kind != "Deployment" || rs.UID == "" {
			continue
		}
		if i, ok := owners[ownerKey{rs.Namespace, owner.Kind, owner.UID}]; ok {
			owners[ownerKey{rs.Namespace, "ReplicaSet", rs.UID}] = i
		}
	}
	counts := map[int]map[string]int{}
	for _, pod := range pods {
		// Old terminating replicas and successful init containers are not failures.
		if pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		owner := metav1.GetControllerOf(&pod)
		if owner == nil {
			continue
		}
		i, ok := owners[ownerKey{pod.Namespace, owner.Kind, owner.UID}]
		if !ok {
			continue
		}
		reasons := currentPodReasons(&pod)
		if counts[i] == nil {
			counts[i] = map[string]int{}
		}
		for reason := range reasons {
			counts[i][reason]++
		}
	}
	for i, reasons := range counts {
		if len(reasons) == 0 {
			continue
		}
		item := &items[i]
		item.ControllerStatus = item.Status
		for reason, count := range reasons {
			item.StatusReasons = append(item.StatusReasons, WorkloadStatusReason{Reason: reason, Pods: count})
		}
		sort.Slice(item.StatusReasons, func(a, b int) bool {
			x, y := item.StatusReasons[a].Reason, item.StatusReasons[b].Reason
			if reasonPriority(x) != reasonPriority(y) {
				return reasonPriority(x) > reasonPriority(y)
			}
			return x < y
		})
		item.Status = item.StatusReasons[0].Reason
	}
}

func currentPodReasons(p *corev1.Pod) map[string]bool {
	reasons := map[string]bool{}
	for _, statuses := range [][]corev1.ContainerStatus{p.Status.InitContainerStatuses, p.Status.ContainerStatuses} {
		for _, cs := range statuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				reasons[cs.State.Waiting.Reason] = true
			}
			if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
				reason := cs.State.Terminated.Reason
				if reason == "" {
					reason = "Error"
				}
				reasons[reason] = true
			}
		}
	}
	if len(reasons) == 0 {
		if p.Status.Reason != "" {
			reasons[p.Status.Reason] = true
		}
		for _, condition := range p.Status.Conditions {
			if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse && condition.Reason != "" {
				reasons[condition.Reason] = true
			}
		}
	}
	return reasons
}

func reasonPriority(reason string) int {
	switch reason {
	case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "OOMKilled", "Error", "CreateContainerConfigError", "CreateContainerError", "RunContainerError", "InvalidImageName", "Evicted", "Unschedulable":
		return 3
	case "PodInitializing", "ContainerCreating":
		return 1
	default:
		return 2
	}
}
