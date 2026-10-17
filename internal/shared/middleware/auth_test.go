package middleware

import (
	"context"
	"github.com/ChenJustin666/axiops/internal/shared/jwtx"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChenJustin666/axiops/internal/shared/authn"
)

type fakeLegacyAuthenticator struct {
	principal authn.Principal
	err       error
	called    *bool
}

func (f fakeLegacyAuthenticator) Authenticate(context.Context, string) (authn.Principal, error) {
	if f.called != nil {
		*f.called = true
	}
	return f.principal, f.err
}

func TestAuthMiddlewareUsesLegacyAuthenticatorOnlyAfterNativeTokenFails(t *testing.T) {
	middleware := NewAuthMiddleware("native-secret").WithLegacyAuthenticator(fakeLegacyAuthenticator{
		principal: authn.Principal{UserID: 42, Username: "mapped-user"},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/menus", nil)
	request.Header.Set("Authorization", "Bearer legacy-token")
	recorder := httptest.NewRecorder()
	middleware.Handle(func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r.Context()) != 42 || GetUsername(r.Context()) != "mapped-user" {
			t.Fatalf("unexpected mapped principal: user=%d username=%q", GetUserID(r.Context()), GetUsername(r.Context()))
		}
		w.WriteHeader(http.StatusNoContent)
	})(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func TestAuthMiddlewareDoesNotFallbackForExpiredNativeToken(t *testing.T) {
	token, err := jwtx.GenerateToken("native-secret", 7, "expired-user", nil, -1)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	middleware := NewAuthMiddleware("native-secret").WithLegacyAuthenticator(fakeLegacyAuthenticator{
		principal: authn.Principal{UserID: 42, Username: "mapped-user"},
		called:    &called,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/menus", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	middleware.Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("expired token reached handler")
	})(recorder, request)
	if called {
		t.Fatal("legacy authenticator was called for an expired native token")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", recorder.Code)
	}
}
