package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestAuditResource(t *testing.T) {
	req := httptest.NewRequest("PUT", "/api/v1/access/bindings/42", nil)
	req.SetPathValue("id", "42")
	resource, id := auditResource(req)
	if resource != "role_binding" || id != 42 {
		t.Fatalf("resource=%q id=%d", resource, id)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/projects/8/members", nil)
	req.SetPathValue("id", "8")
	resource, id = auditResource(req)
	if resource != "project" || id != 8 {
		t.Fatalf("resource=%q id=%d", resource, id)
	}
	req = httptest.NewRequest("POST", "/api/v1/services/17/environment-config", nil)
	req.SetPathValue("service_id", "17")
	resource, id = auditResource(req)
	if resource != "service" || id != 17 {
		t.Fatalf("resource=%q id=%d", resource, id)
	}
}

func TestClientIPUsesPeerAddress(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/roles", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("X-Forwarded-For", "192.0.2.10, 10.0.0.1")
	if got := clientIP(req); got != "10.0.0.2" {
		t.Fatalf("clientIP=%q", got)
	}
}
