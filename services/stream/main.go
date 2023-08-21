package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/services/stream/internal/handler/stream"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "github.com/lib/pq"
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
		Username: "user",
		Password: "password",
		Host:     "0.0.0.0",
		Port:     "5432",
		Database: "postgres",
	}
}

type HTTPConfig struct {
	Port string
	Host string

	IngestOperatorHost string
	IngestOperatorPort string
}

func NewHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Port: "8082",
		Host: "0.0.0.0",

		IngestOperatorHost: "0.0.0.0",
		IngestOperatorPort: "9191",
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

	Config *HTTPConfig
}

func WithIngestOperatorClient(params IngestOperatorClientParams) ingestioncontrollerpb.IngestControllerServiceClient {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.Dial(
		net.JoinHostPort(
			params.Config.IngestOperatorHost,
			params.Config.IngestOperatorPort,
		), opts...)

	if err != nil {
		log.Panicln("Failed to connect to audio service. Error:", err)
	}

	client := ingestioncontrollerpb.NewIngestControllerServiceClient(conn)

	return client
}

type DatabaseConnectionParams struct {
	fx.In

	Dconf *DatabaseConfig
}

func WithDatabaseConnection(params DatabaseConnectionParams) *sql.DB {
	uri := params.Dconf.GetURI()
	db, err := sql.Open(params.Dconf.Driver, uri + "?sslmode=disable")

	if err != nil {
		log.Panicf("Unable connect to database %s. Error: %s \n", uri, err)
	}

	return db
}

func main() {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	fx.New(
		fx.Provide(
			GetRouter,
			fx.Annotate(
				GetRouter,
				fx.As(new(http.Handler)),
			),

			WithOpenAPI3FilterOptions,
			WithIngestOperatorClient,
			WithDatabaseConnection,

			repository.NewActiveStreamRepository,

			NewHTTPConfig,
			NewDatabaseConfig,
			NewHTTPServer,
		),
		fx.Invoke(stream.NewStreaminServiceHandler),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
