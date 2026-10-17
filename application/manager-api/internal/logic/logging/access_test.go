package logging

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ChenJustin666/axiops/common/observabilityaccess"
	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type accessStore struct{}

func (accessStore) Principal(context.Context, int64) (int64, error) { return 7, nil }
func (accessStore) Resolve(context.Context, observabilityaccess.Target) (authz.Request, error) {
	return authz.Request{ClusterID: 2, ProjectID: 10, EnvironmentID: 11, Environment: "qa"}, nil
}

type accessPolicies struct{}

func (accessPolicies) ListUserPolicies(context.Context, int64) ([]authz.Policy, error) {
	return []authz.Policy{{Resource: "logging", Action: "*", Effect: "allow", ScopeType: authz.ScopePlatform}}, nil
}

func authorizedLogContext(t *testing.T, revoked *bool) context.Context {
	t.Helper()
	guard := observabilityaccess.New(accessStore{}, authz.New(accessPolicies{}), nil)
	req := httptest.NewRequest("GET", "/manager/v1/logging/query-mode/options?clusterUuid=cluster-1&namespace=qa", nil)
	req = req.WithContext(context.WithValue(req.Context(), "userId", uint64(7)))
	req, err := guard.Check(req, func(context.Context) error {
		if revoked != nil && *revoked {
			return observabilityaccess.ErrDenied
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return req.Context()
}

func TestBufferedLogResponseRechecksAfterQuery(t *testing.T) {
	revoked := false
	ctx := authorizedLogContext(t, &revoked)
	revoked = true
	if err := observabilityaccess.Recheck(ctx); err == nil {
		t.Fatal("revoked response was authorized")
	}
}
