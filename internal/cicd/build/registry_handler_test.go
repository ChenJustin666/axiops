package build

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

func TestRegistryCatalogValidatesAddressAndScope(t *testing.T) {
	valid := ImageRegistry{Name: "ACR", Type: "acr", URL: "registry.example/team", CredentialID: 3}
	if !validImageRegistry(&valid) || valid.URL != "https://registry.example/team" {
		t.Fatal("valid registry rejected")
	}
	for _, address := range []string{"file:///tmp/config", "https://user:pass@registry.example", "https://registry.example?token=secret", "https://registry.example/team/../other"} {
		item := valid
		item.URL = address
		if validImageRegistry(&item) {
			t.Fatalf("unsafe address accepted: %s", address)
		}
	}
	valid.ProjectID = 42
	if validImageRegistry(&valid) {
		t.Fatal("global catalog accepted a project override")
	}
}

func TestRegistryNamespaceConfiguration(t *testing.T) {
	item := ImageRegistry{Name: "registry", Type: "harbor", URL: "https://registry.example/sinozo", CredentialID: 1, Namespaces: []string{"sinozo", " abc ", "abc", "dbc/team"}}
	if !validImageRegistry(&item) || strings.Join(item.Namespaces, ",") != "sinozo,abc,dbc/team" {
		t.Fatalf("namespace choices not normalized: %+v", item.Namespaces)
	}
	for _, namespace := range []string{"", "../abc", "abc/", "abc//def", "https://other.example", "Uppercase", "abc:tag", "abc@digest"} {
		item.Namespaces = []string{namespace}
		if validImageRegistry(&item) {
			t.Fatalf("invalid namespace accepted: %q", namespace)
		}
	}
}

type denyRegistryAuth struct{}

func (denyRegistryAuth) Authorize(context.Context, int64, authz.Request) (bool, error) {
	return false, nil
}
func (denyRegistryAuth) Scopes(context.Context, int64, string, string) (authz.ScopeSet, error) {
	return authz.ScopeSet{DenyAll: true}, nil
}
func (denyRegistryAuth) Policies(context.Context, int64) ([]authz.Policy, error) { return nil, nil }

func TestRegistryCatalogRejectsUnauthorizedAccessBeforeDatabase(t *testing.T) {
	h := NewHandler(&Logic{authorizer: denyRegistryAuth{}})
	for _, method := range []string{"GET", "POST"} {
		req := httptest.NewRequest(method, "/api/v1/build-registries", strings.NewReader(`{}`))
		recorder := httptest.NewRecorder()
		if method == "GET" {
			h.ListImageRegistries(recorder, req)
		} else {
			h.SaveImageRegistry(recorder, req)
		}
		if recorder.Code != 403 {
			t.Fatalf("unauthorized %s status=%d body=%s", method, recorder.Code, recorder.Body.String())
		}
	}
}
