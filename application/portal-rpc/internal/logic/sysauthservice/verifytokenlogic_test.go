package sysauthservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/authz"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/model"
	"github.com/ChenJustin666/axiops/application/portal-rpc/internal/svc"
	"github.com/ChenJustin666/axiops/application/portal-rpc/pb"
	"github.com/ChenJustin666/axiops/pkg/jwt"
)

type currentPrincipalFixture struct {
	user  *authz.Principal
	role  string
	token *model.SysToken
	err   error
}

func (f *currentPrincipalFixture) User(context.Context, uint64) (*authz.Principal, error) {
	return f.user, f.err
}
func (f *currentPrincipalFixture) Role(context.Context, uint64) (string, error) { return f.role, f.err }
func (f *currentPrincipalFixture) Token(context.Context, string) (*model.SysToken, error) {
	return f.token, f.err
}

func TestJWTPrincipalUsesCurrentAccountAndRoles(t *testing.T) {
	f := &currentPrincipalFixture{user: &authz.Principal{UserID: 7, Username: "operator", Nickname: "Current name", Roles: []string{"viewer"}}}
	l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: f})
	claims := &jwt.JWTClaims{UserName: &jwt.AccountInfo{UserId: 7, UserName: "operator", NickName: "Old name", Uuid: "session-id", Roles: []string{"SUPER_ADMIN"}}}
	claims.ExpiresAt = time.Now().Add(time.Hour).Unix()
	for _, roles := range [][]string{{"viewer"}, {}, {"developer"}} {
		f.user.Roles = roles
		res, err := l.currentJWTPrincipal(claims)
		if err != nil || !res.IsValid || !reflect.DeepEqual(res.Roles, roles) || res.NickName != "Current name" || res.Uuid != "session-id" || res.ExpireTime != claims.ExpiresAt {
			t.Fatalf("current principal not used: response=%+v err=%v", res, err)
		}
	}
	// The exact same token claims are rejected after account deletion/freeze.
	f.user = nil
	res, err := l.currentJWTPrincipal(claims)
	if err != nil || res.IsValid {
		t.Fatalf("inactive account accepted: %+v %v", res, err)
	}
}

func TestJWTPrincipalRejectsIdentityMismatchAndLookupFailure(t *testing.T) {
	claims := &jwt.JWTClaims{UserName: &jwt.AccountInfo{UserId: 7, UserName: "operator", Roles: []string{"SUPER_ADMIN"}}}
	for _, user := range []*authz.Principal{{UserID: 8, Username: "operator"}, {UserID: 7, Username: "another-account"}} {
		l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: &currentPrincipalFixture{user: user}})
		res, err := l.currentJWTPrincipal(claims)
		if err != nil || res.IsValid {
			t.Fatalf("mismatched identity accepted: %+v %v", res, err)
		}
	}
	f := &currentPrincipalFixture{err: errors.New("unavailable")}
	l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: f})
	if res, err := l.currentJWTPrincipal(claims); err == nil || res != nil {
		t.Fatal("lookup error must fail closed")
	}
	l = NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{})
	if res, err := l.currentJWTPrincipal(claims); err == nil || res != nil {
		t.Fatal("missing store must fail closed")
	}
	for _, claims := range []*jwt.JWTClaims{nil, {}, {UserName: &jwt.AccountInfo{}}} {
		if res, err := l.currentJWTPrincipal(claims); err != nil || res.IsValid {
			t.Fatal("malformed claims must be invalid")
		}
	}
}

func TestSystemTokenRevocationAndCurrentOwner(t *testing.T) {
	f := &currentPrincipalFixture{user: &authz.Principal{UserID: 7, Username: "operator", Roles: []string{"viewer"}}, token: &model.SysToken{Id: 1, OwnerType: 1, OwnerId: 7, Status: 1}}
	l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: f})
	check := func(want bool) {
		t.Helper()
		res, err := l.VerifyToken(&pb.VerifyTokenRequest{Token: "Bearer test-system-token"})
		if err != nil || res.IsValid != want {
			t.Fatalf("valid=%v response=%+v error=%v", want, res, err)
		}
		if want && !reflect.DeepEqual(res.Roles, f.user.Roles) {
			t.Fatalf("stale roles: %v", res.Roles)
		}
	}
	check(true)
	f.user.Roles = []string{}
	check(true) // authenticated with no privileges, never inject a default role.
	f.user = nil
	check(false)
	f.token.Status = 0
	check(false)
	f.token.Status = 1
	f.token.ExpireTime = sql.NullTime{Valid: true, Time: time.Now().Add(-time.Second)}
	check(false)
	f.token = nil
	check(false)
}

func TestRoleOwnedSystemTokenRejectsDeletedRole(t *testing.T) {
	f := &currentPrincipalFixture{role: "viewer", token: &model.SysToken{Id: 2, OwnerType: 2, OwnerId: 9, Name: "automation", Status: 1}}
	l := NewVerifyTokenLogic(context.Background(), &svc.ServiceContext{Principals: f})
	res, err := l.VerifyToken(&pb.VerifyTokenRequest{Token: "Bearer test-system-token"})
	if err != nil || !res.IsValid || res.UserId != 0 || !reflect.DeepEqual(res.Roles, []string{"viewer"}) {
		t.Fatalf("role token failed: %+v %v", res, err)
	}
	f.role = ""
	res, err = l.VerifyToken(&pb.VerifyTokenRequest{Token: "Bearer test-system-token"})
	if err != nil || res.IsValid {
		t.Fatal("deleted role accepted")
	}
	f.err = errors.New("unavailable")
	if res, err := l.VerifyToken(&pb.VerifyTokenRequest{Token: "Bearer test-system-token"}); err == nil || res != nil {
		t.Fatal("database error accepted")
	}
}

func TestRefreshCannotRestoreRevokedJWTRoles(t *testing.T) {
	f := &currentPrincipalFixture{user: &authz.Principal{UserID: 7, Username: "operator", Roles: []string{"viewer"}}}
	l := NewRefreshTokenLogic(context.Background(), &svc.ServiceContext{Principals: f})
	claims := &jwt.JWTClaims{UserName: &jwt.AccountInfo{UserId: 7, UserName: "operator", Uuid: "same-session", Roles: []string{"SUPER_ADMIN"}}}
	account, err := l.currentRefreshAccount(claims)
	if err != nil || !reflect.DeepEqual(account.Roles, []string{"viewer"}) || account.Uuid != "same-session" {
		t.Fatalf("refresh uses stale identity: %+v %v", account, err)
	}
	f.user = nil
	if account, err := l.currentRefreshAccount(claims); err == nil || account != nil {
		t.Fatal("inactive account refreshed")
	}
	f.err = errors.New("unavailable")
	if account, err := l.currentRefreshAccount(claims); err == nil || account != nil {
		t.Fatal("lookup failure refreshed")
	}
	if account, err := l.currentRefreshAccount(&jwt.JWTClaims{}); err == nil || account != nil {
		t.Fatal("malformed claims refreshed")
	}
}
