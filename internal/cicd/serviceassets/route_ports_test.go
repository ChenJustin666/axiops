package serviceassets

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func portService(ports ...routeServicePort) []unstructured.Unstructured {
	items := []interface{}{}
	for _, p := range ports {
		items = append(items, map[string]interface{}{"name": p.name, "protocol": p.protocol, "port": p.number, "targetPort": int64(28080)})
	}
	return []unstructured.Unstructured{{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "api"}, "spec": map[string]interface{}{"ports": items}}}}
}

func TestReconcileRoutePorts(t *testing.T) {
	old := Empty()
	old.Routes = []Route{{Name: "route", Service: "api", Port: 8080}}
	for _, tt := range []struct {
		name          string
		before, after []routeServicePort
		edit          int
		want          int
	}{
		{"named", []routeServicePort{{"http", "TCP", 8080}}, []routeServicePort{{"http", "TCP", 28080}}, 8080, 28080},
		{"single unnamed", []routeServicePort{{"", "TCP", 8080}}, []routeServicePort{{"", "TCP", 28080}}, 8080, 28080},
		{"multi named", []routeServicePort{{"http", "TCP", 8080}, {"metrics", "TCP", 9090}}, []routeServicePort{{"metrics", "TCP", 9090}, {"http", "TCP", 28080}}, 8080, 28080},
		{"valid service port stays despite targetPort", []routeServicePort{{"http", "TCP", 8080}}, []routeServicePort{{"http", "TCP", 8080}}, 8080, 8080},
		{"explicit invalid selection", []routeServicePort{{"http", "TCP", 8080}}, []routeServicePort{{"http", "TCP", 28080}}, 1234, 1234},
		{"renamed port ambiguous", []routeServicePort{{"http", "TCP", 8080}}, []routeServicePort{{"admin", "TCP", 28080}}, 8080, 8080},
		{"unnamed multi ambiguous", []routeServicePort{{"", "TCP", 8080}}, []routeServicePort{{"", "TCP", 28080}, {"", "TCP", 9090}}, 8080, 8080},
		{"protocol changed", []routeServicePort{{"http", "TCP", 8080}}, []routeServicePort{{"http", "UDP", 28080}}, 8080, 8080},
		{"service removed", []routeServicePort{{"http", "TCP", 8080}}, nil, 8080, 8080},
	} {
		t.Run(tt.name, func(t *testing.T) {
			next := old
			next.Routes = append([]Route(nil), old.Routes...)
			next.Routes[0].Port = tt.edit
			got, notices := ReconcileRoutePorts(portService(tt.before...), portService(tt.after...), old, next)
			if got.Routes[0].Port != tt.want || (len(notices) > 0) != (tt.want != tt.edit) {
				t.Fatalf("port=%d notices=%v", got.Routes[0].Port, notices)
			}
			if old.Routes[0].Port != 8080 || next.Routes[0].Port != tt.edit {
				t.Fatal("mutated caller settings")
			}
		})
	}
}
