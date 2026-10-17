package observabilityaccess

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type fixtureStore struct {
	user   int64
	broken bool
	envID  int64
}

func (s *fixtureStore) Principal(context.Context, int64) (int64, error) {
	if s.broken {
		return 0, ErrDenied
	}
	return s.user, nil
}
func (s *fixtureStore) Resolve(_ context.Context, t Target) (authz.Request, error) {
	if s.broken || t.ClusterUUID != "cluster-a" {
		return authz.Request{}, ErrDenied
	}
	r := authz.Request{ClusterID: 2}
	switch t.Namespace {
	case "qa":
		r.ProjectID = 10
		r.EnvironmentID = s.envID
		r.Environment = "qa"
	case "prod":
		r.ProjectID = 20
		r.EnvironmentID = 22
		r.Environment = "prod"
	case "":
	default:
		return r, ErrDenied
	}
	if t.ProjectUUID != "" && (t.ProjectUUID != "project-a" || r.ProjectID != 10) {
		return r, ErrDenied
	}
	return r, nil
}

type policies struct {
	list   []authz.Policy
	broken bool
}

func (p *policies) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	if p.broken {
		return nil, errors.New("unavailable")
	}
	return p.list, nil
}
func allow(resource, action, scope string, id int64) authz.Policy {
	return authz.Policy{Resource: resource, Action: action, Effect: "allow", ScopeType: scope, ScopeID: id}
}
func fixture(ps ...authz.Policy) (*Guard, *fixtureStore, *policies) {
	s := &fixtureStore{user: 7, envID: 11}
	p := &policies{list: ps}
	g := New(s, authz.New(p), func(context.Context, uint64) (Target, error) {
		return Target{ClusterUUID: "cluster-a", Namespace: "qa"}, nil
	})
	return g, s, p
}
func checkRequest(g *Guard, path, body string) (*http.Request, error) {
	method := http.MethodGet
	if body != "" {
		method = http.MethodPost
	}
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	r = r.WithContext(context.WithValue(r.Context(), "userId", uint64(7)))
	return g.Check(r, func(context.Context) error { return nil })
}

func TestEnvironmentBoundary(t *testing.T) {
	g, _, _ := fixture(allow("monitoring", "view", authz.ScopeProject, 10), allow("logging", "view", authz.ScopeEnvironment, 11))
	for _, tc := range []struct {
		path, body string
		ok         bool
	}{
		{"/console/v1/namespace-monitor/cpu?clusterUuid=cluster-a&namespace=qa", "", true},
		{"/console/v1/namespace-monitor/cpu?clusterUuid=cluster-a&namespace=prod", "", false},
		{"/console/v1/pod-monitor/list?clusterUuid=cluster-a&namespace=qa", "", true},
		{"/console/v1/pod-monitor/list?clusterUuid=cluster-a", "", false},
		{"/console/v1/cluster-monitor/overview?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/console/v1/ingress-monitor/overview?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/console/v1/flagger-monitor/overview?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","queryText":"error"}`, true},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"prod"}`, false},
		{"/manager/v1/logging/download", `{"clusterUuid":"cluster-a","namespace":"qa"}`, false},
		{"/manager/v1/logging/labels/values", `{"clusterUuid":"cluster-a","namespace":"qa","workspaceId":17,"queryMode":"platform"}`, true},
		{"/manager/v1/logging/labels/values", `{"clusterUuid":"cluster-a","workspaceId":17}`, false},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","sourceType":"system"}`, false},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","mode":"code","expr":"{namespace_name=\"prod\"}"}`, false},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","queryMode":"loki","queryText":"{namespace_name=\"prod\"}"}`, false},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","command":"|~ \"secret\""}`, false},
		{"/manager/v1/logging/events/123?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/console/v1/pod-monitor/container/environment?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/workload/v1/monitoring/prometheusrule?clusterUuid=cluster-a&namespace=qa", "", false},
		{"/console/v1/pod/logs/get?workloadId=17&podName=app", "", true},
		{"/ws/v1/pod/logs/download?workloadId=17&podName=app", "", false},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			_, err := checkRequest(g, tc.path, tc.body)
			if (err == nil) != tc.ok {
				t.Fatalf("allowed=%v want=%v", err == nil, tc.ok)
			}
		})
	}
}

func TestMalformedAndAmbiguousTargets(t *testing.T) {
	g, _, _ := fixture(allow("*", "*", authz.ScopePlatform, 0))
	for _, tc := range []struct{ path, body string }{
		{"/console/v1/namespace-monitor/cpu?clusterUuid=cluster-a&namespace=*", ""},
		{"/console/v1/namespace-monitor/cpu?clusterUuid=cluster-a&Namespace=qa", ""},
		{"/console/v1/namespace-monitor/cpu?clusterUuid=cluster-a&namespace=qa&namespace=prod", ""},
		{"/console/v1/pod-monitor/cpu/usage?clusterUuid=cluster-a&namespace=qa&pod=a%22%7D%20or%20vector(1)%23", ""},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","Namespace":"prod"}`},
		{"/manager/v1/logging/query?namespace=qa", `{"clusterUuid":"cluster-a","namespace":"prod"}`},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa"} {}`},
		{"/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa","projectUuid":"another"}`},
		{"/manager/v1/monitoring/prometheus/cluster-a/config?clusterUuid=cluster-b", ""},
	} {
		if _, err := checkRequest(g, tc.path, tc.body); err == nil {
			t.Fatalf("accepted %s %s", tc.path, tc.body)
		}
	}
}

func TestAggregateCannotBypassNarrowDeny(t *testing.T) {
	for _, scope := range []string{authz.ScopeService, authz.ScopeProject, authz.ScopeEnvironment} {
		p := allow("logging", "view", scope, 33)
		p.Effect = "deny"
		g, _, _ := fixture(allow("*", "*", authz.ScopePlatform, 0), p)
		if _, err := checkRequest(g, "/manager/v1/logging/query", `{"clusterUuid":"cluster-a"}`); err == nil {
			t.Fatal("cluster aggregate bypassed deny")
		}
		if _, err := checkRequest(g, "/manager/v1/logging/query", `{"clusterUuid":"cluster-a","mode":"code","expr":"{}"}`); err == nil {
			t.Fatal("raw query bypassed deny")
		}
	}
	g, _, _ := fixture(allow("logging", "view", authz.ScopeCluster, 2))
	if _, err := checkRequest(g, "/manager/v1/logging/query", `{"clusterUuid":"cluster-a","mode":"code","expr":"{}"}`); err == nil {
		t.Fatal("raw query accepted with cluster-only grant")
	}
}

func TestStreamRevocationAndFrozenTargets(t *testing.T) {
	for _, change := range []string{"policy", "identity", "mapping", "workspace", "session", "database", "principal"} {
		t.Run(change, func(t *testing.T) {
			g, s, p := fixture(allow("*", "*", authz.ScopePlatform, 0))
			actual := Target{ClusterUUID: "cluster-a", Namespace: "qa"}
			g.workspace = func(context.Context, uint64) (Target, error) { return actual, nil }
			session := true
			r := httptest.NewRequest("GET", "/ws/v1/pod/logs/stream?workloadId=17&podName=app", nil)
			r = r.WithContext(context.WithValue(r.Context(), "userId", uint64(7)))
			authorized, err := g.Check(r, func(context.Context) error {
				if !session {
					return ErrDenied
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := CheckTarget(authorized.Context(), actual); err != nil {
				t.Fatal(err)
			}
			if err := CheckTarget(authorized.Context(), Target{ClusterUUID: "cluster-a", Namespace: "prod"}); err == nil {
				t.Fatal("handler retargeted")
			}
			switch change {
			case "policy":
				p.list = nil
			case "identity":
				s.broken = true
			case "mapping":
				s.envID = 55
			case "workspace":
				actual.Namespace = "prod"
			case "session":
				session = false
			case "database":
				p.broken = true
			case "principal":
				s.user = 8
			}
			if err := Recheck(authorized.Context()); err == nil {
				t.Fatal("stream not revoked")
			}
			w := Writer(authorized.Context(), io.Discard)
			if n, err := w.Write([]byte("must not send")); err == nil || n != 0 {
				t.Fatal("download continued after revocation")
			}
		})
	}
	if err := Recheck(context.Background()); err == nil {
		t.Fatal("missing context accepted")
	}
}

func TestReadExportAndManageRemainSeparate(t *testing.T) {
	g, _, _ := fixture(allow("logging", "export", authz.ScopeEnvironment, 11))
	if _, err := checkRequest(g, "/manager/v1/logging/download", `{"clusterUuid":"cluster-a","namespace":"qa"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := checkRequest(g, "/manager/v1/logging/query", `{"clusterUuid":"cluster-a","namespace":"qa"}`); err == nil {
		t.Fatal("export implied view")
	}
	g, _, _ = fixture(allow("monitoring", "manage", authz.ScopeCluster, 2))
	if _, err := checkRequest(g, "/manager/v1/monitoring/prometheus/cluster-a/config", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := New(nil, nil, nil).Check(httptest.NewRequest("GET", "/public", nil), nil); err != nil {
		t.Fatal("unrelated route affected")
	}
}

func TestLeaseRechecksBeforeSendingAfterExpiry(t *testing.T) {
	g, _, p := fixture(allow("*", "*", authz.ScopePlatform, 0))
	r, err := checkRequest(g, "/ws/v1/pod/logs/stream?workloadId=17", "")
	if err != nil {
		t.Fatal(err)
	}
	l := NewLease(r.Context())
	if err := l.Check(); err != nil {
		t.Fatal(err)
	}
	p.list = nil
	l.next = time.Now().Add(-time.Second)
	if err := l.Check(); err == nil {
		t.Fatal("lease retained revoked grants")
	}
}
