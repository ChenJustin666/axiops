package serviceassets

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

type routeServicePort struct {
	name, protocol string
	number         int64
}

func routePorts(items []unstructured.Unstructured, name string) []routeServicePort {
	var result []routeServicePort
	for _, item := range items {
		if item.GetKind() != "Service" || item.GetName() != name {
			continue
		}
		var svc corev1.Service
		if runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &svc) != nil {
			continue
		}
		for _, p := range svc.Spec.Ports {
			protocol := string(p.Protocol)
			if protocol == "" {
				protocol = "TCP"
			}
			result = append(result, routeServicePort{p.Name, protocol, int64(p.Port)})
		}
	}
	return result
}

// ReconcileRoutePorts follows an unchanged managed route across a Service port
// edit. It never guesses from containerPort/targetPort, rewrites native Ingress,
// or repairs an explicitly selected invalid backend. Apply remains strict.
func ReconcileRoutePorts(before, after []unstructured.Unstructured, previous, next Settings) (Settings, []string) {
	next.Routes = append([]Route(nil), next.Routes...)
	var notices []string
	for i, route := range next.Routes {
		oldPorts, newPorts := routePorts(before, route.Service), routePorts(after, route.Service)
		valid := false
		for _, p := range newPorts {
			if p.number == int64(route.Port) {
				valid = true
			}
		}
		if valid {
			continue
		}
		unchanged := false
		for _, old := range previous.Routes {
			if old.Name == route.Name && old.Service == route.Service && old.Port == route.Port {
				unchanged = true
			}
		}
		if !unchanged {
			continue
		}
		var candidates []routeServicePort
		for _, old := range oldPorts {
			if old.number != int64(route.Port) {
				continue
			}
			for _, p := range newPorts {
				if p.protocol == old.protocol && ((old.name != "" && p.name == old.name) || (old.name == "" && p.name == "" && len(oldPorts) == 1 && len(newPorts) == 1)) {
					candidates = append(candidates, p)
				}
			}
		}
		if len(candidates) != 1 {
			continue
		}
		next.Routes[i].Port = int(candidates[0].number)
		notices = append(notices, fmt.Sprintf("路由 %s 已跟随 Service %s 的端口：%d → %d", route.Name, route.Service, route.Port, candidates[0].number))
	}
	return next, notices
}
