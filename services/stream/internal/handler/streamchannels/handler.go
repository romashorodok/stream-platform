package streamchannels

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/openapi3utils"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamchannelssvc"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/stream_channels.openapi.yaml

type handler struct {
	Unimplemented

	handlerSpecValidator openapi3utils.HandlerSpecValidator
	streamChannels       *streamchannelssvc.StreamChannelsService
}

var _ ServerInterface = (*handler)(nil)
var _ httputils.HttpHandler = (*handler)(nil)

func (hand *handler) StreamChannelsServiceStreamChannelList(w http.ResponseWriter, r *http.Request) {
	result, err := hand.streamChannels.GetActiveStreamsList(r.Context())
	if err != nil {
		unableGetActiveStreamsList(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

func (hand *handler) StreamChannelsServiceGetStreamChannel(w http.ResponseWriter, r *http.Request, username string) {
	result, err := hand.streamChannels.GetActiveStream(r.Context(), username)
	if err != nil {
		unableGetActiveStream(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

func (h *handler) GetOption() httputils.HttpHandlerOption {
	return func(hand http.Handler) {
		switch hand.(type) {
		case *chi.Mux:
			mux := hand.(*chi.Mux)

			spec, err := GetSwagger()
			if err != nil {
				log.Panicf("unable get openapi spec for streamchannels.handler.Err: %s", err)
			}
			spec.Servers = nil

			HandlerWithOptions(h, ChiServerOptions{
				BaseRouter: mux,
				Middlewares: []MiddlewareFunc{
					h.handlerSpecValidator(spec),
					httputils.JsonMiddleware(),
				},
			})
		default:
			panic("unsupported streamchannels handler")
		}
	}
}

type StreamChannelServiceHandlerParams struct {
	fx.In

	HandlerSpecValidator openapi3utils.HandlerSpecValidator
	StreamChannels       *streamchannelssvc.StreamChannelsService
}

func NewStreamChannelsServiceHandler(params StreamChannelServiceHandlerParams) *handler {
	return &handler{
		streamChannels:       params.StreamChannels,
		handlerSpecValidator: params.HandlerSpecValidator,
	}
}
