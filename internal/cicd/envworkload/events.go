package envworkload

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Walk only controller ownership in the authorized namespace; matching labels
// or a reused object name must never include another workload's events.
func (c *k8sClient) relatedEventUIDs(ctx context.Context, ns, kind string, uid types.UID) (map[types.UID]bool, error) {
	uids := map[types.UID]bool{uid: true}
	add := func(object metav1.Object, parentKind string) {
		owner := metav1.GetControllerOf(object)
		if object.GetNamespace() == ns && object.GetUID() != "" && owner != nil && owner.Kind == parentKind && uids[owner.UID] {
			uids[object.GetUID()] = true
		}
	}
	parentKind := ""
	switch kind {
	case "deployment":
		list, err := c.cs.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return uids, err
		}
		for i := range list.Items {
			add(&list.Items[i], "Deployment")
		}
		parentKind = "ReplicaSet"
	case "cronjob":
		list, err := c.listJobs(ctx, ns)
		if err != nil {
			return uids, err
		}
		for i := range list {
			add(&list[i], "CronJob")
		}
		parentKind = "Job"
	case "job":
		parentKind = "Job"
	case "statefulset":
		parentKind = "StatefulSet"
	case "daemonset":
		parentKind = "DaemonSet"
	default:
		return uids, nil
	}
	pods, err := c.listPods(ctx, ns)
	if err != nil {
		return uids, err
	}
	for i := range pods {
		add(&pods[i], parentKind)
	}
	return uids, nil
}

func (c *k8sClient) listResourceEvents(ctx context.Context, ns, selector string, uids map[types.UID]bool) ([]corev1.Event, error) {
	var result []corev1.Event
	options := metav1.ListOptions{FieldSelector: selector, Limit: 200}
	for {
		list, err := c.cs.CoreV1().Events(ns).List(ctx, options)
		if err != nil {
			return nil, err
		}
		for _, event := range list.Items {
			if event.Namespace == ns && uids[event.InvolvedObject.UID] && (event.InvolvedObject.Namespace == "" || event.InvolvedObject.Namespace == ns) {
				result = append(result, event)
			}
		}
		if list.Continue == "" {
			return result, nil
		}
		options.Continue = list.Continue
	}
}
