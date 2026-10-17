package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestSyncPathValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/42/services/7", nil)
	req = pathvar.WithVars(req, map[string]string{"id": "42", "service_id": "7"})

	called := false
	handler := SyncPathValues(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if got := r.PathValue("id"); got != "42" {
			t.Fatalf("PathValue(id) = %q, want 42", got)
		}
		if got := r.PathValue("service_id"); got != "7" {
			t.Fatalf("PathValue(service_id) = %q, want 7", got)
		}
	})

	handler(httptest.NewRecorder(), req)
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestSyncPathValuesWithoutParameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	handler := SyncPathValues(func(_ http.ResponseWriter, r *http.Request) {
		if got := r.PathValue("id"); got != "" {
			t.Fatalf("PathValue(id) = %q, want empty", got)
		}
	})

	handler(httptest.NewRecorder(), req)
}
