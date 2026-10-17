package build

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistryConnectivitySupportsBearerChallenge(t *testing.T) {
	var tokenRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/":
			if r.Header.Get("Authorization") == "Bearer test-token" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s/token",service="test-registry"`, "http://"+r.Host))
			w.WriteHeader(http.StatusUnauthorized)
		case "/token":
			tokenRequests++
			user, password, ok := r.BasicAuth()
			if !ok || user != "robot" || password != "secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"test-token"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	err := testRegistryConnectivity(context.Background(), server.URL+"/sinozo", &RegistryAuth{Username: "robot", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if tokenRequests != 1 {
		t.Fatalf("expected one token exchange, got %d", tokenRequests)
	}
}

func TestRegistryConnectivityRejectsInvalidBasicCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoded := base64.StdEncoding.EncodeToString([]byte("robot:secret"))
		if r.Header.Get("Authorization") != "Basic "+encoded {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := testRegistryConnectivity(context.Background(), server.URL, &RegistryAuth{Username: "robot", Password: "wrong"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected an explicit 401 error, got %v", err)
	}
}
