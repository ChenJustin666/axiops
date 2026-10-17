package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
	"google.golang.org/grpc"
)

type authFixture struct {
	sysauthservice.SysAuthService
	verified      *sysauthservice.VerifyTokenResponse
	verifyErr     error
	permission    *sysauthservice.ApiAuthResponse
	permissionErr error
	checked       *sysauthservice.ApiAuthRequest
}

func (f *authFixture) VerifyToken(context.Context, *sysauthservice.VerifyTokenRequest, ...grpc.CallOption) (*sysauthservice.VerifyTokenResponse, error) {
	return f.verified, f.verifyErr
}
func (f *authFixture) ApiAuth(ctx context.Context, in *sysauthservice.ApiAuthRequest, opts ...grpc.CallOption) (*sysauthservice.ApiAuthResponse, error) {
	f.checked = in
	return f.permission, f.permissionErr
}

func TestLegacyMiddlewareUsesVerifiedIdentityForModuleAuthorization(t *testing.T) {
	for _, path := range []string{"/manager/v1/logging/query", "/manager/v1/logging/download", "/console/v1/monitor/pod"} {
		f := &authFixture{verified: &sysauthservice.VerifyTokenResponse{IsValid: true, UserId: 7, Username: "operator", Roles: []string{"viewer"}}, permission: &sysauthservice.ApiAuthResponse{IsAuth: true}}
		called := false
		h := NewJWTAuthMiddleware(f).Handle(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if r.Context().Value("userId") != uint64(7) || !reflect.DeepEqual(r.Context().Value("roles"), []string{"viewer"}) {
				t.Fatal("verified principal not propagated")
			}
		})
		r := httptest.NewRequest(http.MethodPost, path, nil)
		r.Header.Set("Authorization", "Bearer same-session")
		r.Header.Set("X-User-Id", "1")
		r.Header.Set("X-Roles", "SUPER_ADMIN")
		h(httptest.NewRecorder(), r)
		if !called || f.checked == nil || f.checked.ApiPath != path || f.checked.ApiMethod != "POST" || !reflect.DeepEqual(f.checked.UserRoles, []string{"viewer"}) {
			t.Fatalf("incorrect authorization: %+v", f.checked)
		}
		// A later verification response revokes access for the unchanged token.
		f.verified.Roles = []string{}
		f.permission.IsAuth = false
		called = false
		w := httptest.NewRecorder()
		h(w, r)
		if called || w.Code != http.StatusUnauthorized || len(f.checked.UserRoles) != 0 {
			t.Fatal("revoked privileges reached handler")
		}
	}
}

func TestLegacyMiddlewareFailsClosed(t *testing.T) {
	for _, f := range []*authFixture{
		{},
		{verifyErr: errors.New("unavailable")},
		{verified: &sysauthservice.VerifyTokenResponse{}},
		{verified: &sysauthservice.VerifyTokenResponse{IsValid: true}},
		{verified: &sysauthservice.VerifyTokenResponse{IsValid: true}, permissionErr: errors.New("unavailable")},
		{verified: &sysauthservice.VerifyTokenResponse{IsValid: true}, permission: &sysauthservice.ApiAuthResponse{}},
	} {
		called := false
		h := NewJWTAuthMiddleware(f).Handle(func(http.ResponseWriter, *http.Request) { called = true })
		h(httptest.NewRecorder(), httptest.NewRequest("GET", "/manager/v1/logging/query", nil))
		if called {
			t.Fatal("failed authorization reached module handler")
		}
	}
}
