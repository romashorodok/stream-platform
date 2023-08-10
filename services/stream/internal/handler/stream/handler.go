package stream

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/service.openapi.yaml

type StreamingService struct {
	Unimplemented
}

type StreamingServiceParams struct {
	fx.In

	Lifecycle     fx.Lifecycle
	Router        *chi.Mux
	FilterOptions openapi3filter.Options
}

func NewStreaminServiceHandler(params StreamingServiceParams) *StreamingService {
	service := &StreamingService{}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			spec, err := GetSwagger()
			// NOTE: If no setup this to nil handler will not work. Why???
			spec.Servers = nil

			if err != nil {
				return fmt.Errorf("unable get openapi spec. %s", err)
			}

			params.Router.Use(openapi.NewOpenAPIRequestMiddleware(spec, &openapi.Options{
				Options: params.FilterOptions,
			}))

			HandlerFromMux(service, params.Router)

			return nil
		},
	})

	return service
}
