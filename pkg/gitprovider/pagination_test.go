package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestRepositoryPaginationSurvivesServerPageSizeCap(t *testing.T) {
	for _, providerType := range []string{"gitea", "github", "gitee", "gitlab"} {
		t.Run(providerType, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				if page != calls {
					t.Errorf("unexpected pagination: %s", r.URL.RawQuery)
				}
				if providerType == "gitea" && r.URL.Query().Get("limit") != "100" {
					t.Error("Gitea limit missing")
				}
				if r.Header.Get("Authorization") != "token test-token" && r.Header.Get("PRIVATE-TOKEN") != "test-token" {
					t.Error("token lost on next page")
				}
				// Request 100; server caps responses to 50. Target repo is on page 2.
				rows := make([]map[string]interface{}, 0)
				for i := (page - 1) * 50; i < page*50 && i < 73; i++ {
					name := fmt.Sprintf("team/repo-%03d", i)
					if i == 72 {
						name = "server/risk_server"
					}
					rows = append(rows, map[string]interface{}{"id": i + 1, "name": name, "full_name": name, "path_with_namespace": name, "private": true, "visibility": "private"})
				}
				_ = json.NewEncoder(w).Encode(rows)
			}))
			defer server.Close()
			provider, err := New(&Config{Type: providerType, BaseURL: server.URL, Token: "test-token"})
			if err != nil {
				t.Fatal(err)
			}
			items, err := provider.ListRepositories(context.Background())
			if err != nil || len(items) != 73 || calls != 3 || items[72].FullName != "server/risk_server" {
				t.Fatalf("repositories truncated: count=%d calls=%d err=%v", len(items), calls, err)
			}
		})
	}
}

func TestRepositoryPaginationFailsExplicitlyOnLaterPageErrorOrRepeatedPage(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusOK} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Query().Get("page") == "2" && status != http.StatusOK {
				w.WriteHeader(status)
				return
			}
			_, _ = w.Write([]byte(`[{"id":1,"full_name":"team/repo"}]`))
		}))
		provider, _ := New(&Config{Type: "gitea", BaseURL: server.URL, Token: "test-token"})
		items, err := provider.ListRepositories(context.Background())
		server.Close()
		want := ErrUnauthorized
		if status == http.StatusOK {
			want = ErrInvalidResponse
		}
		if !errors.Is(err, want) || items != nil || calls != 2 {
			t.Fatalf("partial list or infinite loop on status %d: items=%v calls=%d err=%v", status, items, calls, err)
		}
	}
}

func TestRepositoryPaginationDoesNotFollowExternalLink(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Link", `<https://untrusted.example/next>; rel="next"`)
		if calls == 1 {
			_, _ = w.Write([]byte(`[{"id":1,"full_name":"team/repo"}]`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	provider, _ := New(&Config{Type: "gitea", BaseURL: server.URL, Token: "test-token"})
	items, err := provider.ListRepositories(context.Background())
	if err != nil || len(items) != 1 || calls != 2 {
		t.Fatalf("unexpected link handling: %v %v", items, err)
	}
}
