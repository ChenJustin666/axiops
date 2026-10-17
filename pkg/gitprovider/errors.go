package gitprovider

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupportedProvider = errors.New("gitprovider: unsupported provider type")
	ErrUnauthorized        = errors.New("gitprovider: authentication failed or token expired")
	ErrNotFound            = errors.New("gitprovider: repository or resource not found")
	ErrRateLimit           = errors.New("gitprovider: API rate limit exceeded")
	ErrWebhookSignature    = errors.New("gitprovider: webhook signature verification failed")
	ErrNetworkTimeout      = errors.New("gitprovider: network request timeout")
	ErrInvalidResponse     = errors.New("gitprovider: invalid response format")
)

// APIError is returned for a non-authentication HTTP response from a Git
// provider. The body is retained for server logs but is not sent to the UI.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gitprovider: API returned HTTP %d", e.Status)
}
