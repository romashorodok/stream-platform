package service

import (
	"log"
	"net"

	"github.com/getkin/kin-openapi/openapi3filter"
	identitypb "github.com/romashorodok/stream-platform/gen/golang/identity/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"github.com/romashorodok/stream-platform/pkg/openapi3utils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PublicKeyClientConfig struct {
	Host string
	Port string
}

func NewPublicKeyClientConfig() *PublicKeyClientConfig {
	return &PublicKeyClientConfig{
		Host: envutils.Env(
			variables.STREAM_IDENTITY_GRPC_PUBLIC_KEY_HOST,
			variables.STREAM_IDENTITY_GRPC_PUBLIC_KEY_HOST_DEFAULT,
		),
		Port: envutils.Env(
			variables.STREAM_IDENTITY_GRPC_PUBLIC_KEY_PORT,
			variables.STREAM_IDENTITY_GRPC_PUBLIC_KEY_PORT_DEFAULT,
		),
	}
}

type PublicKeyClientParams struct {
	fx.In

	Config *PublicKeyClientConfig
}

func NewPublicKeyClient(params PublicKeyClientParams) identitypb.PublicKeyServiceClient {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(
		net.JoinHostPort(
			params.Config.Host,
			params.Config.Port,
		), opts...)

	if err != nil {
		log.Panicln("Failed to connect to audio service. Error:", err)
	}

	return identitypb.NewPublicKeyServiceClient(conn)
}

type IdentityPublicKeyResolverParams struct {
	fx.In

	Client identitypb.PublicKeyServiceClient
}

func NewIdentityPublicKeyResolver(params IdentityPublicKeyResolverParams) auth.IdentityPublicKeyResolver {
	return auth.NewGRPCPublicKeyResolver(params.Client)
}

func NewStreamHttpConfig() *StreamHttpConfig {
	return &StreamHttpConfig{
		Port: envutils.Env(variables.STREAM_HTTP_PORT, variables.STREAM_HTTP_PORT_DEFAULT),
		Host: envutils.Env(variables.STREAM_HTTP_HOST, variables.STREAM_HTTP_HOST_DEFAULT),
	}
}

type AsymmetricEncryptionAuthenticatorParams struct {
	fx.In

	Resolver auth.IdentityPublicKeyResolver
}

func NewAsymmetricEncryptionAuthenticator(params AsymmetricEncryptionAuthenticatorParams) openapi3filter.AuthenticationFunc {
	return auth.NewAsymmetricEncryptionAuthenticator(params.Resolver)
}

type OpenAPI3FilterOptionsParams struct {
	fx.In

	AuthFunc openapi3filter.AuthenticationFunc
}

func NewOpenAPI3FilterOptions(params OpenAPI3FilterOptionsParams) openapi3filter.Options {
	return openapi3filter.Options{
		AuthenticationFunc: params.AuthFunc,
		MultiError:         true,
	}
}

type NewSpecOptionsHandlerConstructorParams struct {
	fx.In

	Options openapi3filter.Options
}

func NewSpecOptionsHandlerConstructor(params NewSpecOptionsHandlerConstructorParams) openapi3utils.HandlerSpecValidator {
	return func(spec *openapi3utils.Spec) openapi3utils.HandlerFunc {
		return openapi.NewOpenAPIRequestMiddleware(spec, &openapi.Options{
			Options: params.Options,
		})
	}
}

var OpenapiModule = fx.Module("openapi",
	fx.Provide(
		NewPublicKeyClientConfig,
		NewPublicKeyClient,
		NewAsymmetricEncryptionAuthenticator,
		NewOpenAPI3FilterOptions,
		fx.Private,
	),
	fx.Provide(
		NewIdentityPublicKeyResolver,
		NewSpecOptionsHandlerConstructor,
	),
)
