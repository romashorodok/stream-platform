package streamchannels

import (
	"context"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/stream_channels.openapi.yaml

type StreamChannelsService struct {
	Unimplemented
}

func (*StreamChannelsService) StreamChannelsServiceStreamChannelList(w http.ResponseWriter, r *http.Request) {
}

var _ ServerInterface = (*StreamChannelsService)(nil)

type StreamChannelServiceHandlerParams struct {
	fx.In

	lifecycle     fx.Lifecycle
	Router        *chi.Mux
	FilterOptions openapi3filter.Options
	AuthAsymm     openapi3filter.AuthenticationFunc
}

func NewStreamChannelsServiceHandler(params StreamChannelServiceHandlerParams) *StreamChannelsService {
	service := &StreamChannelsService{}

	params.lifecycle.Append(fx.OnStart(
		func(context.Context) error {
			spec, err := GetSwagger()
			if err != nil {
				return fmt.Errorf("unable get openapi spec. %s", err)
			}
			spec.Servers = nil

			params.FilterOptions.AuthenticationFunc = params.AuthAsymm
			params.Router.Use(openapi.NewOpenAPIRequestMiddleware(spec, &openapi.Options{
				Options: params.FilterOptions,
			}))

			HandlerFromMux(service, params.Router)

			return nil
		},
	))

	return service
}
