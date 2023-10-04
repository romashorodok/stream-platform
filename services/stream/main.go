package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	identitypb "github.com/romashorodok/stream-platform/gen/golang/identity/v1alpha"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"github.com/romashorodok/stream-platform/services/stream/internal/handler/stream"
	"github.com/romashorodok/stream-platform/services/stream/internal/ingestcontroller"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamsvc"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ingestworker "github.com/romashorodok/stream-platform/services/stream/internal/workers/ingest"
)

const (
	HTTP_HOST_DEFAULT = "0.0.0.0"
	HTTP_PORT_DEFAULT = "8082"

	DATABASE_HOST_DEFAULT     = "0.0.0.0"
	DATABASE_PORT_DEFAULT     = "5432"
	DATABASE_USERNAME_DEFAULT = "user"
	DATABASE_PASSWORD_DEFAULT = "password"
	DATABASE_NAME_DEFAULT     = "postgres"

	INGEST_OPERATOR_HOST_DEFAULT = "0.0.0.0"
	INGEST_OPERATOR_PORT_DEFAULT = "9191"

	IDENTITY_PUBLIC_KEY_HOST_DEFAULT = "0.0.0.0"
	IDENTITY_PUBLIC_KEY_PORT_DEFAULT = "9093"
)

const (
	HTTP_HOST_VAR = "HTTP_HOST"
	HTTP_PORT_VAR = "HTTP_PORT"

	DATABASE_HOST_VAR     = "DATABASE_HOST"
	DATABASE_PORT_VAR     = "DATABASE_PORT"
	DATABASE_USERNAME_VAR = "DATABASE_USERNAME"
	DATABASE_PASSWORD_VAR = "DATABASE_PASSWORD"
	DATABASE_NAME_VAR     = "DATABASE_NAME"

	INGEST_OPERATOR_HOST_VAR = "INGEST_OPERATOR_HOST"
	INGEST_OPERATOR_PORT_VAR = "INGEST_OPERATOR_PORT"

	IDENTITY_PUBLIC_KEY_HOST_VAR = "IDENTITY_PUBLIC_KEY_HOST"
	IDENTITY_PUBLIC_KEY_PORT_VAR = "IDENTITY_PUBLIC_KEY_PORT"
)

type HTTPServerParams struct {
	fx.In

	Config    *HTTPConfig
	Handler   http.Handler
	Lifecycle fx.Lifecycle
}

func NewHTTPServer(params HTTPServerParams) *http.Server {
	server := &http.Server{
		Addr:    net.JoinHostPort(params.Config.Host, params.Config.Port),
		Handler: params.Handler,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}
			go server.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})

	return server
}

type DatabaseConfig struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
	Driver   string
}

func (dconf *DatabaseConfig) GetURI() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		dconf.Driver,
		dconf.Username,
		dconf.Password,
		dconf.Host,
		dconf.Port,
		dconf.Database,
	)
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:   "postgres",
		Username: envutils.Env(DATABASE_USERNAME_VAR, DATABASE_USERNAME_DEFAULT),
		Password: envutils.Env(DATABASE_PASSWORD_VAR, DATABASE_PASSWORD_DEFAULT),
		Host:     envutils.Env(DATABASE_HOST_VAR, DATABASE_HOST_DEFAULT),
		Port:     envutils.Env(DATABASE_PORT_VAR, DATABASE_PORT_DEFAULT),
		Database: envutils.Env(DATABASE_NAME_VAR, DATABASE_NAME_DEFAULT),
	}
}

type HTTPConfig struct {
	Port string
	Host string
}

func NewHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Port: envutils.Env(HTTP_PORT_VAR, HTTP_PORT_DEFAULT),
		Host: envutils.Env(HTTP_HOST_VAR, HTTP_HOST_DEFAULT),
	}
}

type IngestOperatorConfig struct {
	Port string
	Host string
}

func NewIngestOperatorConfig() *IngestOperatorConfig {
	return &IngestOperatorConfig{
		Port: envutils.Env(INGEST_OPERATOR_PORT_VAR, INGEST_OPERATOR_PORT_DEFAULT),
		Host: envutils.Env(INGEST_OPERATOR_HOST_VAR, INGEST_OPERATOR_HOST_DEFAULT),
	}
}

var router = chi.NewRouter()

func GetRouter() *chi.Mux {
	return router
}

func WithOpenAPI3FilterOptions() openapi3filter.Options {
	authProvider, _ := auth.NewFakeAuthenticator()
	options := openapi3filter.Options{
		AuthenticationFunc: auth.NewAuthenticator(authProvider),
		MultiError:         true,
	}
	return options
}

type IngestOperatorClientParams struct {
	fx.In

	NatsJs       nats.JetStreamContext
	NatsConn     *nats.Conn
	SystemConfig *service.StreamSystemConfig
	Config       *IngestOperatorConfig
}

func WithIngestOperatorClient(params IngestOperatorClientParams) ingestioncontrollerpb.IngestControllerServiceClient {
	if params.SystemConfig.Standalone {
		return ingestcontroller.NewStandaloneIngestControllerStub(ingestcontroller.StandaloneIngestControllerStubParams{
			Config: params.SystemConfig,
			Conn:   params.NatsConn,
			JS:     params.NatsJs,
		})
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(
		net.JoinHostPort(
			params.Config.Host,
			params.Config.Port,
		), opts...)

	if err != nil {
		log.Panicln("Failed to connect to audio service. Error:", err)
	}

	return ingestioncontrollerpb.NewIngestControllerServiceClient(conn)
}

type PublicKeyClientConfig struct {
	Host string
	Port string
}

func NewPublicKeyClientConfig() *PublicKeyClientConfig {
	return &PublicKeyClientConfig{
		Port: envutils.Env(IDENTITY_PUBLIC_KEY_PORT_VAR, IDENTITY_PUBLIC_KEY_PORT_DEFAULT),
		Host: envutils.Env(IDENTITY_PUBLIC_KEY_HOST_VAR, IDENTITY_PUBLIC_KEY_HOST_DEFAULT),
	}
}

type PublicKeyClientParams struct {
	fx.In

	Config *PublicKeyClientConfig
}

func WithPublicKeyClient(params PublicKeyClientParams) identitypb.PublicKeyServiceClient {
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

type DatabaseConnectionParams struct {
	fx.In

	Dconf *DatabaseConfig
}

func WithDatabaseConnection(params DatabaseConnectionParams) *sql.DB {
	uri := params.Dconf.GetURI()
	db, err := sql.Open(params.Dconf.Driver, uri+"?sslmode=disable")

	if err != nil {
		log.Panicf("Unable connect to database %s. Error: %s \n", uri, err)
	}

	return db
}

type IdentityPublicKeyResolverParams struct {
	fx.In

	Client identitypb.PublicKeyServiceClient
}

func WithIdentityPublicKeyResolver(params IdentityPublicKeyResolverParams) auth.IdentityPublicKeyResolver {
	return auth.NewGRPCPublicKeyResolver(params.Client)
}

type AsymmetricEncryptionAuthenticatorParams struct {
	fx.In

	Resolver auth.IdentityPublicKeyResolver
}

func WithAsymmetricEncryptionAuthenticator(params AsymmetricEncryptionAuthenticatorParams) openapi3filter.AuthenticationFunc {
	return auth.NewAsymmetricEncryptionAuthenticator(params.Resolver)
}

type RefreshTokenAuthenticatorParams struct {
	fx.In

	Resolver auth.IdentityPublicKeyResolver
}

func WithRefreshTokenAuthenticator(params RefreshTokenAuthenticatorParams) *auth.RefreshTokenAuthenticator {
	return auth.NewRefreshTokenAuthenticator(params.Resolver)
}

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{
		Host: envutils.Env(variables.NATS_HOST, variables.NATS_HOST_DEFAULT),
		Port: envutils.Env(variables.NATS_PORT, variables.NATS_PORT_DEFAULT),
	}
}

type NatsConnectionParams struct {
	fx.In

	Config *NatsConfig
}

func WithNatsConnection(params NatsConnectionParams) *nats.Conn {
	conn, err := nats.Connect(params.Config.GetUrl())
	if err != nil {
		log.Panicf("Unable start nats connection. Err: %s", err)
		os.Exit(1)
	}

	return conn
}

type NatsJetstreamParams struct {
	fx.In

	Conn *nats.Conn
}

func NewNatsJetstream(params NatsJetstreamParams) nats.JetStreamContext {
	js, err := params.Conn.JetStream()
	if err != nil {
		log.Panicf("Unable start nats jetstream connection. Err: %s", err)
		os.Exit(1)
	}

	js.AddStream(subject.INGEST_DESTROYING_STREAM_CONFIG)

	return js
}

func main() {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	fx.New(
		service.StreamSystemModule,

		fx.Provide(
			GetRouter,
			fx.Annotate(
				GetRouter,
				fx.As(new(http.Handler)),
			),

			WithOpenAPI3FilterOptions,
			WithIngestOperatorClient,
			WithPublicKeyClient,
			WithDatabaseConnection,
			WithIdentityPublicKeyResolver,
			WithAsymmetricEncryptionAuthenticator,
			WithRefreshTokenAuthenticator,

			NewNatsConfig,
			WithNatsConnection,
			NewNatsJetstream,

			repository.NewActiveStreamRepository,
			repository.NewStreamEgressRepository,
			streamsvc.NewStreamStatus,
			streamsvc.NewStreamService,

			NewHTTPConfig,
			NewDatabaseConfig,
			NewIngestOperatorConfig,
			NewPublicKeyClientConfig,

			NewHTTPServer,
		),
		fx.Invoke(stream.NewStreaminServiceHandler),
		fx.Invoke(func(*http.Server) {}),
		fx.Invoke(ingestworker.StartIngestStatusWorker),
		fx.Invoke(ingestworker.StartIngestDestroyedWorker),
	).Run()
}
