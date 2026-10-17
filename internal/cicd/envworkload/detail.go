package envworkload

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"
	"github.com/ChenJustin666/axiops/internal/shared/middleware"
	"github.com/ChenJustin666/axiops/internal/shared/response"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

type ResourceEvent struct {
	ResourceKind string `json:"resource_kind,omitempty"`
	ResourceName string `json:"resource_name,omitempty"`
	Type         string `json:"type"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Count        int32  `json:"count"`
	Time         int64  `json:"time"`
}
type ResourceDetail struct {
	Namespace   string          `json:"namespace"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	YAML        string          `json:"yaml"`
	Redacted    bool            `json:"redacted"`
	Events      []ResourceEvent `json:"events"`
	EventsError string          `json:"events_error,omitempty"`
}

func resourceReadCode(err error) int {
	switch {
	case err == nil:
		return errcode.CodeSuccess
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return errcode.CodeNoPermission
	case apierrors.IsNotFound(err):
		return errcode.CodeNotFound
	default:
		return errcode.CodeClusterConnectFail
	}
}

func resourceRoute(c *k8sClient, kind string) (rest.Interface, string) {
	switch kind {
	case "deployment":
		return c.cs.AppsV1().RESTClient(), "deployments"
	case "statefulset":
		return c.cs.AppsV1().RESTClient(), "statefulsets"
	case "daemonset":
		return c.cs.AppsV1().RESTClient(), "daemonsets"
	case "job":
		return c.cs.BatchV1().RESTClient(), "jobs"
	case "cronjob":
		return c.cs.BatchV1().RESTClient(), "cronjobs"
	case "ingress":
		return c.cs.NetworkingV1().RESTClient(), "ingresses"
	case "pod":
		return c.cs.CoreV1().RESTClient(), "pods"
	case "service":
		return c.cs.CoreV1().RESTClient(), "services"
	case "persistentvolumeclaim":
		return c.cs.CoreV1().RESTClient(), "persistentvolumeclaims"
	case "configmap":
		return c.cs.CoreV1().RESTClient(), "configmaps"
	case "secret":
		return c.cs.CoreV1().RESTClient(), "secrets"
	default:
		return nil, ""
	}
}

func (l *Logic) GetResourceDetail(ctx context.Context, userID, environmentID int64, kind, name string) (*ResourceDetail, int) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if environmentID <= 0 || len(validation.IsDNS1123Subdomain(name)) != 0 {
		return nil, errcode.CodeParamInvalid
	}
	kind = strings.ToLower(kind)
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	client, resource := resourceRoute(c, kind)
	if client == nil {
		return nil, errcode.CodeParamInvalid
	}
	raw, err := client.Get().Namespace(ns).Resource(resource).Name(name).Do(ctx).Raw()
	if err != nil {
		return nil, resourceReadCode(err)
	}
	var object map[string]interface{}
	if json.Unmarshal(raw, &object) != nil {
		return nil, errcode.CodeClusterConnectFail
	}
	metadata, ok := object["metadata"].(map[string]interface{})
	if !ok || metadata["namespace"] != ns || metadata["name"] != name {
		return nil, errcode.CodeParamInvalid
	}
	uid, _ := metadata["uid"].(string)
	delete(metadata, "managedFields")
	if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
		delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}
	result := &ResourceDetail{Namespace: ns, Name: name, Kind: kind, Events: []ResourceEvent{}}
	if kind == "secret" {
		result.Redacted = true
		delete(metadata, "annotations")
		for _, field := range []string{"data", "stringData"} {
			if data, ok := object[field].(map[string]interface{}); ok {
				for key := range data {
					data[key] = "<redacted>"
				}
			}
		}
	}
	manifest, err := yaml.Marshal(object)
	if err != nil {
		return nil, errcode.CodeClusterConnectFail
	}
	result.YAML = string(manifest)
	if uid != "" {
		uids, relatedErr := c.relatedEventUIDs(ctx, ns, kind, types.UID(uid))
		if relatedErr != nil {
			result.EventsError = "关联资源读取失败，事件可能不完整；请检查集群权限或稍后刷新"
		}
		selector := fields.OneTermEqualSelector("involvedObject.uid", uid).String()
		if len(uids) > 1 {
			selector = ""
		}
		events, err := c.listResourceEvents(ctx, ns, selector, uids)
		if err != nil {
			result.EventsError = "事件读取失败，请检查集群权限或稍后刷新"
		} else {
			for _, event := range events {
				stamp := event.LastTimestamp.Time
				count := event.Count
				if event.Series != nil {
					stamp = event.Series.LastObservedTime.Time
					count = event.Series.Count
				}
				if stamp.IsZero() {
					stamp = event.EventTime.Time
				}
				if stamp.IsZero() {
					stamp = event.CreationTimestamp.Time
				}
				if count == 0 {
					count = 1
				}
				result.Events = append(result.Events, ResourceEvent{ResourceKind: event.InvolvedObject.Kind, ResourceName: event.InvolvedObject.Name, Type: event.Type, Reason: event.Reason, Message: event.Message, Count: count, Time: stamp.UnixMilli()})
			}
			sort.SliceStable(result.Events, func(i, j int) bool { return result.Events[i].Time > result.Events[j].Time })
			if len(result.Events) > 200 {
				result.Events = result.Events[:200]
			}
		}
	}
	return result, errcode.CodeSuccess
}

func (l *Logic) GetPodLogs(ctx context.Context, userID, environmentID int64, name, container string, previous bool) (string, int) {
	if environmentID <= 0 || len(validation.IsDNS1123Subdomain(name)) != 0 || len(validation.IsDNS1123Label(container)) != 0 {
		return "", errcode.CodeParamInvalid
	}
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return "", code
	}
	pod, err := c.cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", resourceReadCode(err)
	}
	found := false
	for _, item := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if item.Name == container {
			found = true
		}
	}
	for _, item := range pod.Spec.EphemeralContainers {
		if item.Name == container {
			found = true
		}
	}
	if !found {
		return "", errcode.CodeParamInvalid
	}
	tail, limit := int64(500), int64(512*1024)
	raw, err := c.cs.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{Container: container, Previous: previous, TailLines: &tail, LimitBytes: &limit, Timestamps: true}).DoRaw(ctx)
	if err != nil {
		return "", resourceReadCode(err)
	}
	return string(raw), errcode.CodeSuccess
}

func (h *Handler) ResourceDetail(w http.ResponseWriter, r *http.Request) {
	result, code := h.logic.GetResourceDetail(r.Context(), middleware.GetUserID(r.Context()), envID(r), r.URL.Query().Get("kind"), r.URL.Query().Get("name"))
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, result)
}
func (h *Handler) PodLogs(w http.ResponseWriter, r *http.Request) {
	logs, code := h.logic.GetPodLogs(r.Context(), middleware.GetUserID(r.Context()), envID(r), r.URL.Query().Get("name"), r.URL.Query().Get("container"), r.URL.Query().Get("previous") == "1")
	if code != errcode.CodeSuccess {
		response.Error(w, code)
		return
	}
	response.Success(w, map[string]string{"logs": logs})
}
