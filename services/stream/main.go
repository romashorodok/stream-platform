package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"github.com/romashorodok/stream-platform/services/stream/internal/handler/stream"
	"github.com/romashorodok/stream-platform/services/stream/internal/handler/streamchannels"
	"github.com/romashorodok/stream-platform/services/stream/internal/ingestcontroller"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamchannelssvc"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamsvc"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ingestworker "github.com/romashorodok/stream-platform/services/stream/internal/workers/ingest"
)

const (
	DATABASE_HOST_DEFAULT     = "0.0.0.0"
	DATABASE_PORT_DEFAULT     = "5432"
	DATABASE_USERNAME_DEFAULT = "user"
	DATABASE_PASSWORD_DEFAULT = "password"
	DATABASE_NAME_DEFAULT     = "postgres"

	INGEST_OPERATOR_HOST_DEFAULT = "0.0.0.0"
	INGEST_OPERATOR_PORT_DEFAULT = "9191"
)

const (
	DATABASE_HOST_VAR     = "DATABASE_HOST"
	DATABASE_PORT_VAR     = "DATABASE_PORT"
	DATABASE_USERNAME_VAR = "DATABASE_USERNAME"
	DATABASE_PASSWORD_VAR = "DATABASE_PASSWORD"
	DATABASE_NAME_VAR     = "DATABASE_NAME"

	INGEST_OPERATOR_HOST_VAR = "INGEST_OPERATOR_HOST"
	INGEST_OPERATOR_PORT_VAR = "INGEST_OPERATOR_PORT"
)

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

	fx.New(
		service.ServerModule,
		service.OpenapiModule,
		service.StreamSystemModule,

		fx.Provide(httputils.AsHttpHandler(stream.NewStreaminServiceHandler)),
		fx.Provide(httputils.AsHttpHandler(streamchannels.NewStreamChannelsServiceHandler)),

		fx.Invoke(service.StartStreamHttp),

		fx.Provide(
			WithRefreshTokenAuthenticator,

			NewIngestOperatorConfig,
			WithIngestOperatorClient,

			WithDatabaseConnection,

			NewNatsConfig,
			WithNatsConnection,
			NewNatsJetstream,
			streamchannelssvc.NewStreamChannelsService,

			repository.NewActiveStreamRepository,
			repository.NewStreamEgressRepository,
			streamsvc.NewStreamStatus,
			streamsvc.NewStreamService,

			NewDatabaseConfig,
		),
		fx.Invoke(ingestworker.StartIngestStatusWorker),
		fx.Invoke(ingestworker.StartIngestDestroyedWorker),
	).Run()
}
