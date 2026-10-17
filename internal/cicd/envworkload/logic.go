package envworkload

import (
	"context"
	"strings"

	"github.com/ChenJustin666/axiops/internal/shared/errcode"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

// EnvironmentResolver returns the cluster id + namespace an environment is
// bound to, but only after the caller passed the environment "view" check.
// It is satisfied by *projectenv.Logic, keeping authorization in one place.
type EnvironmentResolver interface {
	ResolveK8sTarget(ctx context.Context, userID, environmentID int64) (clusterID int64, namespace string, code int)
}

type Logic struct {
	resolver EnvironmentResolver
	factory  RESTConfigFactory
	restarts interface {
		RestartCapabilities(context.Context, int64, int64) (map[string]bool, error)
	}
}

func (l *Logic) WithRestartCapabilities(provider interface {
	RestartCapabilities(context.Context, int64, int64) (map[string]bool, error)
}) *Logic { l.restarts = provider; return l }

func NewLogic(resolver EnvironmentResolver, factory RESTConfigFactory) *Logic {
	return &Logic{resolver: resolver, factory: factory}
}

// client resolves the environment, authorizes, and returns a read-only k8s
// client bound to the same cluster the environment deploys to.
func (l *Logic) client(ctx context.Context, userID, environmentID int64) (*k8sClient, string, int) {
	clusterID, namespace, code := l.resolver.ResolveK8sTarget(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, "", code
	}
	if clusterID <= 0 || strings.TrimSpace(namespace) == "" {
		return nil, "", errcode.CodeParamInvalid
	}
	c, err := newK8sClient(ctx, l.factory, clusterID)
	if err != nil {
		return nil, "", errcode.CodeClusterConnectFail
	}
	return c, namespace, errcode.CodeSuccess
}

// ListWorkloads returns workload detail rows for one kind.
func (l *Logic) ListWorkloads(ctx context.Context, userID, environmentID int64, kind, search string) ([]WorkloadItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	var items []WorkloadItem
	var err error
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment":
		var list []appsv1.Deployment
		if list, err = c.listDeployments(ctx, ns); err == nil {
			for i := range list {
				items = append(items, deploymentToItem(&list[i]))
			}
		}
	case "statefulset":
		var list []appsv1.StatefulSet
		if list, err = c.listStatefulSets(ctx, ns); err == nil {
			for i := range list {
				items = append(items, statefulSetToItem(&list[i]))
			}
		}
	case "daemonset":
		var list []appsv1.DaemonSet
		if list, err = c.listDaemonSets(ctx, ns); err == nil {
			for i := range list {
				items = append(items, daemonSetToItem(&list[i]))
			}
		}
	case "cronjob":
		var list []batchv1.CronJob
		if list, err = c.listCronJobs(ctx, ns); err == nil {
			for i := range list {
				items = append(items, cronJobToItem(&list[i]))
			}
		}
	case "job":
		var list []batchv1.Job
		if list, err = c.listJobs(ctx, ns); err == nil {
			for i := range list {
				items = append(items, jobToItem(&list[i]))
			}
		}
	default:
		return nil, errcode.CodeParamInvalid
	}
	if err != nil {
		return nil, resourceReadCode(err)
	}
	items = filterWorkloads(items, search)
	if l.restarts != nil && len(items) > 0 {
		capabilities, err := l.restarts.RestartCapabilities(ctx, userID, environmentID)
		if err != nil {
			return nil, errcode.CodeDBError
		}
		for i := range items {
			items[i].RestartAllowed = capabilities[items[i].Kind+"/"+items[i].Name]
		}
	}
	if err := c.enrichWorkloadStatuses(ctx, ns, items); err != nil {
		return nil, resourceReadCode(err)
	}
	if items == nil {
		items = []WorkloadItem{}
	}
	return items, errcode.CodeSuccess
}

// ListPods returns pod rows for the environment namespace.
func (l *Logic) ListPods(ctx context.Context, userID, environmentID int64, search string) ([]PodItem, int) {
	c, ns, code := l.client(ctx, userID, environmentID)
	if code != errcode.CodeSuccess {
		return nil, code
	}
	pods, err := c.listPods(ctx, ns)
	if err != nil {
		return nil, resourceReadCode(err)
	}
	items := make([]PodItem, 0, len(pods))
	for i := range pods {
		it := podToItem(&pods[i])
		if search == "" || strings.Contains(strings.ToLower(it.Name), strings.ToLower(strings.TrimSpace(search))) {
			items = append(items, it)
		}
	}
	return items, errcode.CodeSuccess
}

func filterWorkloads(items []WorkloadItem, search string) []WorkloadItem {
	kw := strings.ToLower(strings.TrimSpace(search))
	if kw == "" {
		return items
	}
	out := items[:0]
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Name), kw) {
			out = append(out, it)
		}
	}
	return out
}
