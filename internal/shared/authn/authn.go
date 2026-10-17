package authn

import "context"

type Principal struct {
	UserID   int64
	Username string
}

// LegacyAuthenticator verifies a legacy token and returns only an identity
// that has already been mapped to the target AxiOps user model.
type LegacyAuthenticator interface {
	Authenticate(ctx context.Context, bearerToken string) (Principal, error)
}
