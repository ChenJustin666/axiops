package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authn"
)

type fakeVerifier struct {
	principal LegacyPrincipal
	err       error
}

func (f fakeVerifier) Verify(context.Context, string) (LegacyPrincipal, error) {
	return f.principal, f.err
}

type fakeMapper struct {
	principal authn.Principal
	err       error
}

func (f fakeMapper) TargetPrincipal(context.Context, string, int64) (authn.Principal, error) {
	return f.principal, f.err
}

func TestResolverMapsVerifiedLegacyIdentity(t *testing.T) {
	resolver := NewResolver("legacy_autoops", time.Second,
		fakeVerifier{principal: LegacyPrincipal{ExternalUserID: 9, Username: "legacy"}},
		fakeMapper{principal: authn.Principal{UserID: 42, Username: "target"}},
	)
	principal, err := resolver.Authenticate(context.Background(), "Bearer token")
	if err != nil || principal.UserID != 42 || principal.Username != "target" {
		t.Fatalf("principal=%#v err=%v", principal, err)
	}
}

func TestResolverRejectsInvalidOrUnmappedIdentity(t *testing.T) {
	resolver := NewResolver("legacy_autoops", 0,
		fakeVerifier{principal: LegacyPrincipal{ExternalUserID: 0}}, fakeMapper{})
	if _, err := resolver.Authenticate(context.Background(), "Bearer token"); !errors.Is(err, ErrInvalidLegacyToken) {
		t.Fatalf("err=%v", err)
	}

	resolver = NewResolver("legacy_autoops", 0,
		fakeVerifier{principal: LegacyPrincipal{ExternalUserID: 9}}, fakeMapper{err: ErrIdentityUnmapped})
	if _, err := resolver.Authenticate(context.Background(), "Bearer token"); !errors.Is(err, ErrIdentityUnmapped) {
		t.Fatalf("err=%v", err)
	}

	resolver = NewResolver("legacy_autoops", 0,
		fakeVerifier{principal: LegacyPrincipal{ExternalUserID: 9}},
		fakeMapper{principal: authn.Principal{UserID: 0, Username: ""}},
	)
	if _, err := resolver.Authenticate(context.Background(), "Bearer token"); !errors.Is(err, ErrIdentityUnmapped) {
		t.Fatalf("invalid target principal err=%v", err)
	}
}
