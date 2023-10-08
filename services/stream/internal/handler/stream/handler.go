package stream

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/openapi3utils"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamsvc"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/service.openapi.yaml

type StreamingService struct {
	Unimplemented

	handlerSpecValidator openapi3utils.HandlerSpecValidator
	refreshTokenAuth     *auth.RefreshTokenAuthenticator
	streamStatus         *streamsvc.StreamStatus
	streamService        *streamsvc.StreamService
	nats                 *nats.Conn
}

var _ ServerInterface = (*StreamingService)(nil)
var _ httputils.HttpHandler = (*StreamingService)(nil)

func (s *StreamingService) StreamingServiceStreamStart(w http.ResponseWriter, r *http.Request) {
	var request StreamStartRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionFailed, "Unable deserialize request body.", err.Error())
		return
	}

	token, err := auth.WithTokenPayload(r.Context())
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Not found user token payload.", err.Error())
		return
	}

	if err := s.streamService.StartIngestServer(r.Context(), token); err != nil {
		unableStartIngestServerErrorHandler(w, err)
		return
	}
}

func (s *StreamingService) StreamingServiceStreamStop(w http.ResponseWriter, r *http.Request) {
	token, err := auth.WithTokenPayload(r.Context())
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionFailed, "Not found user token payload", err.Error())
		return
	}

	if err := s.streamService.StopIngestServer(r.Context(), token); err != nil {
		unableStopIngestServerErrorHandler(w, err)
		return
	}
}

func (h *StreamingService) GetOption() httputils.HttpHandlerOption {
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
				BaseRouter:  mux,
				Middlewares: []MiddlewareFunc{h.handlerSpecValidator(spec)},
			})
		default:
			panic("unsupported streamchannels handler")
		}
	}
}

type StreamingServiceParams struct {
	fx.In

	HandlerSpecValidator openapi3utils.HandlerSpecValidator
	StreamStatus         *streamsvc.StreamStatus
	RefreshTokenAuth     *auth.RefreshTokenAuthenticator
	Nats                 *nats.Conn
	StreamService        *streamsvc.StreamService
}

func NewStreaminServiceHandler(params StreamingServiceParams) *StreamingService {
	return &StreamingService{
		handlerSpecValidator: params.HandlerSpecValidator,
		refreshTokenAuth:     params.RefreshTokenAuth,
		streamStatus:         params.StreamStatus,
		streamService:        params.StreamService,
		nats:                 params.Nats,
	}
}
