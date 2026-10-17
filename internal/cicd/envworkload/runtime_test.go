package envworkload

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type runtimeResolver struct{ denied bool }

func (r runtimeResolver) ResolveK8sTarget(_ context.Context, _ int64, id int64) (int64, string, int) {
	if r.denied {
		return 0, "", errcode.CodeNoPermission
	}
	if id == 1 {
		return 10, "dev", errcode.CodeSuccess
	}
	if id == 2 {
		return 20, "qa", errcode.CodeSuccess
	}
	return 0, "", errcode.CodeNotFound
}

type runtimeFactory struct {
	url        string
	clusterIDs []int64
}

func (f *runtimeFactory) RESTConfig(_ context.Context, id int64) (*rest.Config, error) {
	f.clusterIDs = append(f.clusterIDs, id)
	return &rest.Config{Host: f.url}, nil
}

func TestRuntimeRequestsUseAuthorizedEnvironmentAndRedactSecrets(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/dev/deployments", "/apis/apps/v1/namespaces/qa/deployments":
			ns := "dev"
			if strings.Contains(r.URL.Path, "/qa/") {
				ns = "qa"
			}
			fmt.Fprintf(w, `{"apiVersion":"apps/v1","kind":"DeploymentList","items":[{"metadata":{"name":"api","namespace":%q},"spec":{"replicas":1,"template":{"spec":{"containers":[{"name":"app","image":"registry/api:actual"}]}}},"status":{"readyReplicas":0}}]}`, ns)
		case "/api/v1/namespaces/qa/secrets/auth":
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"auth","namespace":"qa","uid":"secret-id","annotations":{"private":"never-show","kubectl.kubernetes.io/last-applied-configuration":"never-show"}},"data":{"token":"bmV2ZXItc2hvdw=="},"stringData":{"password":"never-show"}}`) // gitleaks:allow -- base64 of the fixed test fixture never-show
		case "/api/v1/namespaces/qa/events":
			if r.URL.Query().Get("fieldSelector") != "involvedObject.uid=secret-id" {
				t.Errorf("events selector=%s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"EventList","items":[]}`)
		case "/api/v1/namespaces/dev/pods/api":
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"api","namespace":"dev"},"spec":{"containers":[{"name":"app"}],"initContainers":[{"name":"init"}]}}`)
		case "/api/v1/namespaces/dev/pods/api/log":
			q := r.URL.Query()
			if q.Get("container") != "init" || q.Get("previous") != "true" || q.Get("tailLines") != "500" || q.Get("limitBytes") != "524288" || q.Get("follow") == "true" {
				t.Errorf("unbounded/wrong log request: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "init finished\n")
		default:
			t.Errorf("unexpected cluster request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	factory := &runtimeFactory{url: server.URL}
	logic := NewLogic(runtimeResolver{}, factory)
	ctx := context.Background()
	for _, id := range []int64{1, 2} {
		rows, code := logic.ListWorkloads(ctx, 7, id, "deployment", "")
		if code != 0 || len(rows) != 1 || rows[0].Ready != "0/1" || rows[0].Status == "Ready" {
			t.Fatalf("unexpected rows=%+v code=%d", rows, code)
		}
	}
	if factory.clusterIDs[0] != 10 || factory.clusterIDs[1] != 20 {
		t.Fatal("wrong environment cluster")
	}
	detail, code := logic.GetResourceDetail(ctx, 7, 2, "secret", "auth")
	if code != 0 || !detail.Redacted || strings.Contains(detail.YAML, "never-show") || strings.Contains(detail.YAML, "bmV2ZXItc2hvdw==") || !strings.Contains(detail.YAML, "redacted") { // gitleaks:allow -- base64 of the fixed test fixture never-show
		t.Fatalf("secret not redacted, code=%d", code)
	}
	logs, code := logic.GetPodLogs(ctx, 7, 1, "api", "init", true)
	if code != 0 || logs != "init finished\n" {
		t.Fatalf("logs=%q code=%d", logs, code)
	}
	if _, code = logic.GetPodLogs(ctx, 7, 1, "api", "other", false); code != errcode.CodeParamInvalid {
		t.Fatal("unknown container accepted")
	}
	before := len(paths)
	denied := NewLogic(runtimeResolver{denied: true}, factory)
	if _, code = denied.GetResourceDetail(ctx, 7, 2, "secret", "auth"); code != errcode.CodeNoPermission {
		t.Fatal("authorization bypassed")
	}
	if _, code = logic.GetResourceDetail(ctx, 7, 1, "pod", "../qa/api"); code != errcode.CodeParamInvalid {
		t.Fatal("invalid resource name accepted")
	}
	if len(paths) != before {
		t.Fatal("denied request contacted cluster")
	}
}

func TestRuntimeReadFailuresAreNotEmptySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","code":403}`)
	}))
	defer server.Close()
	logic := NewLogic(runtimeResolver{}, &runtimeFactory{url: server.URL})
	if _, code := logic.ListWorkloads(context.Background(), 7, 1, "deployment", ""); code != errcode.CodeNoPermission {
		t.Fatalf("permission failure code=%d", code)
	}
	if _, code := logic.GetResourceDetail(context.Background(), 7, 1, "pod", "api"); code != errcode.CodeNoPermission {
		t.Fatalf("detail permission code=%d", code)
	}
}

func TestRuntimeStatusesUseReadinessAndJobConditions(t *testing.T) {
	p := corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}
	if podToItem(&p).Status != "NotReady" {
		t.Fatal("Running was treated as Ready")
	}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "app", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}}}
	if podToItem(&p).Status != "ImagePullBackOff" {
		t.Fatal("lost container reason")
	}
	p.Status.ContainerStatuses = nil
	p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	if podToItem(&p).Status != "Ready" {
		t.Fatal("ready condition ignored")
	}
	j := batchv1.Job{Status: batchv1.JobStatus{Succeeded: 1}}
	if jobToItem(&j).Status == "Completed" {
		t.Fatal("partial job inferred completed")
	}
	j.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
	if jobToItem(&j).Status != "Completed" {
		t.Fatal("job completion ignored")
	}
	one := int32(1)
	d := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Generation: 2}, Spec: appsv1.DeploymentSpec{Replicas: &one}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1, ReadyReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}}
	if deploymentToItem(&d).Status != "Progressing" {
		t.Fatal("old deployment generation treated as healthy")
	}
	d.Status.ObservedGeneration = 2
	if deploymentToItem(&d).Status != "Ready" {
		t.Fatal("ready rollout ignored")
	}
}

func TestWorkloadStatusUsesOwnedPodReason(t *testing.T) {
	one := int32(1)
	deployment := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{UID: "deploy-1", Name: "api", Namespace: "qa"}, Spec: appsv1.DeploymentSpec{Replicas: &one}, Status: appsv1.DeploymentStatus{ReadyReplicas: 0}}
	rs := appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{UID: "rs-1", Name: "api-rs", Namespace: "qa", OwnerReferences: []metav1.OwnerReference{{UID: deployment.UID, Kind: "Deployment", Controller: ptr(true)}}}}
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "api-pod", Namespace: "qa", OwnerReferences: []metav1.OwnerReference{{UID: rs.UID, Kind: "ReplicaSet", Controller: ptr(true)}}}, Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}}}
	items := []WorkloadItem{deploymentToItem(&deployment)}
	applyPodStatuses(items, []corev1.Pod{pod}, []appsv1.ReplicaSet{rs})
	if items[0].Status != "CrashLoopBackOff" || len(items[0].StatusReasons) != 1 || items[0].StatusReasons[0].Pods != 1 {
		t.Fatalf("pod reason was not propagated: %+v", items[0])
	}
}

func ptr(value bool) *bool { return &value }
