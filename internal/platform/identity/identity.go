package identity

import (
	"context"
	"errors"
	"time"

	"github.com/ChenJustin666/axiops/internal/shared/authn"
)

var (
	ErrInvalidLegacyToken = errors.New("legacy token is invalid")
	ErrIdentityUnmapped   = errors.New("legacy identity is not mapped")
	ErrTargetUserDisabled = errors.New("target user is disabled")
)

type LegacyPrincipal struct {
	ExternalUserID int64
	Username       string
}

type LegacyTokenVerifier interface {
	Verify(ctx context.Context, bearerToken string) (LegacyPrincipal, error)
}

type IdentityMapper interface {
	TargetPrincipal(ctx context.Context, provider string, externalUserID int64) (authn.Principal, error)
}

type Resolver struct {
	provider string
	timeout  time.Duration
	verifier LegacyTokenVerifier
	mapper   IdentityMapper
}

func NewResolver(provider string, timeout time.Duration, verifier LegacyTokenVerifier, mapper IdentityMapper) *Resolver {
	return &Resolver{provider: provider, timeout: timeout, verifier: verifier, mapper: mapper}
}

func (r *Resolver) Authenticate(ctx context.Context, bearerToken string) (authn.Principal, error) {
	if r.verifier == nil || r.mapper == nil || r.provider == "" {
		return authn.Principal{}, ErrInvalidLegacyToken
	}
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}
	legacy, err := r.verifier.Verify(ctx, bearerToken)
	if err != nil {
		return authn.Principal{}, err
	}
	if legacy.ExternalUserID <= 0 {
		return authn.Principal{}, ErrInvalidLegacyToken
	}
	principal, err := r.mapper.TargetPrincipal(ctx, r.provider, legacy.ExternalUserID)
	if err != nil {
		return authn.Principal{}, err
	}
	if principal.UserID <= 0 || principal.Username == "" {
		return authn.Principal{}, ErrIdentityUnmapped
	}
	return principal, nil
}
