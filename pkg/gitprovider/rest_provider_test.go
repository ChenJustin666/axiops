package gitprovider

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRESTProviderListsRepositoriesAndReadsFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if r.URL.Path == "/api/v1/user/repos" {
			if r.URL.Query().Get("limit") != "100" || r.URL.Query().Get("affiliation") == "" {
				t.Fatalf("gitea repository query = %q, want limit and affiliation", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("page") != "1" {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"id":7,"name":"api","full_name":"team/api","clone_url":"http://gitea/team/api.git","default_branch":"main","private":true}]`))
			return
		}
		if r.URL.Path == "/api/v1/repos/team/api/contents/README.md" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"content":"` + base64.StdEncoding.EncodeToString([]byte("hello")) + `\n","encoding":"base64"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	provider, err := New(&Config{Type: "gitea", BaseURL: server.URL, Token: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	repositories, err := provider.ListRepositories(context.Background())
	if err != nil || len(repositories) != 1 || repositories[0].FullName != "team/api" {
		t.Fatalf("repositories=%+v err=%v", repositories, err)
	}
	content, err := provider.GetFileContent(context.Background(), "team/api", "README.md", "main")
	if err != nil || string(content) != "hello" {
		t.Fatalf("content=%q err=%v", string(content), err)
	}
}

func TestRESTProviderUsesGitHubEnterpriseAPIPath(t *testing.T) {
	provider := newRESTProvider(&Config{Type: "github", BaseURL: "https://github.intra.example"}, "github")
	if !strings.HasSuffix(provider.apiBase, "/api/v3") {
		t.Fatalf("api base = %q", provider.apiBase)
	}
}

func TestInternalAwareProxyBypassesPrivateIP(t *testing.T) {
	privateReq, err := http.NewRequest(http.MethodGet, "http://192.168.200.13:13000/api/v1/user/repos", nil)
	if err != nil {
		t.Fatal(err)
	}
	if proxy, err := internalAwareProxy(privateReq); err != nil || proxy != nil {
		t.Fatalf("private request proxy = %v, err=%v; expected direct connection", proxy, err)
	}

	publicReq, err := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := internalAwareProxy(publicReq)
	// net/http caches proxy environment on its first use. Assert delegation
	// against that same policy instead of relying on test execution order.
	want, wantErr := http.ProxyFromEnvironment(publicReq)
	if err != wantErr || (proxy == nil) != (want == nil) || proxy != nil && proxy.String() != want.String() {
		t.Fatalf("public request proxy = %v, err=%v; expected %v, %v", proxy, err, want, wantErr)
	}
}
