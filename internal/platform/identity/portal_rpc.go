package identity

import (
	"context"
	"errors"

	"github.com/ChenJustin666/axiops/application/portal-rpc/client/sysauthservice"
)

type PortalRPCVerifier struct {
	client sysauthservice.SysAuthService
}

func NewPortalRPCVerifier(client sysauthservice.SysAuthService) *PortalRPCVerifier {
	return &PortalRPCVerifier{client: client}
}

func (v *PortalRPCVerifier) Verify(ctx context.Context, bearerToken string) (LegacyPrincipal, error) {
	if v.client == nil || bearerToken == "" {
		return LegacyPrincipal{}, ErrInvalidLegacyToken
	}
	response, err := v.client.VerifyToken(ctx, &sysauthservice.VerifyTokenRequest{Token: bearerToken})
	if err != nil {
		return LegacyPrincipal{}, err
	}
	if !response.IsValid || response.UserId == 0 {
		return LegacyPrincipal{}, ErrInvalidLegacyToken
	}
	if response.UserId > uint64(^uint64(0)>>1) {
		return LegacyPrincipal{}, errors.New("legacy user id exceeds int64")
	}
	return LegacyPrincipal{ExternalUserID: int64(response.UserId), Username: response.Username}, nil
}
