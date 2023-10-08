package service

import (
	"context"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"go.uber.org/fx"
)

type StreamHttpConfig struct {
	Port string
	Host string
}

func (s StreamHttpConfig) GetAddr() string {
	return net.JoinHostPort(s.Host, s.Port)
}

var router = chi.NewMux()

func Router() *chi.Mux {
	return router
}

type StreamHttpParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *StreamHttpConfig
	Handler   *chi.Mux
	// watch httputils.httpHandlerGroup if it's []
	Handlers []httputils.HttpHandler `group:"http.handler"`
}

func StartStreamHttp(params StreamHttpParams) {
	server := &http.Server{
		Addr:    params.Config.GetAddr(),
		Handler: params.Handler,
	}

	params.Handler.Use(
		cors.Handler(cors.Options{
			AllowedOrigins: []string{"https://*", "http://*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		}),
	)

	for _, handler := range params.Handlers {
		handler.GetOption()(server.Handler)
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(err)
	}

	go server.Serve(ln)

	params.Lifecycle.Append(
		fx.StopHook(func(ctx context.Context) error {
			return server.Shutdown(ctx)
		}),
	)
}

var ServerModule = fx.Module("server",
	fx.Provide(
		Router,
		fx.Annotate(
			Router,
			fx.As(new(http.Handler)),
			fx.From(new(chi.Mux)),
		),
		NewStreamHttpConfig,
	),
)
