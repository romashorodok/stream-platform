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
	_ "github.com/lib/pq"
	"github.com/romashorodok/stream-platform/pkg/auth"
	identitygrpc "github.com/romashorodok/stream-platform/services/identity/internal/grpc/v1alpha/identity"
	"github.com/romashorodok/stream-platform/services/identity/internal/handler/v1alpha/identity"
	"github.com/romashorodok/stream-platform/services/identity/internal/security"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/privatekey"
	"github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/refreshtoken"
	userrepo "github.com/romashorodok/stream-platform/services/identity/internal/storage/postgres/user"
	"github.com/romashorodok/stream-platform/services/identity/internal/user"
	"go.uber.org/fx"
	"google.golang.org/grpc"
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
		OnStart: func(context.Context) error {
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

type HTTPConfig struct {
	Port string
	Host string
}

func NewHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Port: "8083",
		Host: "0.0.0.0",
	}
}

var router = chi.NewRouter()

func GetRouter() *chi.Mux {
	return router
}

type GRPCServerParams struct {
	fx.In

	Config    *GRPCConfig
	Lifecycle fx.Lifecycle

	Options []grpc.ServerOption `group:"grpc.ServerOption"`
}

func NewGRPCServer(params GRPCServerParams) *grpc.Server {
	server := grpc.NewServer(params.Options...)
	addr := net.JoinHostPort(params.Config.Host, params.Config.Port)

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			go server.Serve(ln)

			return nil
		},
	})

	return server
}

type GRPCConfig struct {
	Port string
	Host string
}

func NewGRPConfig() *GRPCConfig {
	return &GRPCConfig{
		Host: "localhost",
		Port: "9093",
	}
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
		Port:     "5433",
		Database: "postgres",
	}
}

type DatabaseConnectionParams struct {
	fx.In

	Dconf     *DatabaseConfig
	Lifecycle fx.Lifecycle
}

func WithDatabaseConnection(params DatabaseConnectionParams) *sql.DB {
	uri := params.Dconf.GetURI()

	db, err := sql.Open(params.Dconf.Driver, uri+"?sslmode=disable")

	if err != nil {
		log.Panicf("Unable connect to database %s. Error: %s \n", uri, err)
	}

	params.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			db.Close()
			return nil
		},
	})

	return db
}

func WithOpenAPI3FilterOptions() openapi3filter.Options {
	authProvider, _ := auth.NewFakeAuthenticator()
	options := openapi3filter.Options{
		AuthenticationFunc: auth.NewAuthenticator(authProvider),
		MultiError:         true,
	}
	return options
}

func AsGRPCServerOption(f interface{}) interface{} {
	return fx.Annotate(
		f,
		fx.As(new(grpc.ServerOption)),
		fx.ResultTags(`group:"grpc.ServerOption"`),
	)
}

func main() {
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	fx.New(
		fx.Provide(
			GetRouter,
			fx.Annotate(
				GetRouter,
				fx.As(new(http.Handler)),
			),

			NewDatabaseConfig,

			WithDatabaseConnection,
			WithOpenAPI3FilterOptions,

			privatekey.NewPrivateKeyRepositroy,
			userrepo.NewUserRepository,
			security.NewSecurityService,
			refreshtoken.NewRefreshTokenRepository,

			user.NewUserService,

			NewHTTPConfig,
			NewHTTPServer,

			AsGRPCServerOption(identitygrpc.NewPublicKeyListInterceptor),

			NewGRPConfig,
			NewGRPCServer,
		),
		fx.Invoke(identity.NewIdentityHandler),
		fx.Invoke(identitygrpc.NewPublicKeyGRPCService),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
