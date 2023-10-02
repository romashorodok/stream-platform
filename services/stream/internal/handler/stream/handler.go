package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamsvc"
	"go.uber.org/fx"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/service.openapi.yaml

type StreamingService struct {
	Unimplemented

	refreshTokenAuth *auth.RefreshTokenAuthenticator
	streamStatus     *streamsvc.StreamStatus
	streamService    *streamsvc.StreamService
	nats             *nats.Conn
}

var _ ServerInterface = (*StreamingService)(nil)

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

type StreamingServiceParams struct {
	fx.In

	Lifecycle        fx.Lifecycle
	Router           *chi.Mux
	FilterOptions    openapi3filter.Options
	IngestController ingestioncontrollerpb.IngestControllerServiceClient
	ActiveStreamRepo *repository.ActiveStreamRepository
	StreamStatus     *streamsvc.StreamStatus
	RefreshTokenAuth *auth.RefreshTokenAuthenticator
	AuthAsymm        openapi3filter.AuthenticationFunc
	Nats             *nats.Conn
	StreamService    *streamsvc.StreamService
}

func NewStreaminServiceHandler(params StreamingServiceParams) *StreamingService {
	service := &StreamingService{
		refreshTokenAuth: params.RefreshTokenAuth,
		streamStatus:     params.StreamStatus,
		streamService:    params.StreamService,
		nats:             params.Nats,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			spec, err := GetSwagger()
			// NOTE: If don't do this validation will not work.
			spec.Servers = nil

			if err != nil {
				return fmt.Errorf("unable get openapi spec. %s", err)
			}

			params.FilterOptions.AuthenticationFunc = params.AuthAsymm
			params.Router.Use(openapi.NewOpenAPIRequestMiddleware(spec, &openapi.Options{
				Options: params.FilterOptions,
			}))

			HandlerFromMux(service, params.Router)

			return nil
		},
	})

	return service
}
