package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/lestrrat-go/jwx/v2/jwk"
	identitypb "github.com/romashorodok/stream-platform/gen/golang/identity/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/tokenutils"
	"google.golang.org/grpc/metadata"
)

type SecurityError struct {
	Err error
}

func GetSecurityErrorPrefix() string {
	return ": Security Error"
}

func (s *SecurityError) Error() string {
	return fmt.Sprintf("%s %s", s.Err.Error(), GetSecurityErrorPrefix())
}

func NewSecurityError(err error) *SecurityError {
	return &SecurityError{Err: err}
}

type GRPCPublicKeyResolver struct {
	client identitypb.PublicKeyServiceClient
}

var _ IdentityPublicKeyResolver = (*GRPCPublicKeyResolver)(nil)

func (r *GRPCPublicKeyResolver) GetKeys(ctx context.Context, token string) (jwk.Set, error) {

	md := metadata.Pairs("authorization", token)
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := r.client.PublicKeyList(ctx, &identitypb.PublicKeyListRequest{})

	if err != nil {
		return nil, fmt.Errorf("Unable get public keyset from identity service. Error: %s", err)
	}

	// TODO: Cache it by kid of token. By 15 min expire time
	rawKeyset := resp.Result
	keyset, err := jwk.Parse(rawKeyset)
	if err != nil {
		return nil, fmt.Errorf("Unable parse keyset. Error: %s", err)
	}

	return keyset, nil
}

func NewGRPCPublicKeyResolver(grpcClient identitypb.PublicKeyServiceClient) *GRPCPublicKeyResolver {
	return &GRPCPublicKeyResolver{client: grpcClient}
}

func authenticator(ctx context.Context, input *openapi3filter.AuthenticationInput, resolver IdentityPublicKeyResolver) error {
	if input.SecuritySchemeName != "BearerAuth" {
		return fmt.Errorf("security scheme %s != 'BearerAuth'", input.SecuritySchemeName)
	}

	request := input.RequestValidationInput.Request
	rawToken := request.Header.Get("Authorization")
	if rawToken == "" {
		return errors.New("empty bearer token")
	}

	plainToken := tokenutils.TrimTokenBearer(rawToken)

	keyset, err := resolver.GetKeys(ctx, plainToken)
	if err != nil {
		return fmt.Errorf("Not found keyset for token. Error: %s", err)
	}

	verifiedPayload, err := tokenutils.VerifyTokenByKeySet(keyset, plainToken)
	if err != nil {
		return fmt.Errorf("Not verified token. Error: %s", err)
	}

	err = tokenutils.ValidateToken(string(verifiedPayload))
	if err != nil {
		return fmt.Errorf("Invalid token. Error: %s", err)
	}

	authContext := context.WithValue(
		request.Context(),
		TOKEN_CONTEXT_VALUE,
		verifiedPayload,
	)

	newReq := request.Clone(authContext)

	input.RequestValidationInput.Request = newReq.WithContext(authContext)

	return nil
}

type IdentityPublicKeyResolver interface {
	GetKeys(context.Context, string) (jwk.Set, error)
}

func NewAsymmetricEncryptionAuthenticator(resolver IdentityPublicKeyResolver) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		if err := authenticator(ctx, input, resolver); err != nil {
			return NewSecurityError(err)
		}
		return nil
	}
}
