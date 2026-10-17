package templatelib

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authz"
)

type fakeAccess struct {
	view  bool
	write bool
}

func (a fakeAccess) Scopes(context.Context, int64, string, string) (authz.ScopeSet, error) {
	return authz.ScopeSet{AllowAll: a.view}, nil
}
func (a fakeAccess) Authorize(context.Context, int64, authz.Request) (bool, error) {
	return a.write, nil
}
func (a fakeAccess) Policies(context.Context, int64) ([]authz.Policy, error) { return nil, nil }

func TestTemplateAuthorizationBeforeDatabaseAccess(t *testing.T) {
	h := NewHandler(nil, fakeAccess{})
	for _, entry := range []struct {
		method string
		call   func(*httptest.ResponseRecorder)
	}{
		{"read", func(w *httptest.ResponseRecorder) { h.List(w, httptest.NewRequest("GET", "/", nil)) }},
		{"versions", func(w *httptest.ResponseRecorder) { h.Versions(w, httptest.NewRequest("GET", "/", nil)) }},
		{"version", func(w *httptest.ResponseRecorder) { h.Version(w, httptest.NewRequest("GET", "/", nil)) }},
		{"usages", func(w *httptest.ResponseRecorder) { h.Usages(w, httptest.NewRequest("GET", "/", nil)) }},
		{"service template", func(w *httptest.ResponseRecorder) { h.ServiceTemplate(w, httptest.NewRequest("GET", "/", nil)) }},
		{"create", func(w *httptest.ResponseRecorder) { h.Save(w, httptest.NewRequest("POST", "/", nil)) }},
		{"archive", func(w *httptest.ResponseRecorder) { h.Archive(w, httptest.NewRequest("DELETE", "/", nil)) }},
	} {
		w := httptest.NewRecorder()
		entry.call(w)
		if w.Code != 403 {
			t.Fatalf("%s code=%d", entry.method, w.Code)
		}
	}
}
func TestPreviewKeepsPlaceholdersAndRejectsMalformedInput(t *testing.T) {
	h := NewHandler(nil, fakeAccess{view: true})
	data, _ := json.Marshal(Defaults()[0])
	w := httptest.NewRecorder()
	h.Preview(w, httptest.NewRequest("POST", "/", bytes.NewReader(data)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var result struct {
		Data struct {
			Definition string `json:"definition"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Data.Definition == "" {
		t.Fatal("missing snapshot")
	}
	for _, body := range []string{`{"kind":"k8s_yaml","unknown":1}`, string(data) + ` {}`} {
		w = httptest.NewRecorder()
		h.Preview(w, httptest.NewRequest("POST", "/", bytes.NewBufferString(body)))
		if w.Code == 200 {
			t.Fatal("invalid request accepted")
		}
	}
}
