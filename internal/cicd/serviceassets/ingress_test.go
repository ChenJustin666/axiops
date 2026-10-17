package serviceassets

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"testing"
)

func albRoute() Route {
	return Route{Name: "route", Host: "api.example.com", Path: "/", Service: "risk-server", Port: 9099, Class: "alb"}
}

func TestIngressALBRoundTrip(t *testing.T) {
	for _, pathType := range []string{"", "Prefix", "Exact", "ImplementationSpecific"} {
		for _, tlsMode := range []string{"off", "cloud", "secret", "legacy"} {
			t.Run(pathType+"/"+tlsMode, func(t *testing.T) {
				route := albRoute()
				route.PathType = pathType
				if pathType == "ImplementationSpecific" {
					route.Path = "/api(/|$)(.*)"
				}
				route.Annotations = map[string]string{"alb.ingress.kubernetes.io/use-regex": "true", "alb.ingress.kubernetes.io/rewrite-target": "/${2}", "alb.ingress.kubernetes.io/order": "1", "alb.ingress.kubernetes.io/listen-ports": `[{"HTTP":80},{"HTTPS":443}]`}
				route.Labels = map[string]string{"ingress-controller": "alb"}
				if tlsMode == "cloud" || tlsMode == "secret" {
					route.TLSEnabled = true
					route.TLSHosts = []string{route.Host}
				}
				if tlsMode == "secret" || tlsMode == "legacy" {
					route.TLSSecret = "tls-cert"
				}
				settings := Empty()
				settings.Routes = []Route{route}
				data, _ := json.Marshal(settings)
				var restored Settings
				if err := json.Unmarshal(data, &restored); err != nil {
					t.Fatal(err)
				}
				out, _, err := Apply(baseManifest(), restored, 17, "risk-prod")
				if err != nil {
					t.Fatal(err)
				}
				ingress := out[len(out)-1]
				if ingress.GetKind() != "Ingress" || ingress.GetNamespace() != "risk-prod" || !reflect.DeepEqual(ingress.GetAnnotations(), route.Annotations) {
					t.Fatalf("incorrect ingress: %#v", ingress)
				}
				if ingress.GetLabels()["ingress-controller"] != "alb" || ingress.GetLabels()["axiops.io/service-id"] != "17" {
					t.Fatal("lost labels")
				}
				rules, _, _ := unstructured.NestedSlice(ingress.Object, "spec", "rules")
				path := rules[0].(map[string]interface{})["http"].(map[string]interface{})["paths"].([]interface{})[0].(map[string]interface{})
				if path["pathType"] != routePathType(route) || path["path"] != route.Path {
					t.Fatal("incorrect path")
				}
				backend := path["backend"].(map[string]interface{})["service"].(map[string]interface{})
				if backend["name"] != "risk-server" || backend["port"].(map[string]interface{})["number"] != int64(9099) {
					t.Fatal("incorrect backend")
				}
				tls, found, _ := unstructured.NestedSlice(ingress.Object, "spec", "tls")
				if found != (tlsMode != "off") {
					t.Fatal("unexpected TLS")
				}
				if found {
					entry := tls[0].(map[string]interface{})
					if !reflect.DeepEqual(entry["hosts"], []interface{}{route.Host}) {
						t.Fatal("incorrect TLS domains")
					}
					_, hasSecret := entry["secretName"]
					if hasSecret != (tlsMode == "secret" || tlsMode == "legacy") {
						t.Fatal("incorrect certificate source")
					}
				}
			})
		}
	}
}

func TestIngressRejectsInvalidConfiguration(t *testing.T) {
	cases := map[string]func(*Route){
		"path type":          func(r *Route) { r.PathType = "invalid" },
		"path traversal":     func(r *Route) { r.Path = "/../api" },
		"tls domain missing": func(r *Route) { r.TLSEnabled = true },
		"tls domain invalid": func(r *Route) { r.TLSEnabled = true; r.TLSHosts = []string{"https://api.example.com"} },
		"tls disabled":       func(r *Route) { r.TLSHosts = []string{r.Host} },
		"invalid annotation": func(r *Route) { r.Annotations = map[string]string{"": "value"} },
		"system labels":      func(r *Route) { r.Labels = map[string]string{"axiops.io/service-id": "999"} },
		"regex boolean":      func(r *Route) { r.Annotations = map[string]string{"alb.ingress.kubernetes.io/use-regex": "yes"} },
		"listener string": func(r *Route) {
			r.Annotations = map[string]string{"alb.ingress.kubernetes.io/listen-ports": `[{"HTTP":"80"}]`}
		},
		"listener duplicate": func(r *Route) {
			r.Annotations = map[string]string{"alb.ingress.kubernetes.io/listen-ports": `[{"HTTP":80},{"HTTPS":80}]`}
		},
		"listener fraction": func(r *Route) {
			r.Annotations = map[string]string{"alb.ingress.kubernetes.io/listen-ports": `[{"HTTP":80.5}]`}
		},
		"wrong backend":                          func(r *Route) { r.Service = "missing" },
		"container port instead of service port": func(r *Route) { r.Port = 8080 },
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			r := albRoute()
			edit(&r)
			s := Empty()
			s.Routes = []Route{r}
			if _, _, err := Apply(baseManifest(), s, 17, "test"); err == nil {
				t.Fatal("invalid ingress accepted")
			}
		})
	}
}
