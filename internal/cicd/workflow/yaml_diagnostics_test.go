package workflow

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestContainerDiagnosticsHandlesMissingStateAndPreviousCrash(t *testing.T) {
	pod := unstructured.Unstructured{Object: map[string]interface{}{"status": map[string]interface{}{"containerStatuses": []interface{}{
		map[string]interface{}{"name": "starting"},
		map[string]interface{}{"name": "api", "state": map[string]interface{}{"waiting": map[string]interface{}{"reason": "CrashLoopBackOff"}}, "lastState": map[string]interface{}{"terminated": map[string]interface{}{"reason": "Error", "exitCode": int64(1)}}},
	}}}}
	result := podContainerDiagnostic(pod)
	if strings.Contains(result, "<nil>") {
		t.Fatal("missing optional fields leaked as <nil>")
	}
	for _, expected := range []string{"starting ready=false", "CrashLoopBackOff", "previous=Error exitCode=1"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("missing %q: %s", expected, result)
		}
	}
}

func TestDynamicDiagnosticsSelectorsEventsAndRedaction(t *testing.T) {
	client := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		{Version: "v1", Resource: "pods"}: "PodList", {Version: "v1", Resource: "events"}: "EventList",
	})
	workload := unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]interface{}{"name": "api", "uid": "workload-uid"}, "spec": map[string]interface{}{"selector": map[string]interface{}{"matchExpressions": []interface{}{map[string]interface{}{"key": "app", "operator": "In", "values": []interface{}{"api"}}}}}}}
	queriedPods := false
	uids := []string{}
	client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		a := action.(ktesting.ListAction)
		queriedPods = true
		if a.GetNamespace() != "qa" || a.GetListRestrictions().Labels.String() != "app in (api)" {
			t.Fatalf("wrong pod scope: %+v", a)
		}
		return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "api-1", "uid": "pod-uid", "labels": map[string]interface{}{"app": "api"}}, "status": map[string]interface{}{"phase": "Pending", "containerStatuses": []interface{}{map[string]interface{}{"name": "api", "state": map[string]interface{}{"waiting": map[string]interface{}{"reason": "ImagePullBackOff"}}}}}}}}}, nil
	})
	client.PrependReactor("list", "events", func(action ktesting.Action) (bool, runtime.Object, error) {
		a := action.(ktesting.ListAction)
		uids = append(uids, a.GetListRestrictions().Fields.String())
		return true, &unstructured.UnstructuredList{Items: []unstructured.Unstructured{{Object: map[string]interface{}{"reason": "Failed", "message": "pull denied token=hidden https://user:pass@example.com", "lastTimestamp": "2026-09-29T10:00:00Z"}}}}, nil
	})
	result, err := (&dynamicResourceApplier{client: client}).DiagnoseWorkload(context.Background(), "qa", workload)
	if err != nil || !queriedPods || len(uids) != 2 || uids[0] != "involvedObject.uid=workload-uid" || uids[1] != "involvedObject.uid=pod-uid" {
		t.Fatalf("result=%s uids=%v err=%v", result, uids, err)
	}
	if !strings.Contains(result, "ImagePullBackOff") || !strings.Contains(result, "[event] Pod/api-1") || strings.Contains(result, "hidden") || strings.Contains(result, "user:pass") {
		t.Fatalf("diagnostic=%s", result)
	}
	queriedPods = false
	unstructured.SetNestedMap(workload.Object, map[string]interface{}{}, "spec", "selector")
	_, err = (&dynamicResourceApplier{client: client}).DiagnoseWorkload(context.Background(), "qa", workload)
	if err != nil || queriedPods {
		t.Fatalf("empty selector queried namespace err=%v", err)
	}
}
