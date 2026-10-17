package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Query by the complete selector, including matchExpressions, never by an
// empty selector (which would expose unrelated pods in the namespace).
func (a *dynamicResourceApplier) DiagnoseWorkload(ctx context.Context, ns string, workload unstructured.Unstructured) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	selector, found, err := unstructured.NestedMap(workload.Object, "spec", "selector")
	if err != nil || !found {
		return "[pods] workload selector unavailable", nil
	}
	var labelSelector metav1.LabelSelector
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(selector, &labelSelector); err != nil {
		return "", err
	}
	parsed, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return "", err
	}
	if parsed.Empty() {
		return "[pods] empty selector; pod lookup skipped", nil
	}
	pods, err := a.client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "pods"}).Namespace(ns).List(ctx, metav1.ListOptions{LabelSelector: parsed.String(), Limit: 20})
	if err != nil {
		return "", fmt.Errorf("list pods: %w", err)
	}
	sort.Slice(pods.Items, func(i, j int) bool {
		a, b := pods.Items[i].GetCreationTimestamp(), pods.Items[j].GetCreationTimestamp()
		if a.Equal(&b) {
			return pods.Items[i].GetName() < pods.Items[j].GetName()
		}
		return a.After(b.Time)
	})
	details := []string{}
	// Query events by UID to avoid mixing historical events for recycled names.
	eventTargets := []unstructured.Unstructured{workload}
	limit := len(pods.Items)
	if limit > 5 {
		limit = 5
	}
	for _, pod := range pods.Items[:limit] {
		phase, _, _ := unstructured.NestedString(pod.Object, "status", "phase")
		details = append(details, fmt.Sprintf("[pod] %s phase=%s %s", pod.GetName(), phase, podContainerDiagnostic(pod)))
		conditions, _, _ := unstructured.NestedSlice(pod.Object, "status", "conditions")
		for _, raw := range conditions {
			c, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if c["status"] == "False" {
				details = append(details, conditionDiagnostic("[pod] "+pod.GetName(), c))
			}
		}
		eventTargets = append(eventTargets, pod)
	}
	if limit == 0 {
		if replicas, found, _ := unstructured.NestedInt64(workload.Object, "spec", "replicas"); found && replicas == 0 {
			details = append(details, "[pods] no matching pods; desired replicas=0 (scaled down)")
		} else {
			details = append(details, "[pods] no matching pods yet")
		}
	}
	if len(pods.Items) > limit || pods.GetContinue() != "" {
		details = append(details, "[pods] showing at most 5 pods; additional pods omitted")
	}
	for _, target := range eventTargets {
		if target.GetUID() == "" {
			continue
		}
		events, err := a.client.Resource(schema.GroupVersionResource{Version: "v1", Resource: "events"}).Namespace(ns).List(ctx, metav1.ListOptions{FieldSelector: "involvedObject.uid=" + string(target.GetUID()), Limit: 50})
		if err != nil {
			details = append(details, "[events] unavailable: "+err.Error())
			break
		}
		sort.Slice(events.Items, func(i, j int) bool { return eventTime(events.Items[i]) > eventTime(events.Items[j]) })
		count := len(events.Items)
		if count > 5 {
			count = 5
		}
		for _, event := range events.Items[:count] {
			reason, _, _ := unstructured.NestedString(event.Object, "reason")
			message, _, _ := unstructured.NestedString(event.Object, "message")
			details = append(details, fmt.Sprintf("[event] %s/%s %s %s: %s", target.GetKind(), target.GetName(), eventTime(event), reason, message))
		}
		if len(events.Items) > count || events.GetContinue() != "" {
			details = append(details, "[events] additional events omitted")
		}
	}
	for i := range details {
		details[i] = safeDiagnostic(details[i])
	}
	return truncateDiagnostic(strings.Join(details, "\n"), 16*1024), nil
}

func eventTime(event unstructured.Unstructured) string {
	for _, path := range [][]string{{"series", "lastObservedTime"}, {"lastTimestamp"}, {"eventTime"}, {"metadata", "creationTimestamp"}} {
		s, _, _ := unstructured.NestedString(event.Object, path...)
		if s != "" {
			return s
		}
	}
	return ""
}

func podContainerDiagnostic(pod unstructured.Unstructured) string {
	parts := []string{}
	for _, field := range []string{"initContainerStatuses", "containerStatuses"} {
		statuses, _, _ := unstructured.NestedSlice(pod.Object, "status", field)
		for _, raw := range statuses {
			status, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := status["name"].(string)
			ready, _ := status["ready"].(bool)
			state, _ := status["state"].(map[string]interface{})
			summary := fmt.Sprintf("container=%s ready=%t", name, ready)
			if count, ok := status["restartCount"]; ok {
				summary += fmt.Sprintf(" restarts=%v", count)
			}
			if waiting, ok := state["waiting"].(map[string]interface{}); ok {
				summary += " waiting=" + diagnosticValue(waiting["reason"])
				if message := diagnosticValue(waiting["message"]); message != "" {
					summary += " " + message
				}
			}
			if terminated, ok := state["terminated"].(map[string]interface{}); ok {
				summary += " terminated=" + diagnosticValue(terminated["reason"])
				if code, ok := terminated["exitCode"]; ok {
					summary += fmt.Sprintf(" exitCode=%v", code)
				}
			}
			last, _ := status["lastState"].(map[string]interface{})
			if terminated, ok := last["terminated"].(map[string]interface{}); ok {
				summary += " previous=" + diagnosticValue(terminated["reason"])
				if code, ok := terminated["exitCode"]; ok {
					summary += fmt.Sprintf(" exitCode=%v", code)
				}
			}
			parts = append(parts, summary)
		}
	}
	return strings.Join(parts, "; ")
}

func diagnosticValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func conditionDiagnostic(prefix string, condition map[string]interface{}) string {
	line := prefix + " " + diagnosticValue(condition["type"]) + "=" + diagnosticValue(condition["status"])
	if reason := diagnosticValue(condition["reason"]); reason != "" {
		line += " reason=" + reason
	}
	if message := diagnosticValue(condition["message"]); message != "" {
		line += " " + message
	}
	return line
}
