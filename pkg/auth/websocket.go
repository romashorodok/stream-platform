package auth

import (
	"context"
	"fmt"

	"github.com/romashorodok/stream-platform/pkg/tokenutils"
)

type RefreshTokenAuthenticator struct {
	resolver IdentityPublicKeyResolver
}

func (s *RefreshTokenAuthenticator) Validate(context context.Context, plainToken string) ([]byte, error) {
	keyset, err := s.resolver.GetKeys(context, plainToken)

	verifiedPayload, err := tokenutils.VerifyTokenByKeySet(keyset, plainToken)
	if err != nil {
		return nil, fmt.Errorf("Not verified token. Error: %s", err)
	}

	err = tokenutils.ValidateToken(string(verifiedPayload))
	if err != nil {
		return nil, fmt.Errorf("Invalid token. Error: %s", err)
	}

	return verifiedPayload, nil
}

func NewRefreshTokenAuthenticator(Resolver IdentityPublicKeyResolver) *RefreshTokenAuthenticator {
	return &RefreshTokenAuthenticator{
		resolver: Resolver,
	}
}
