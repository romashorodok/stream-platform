package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/service.openapi.yaml

type StreamingService struct {
	Unimplemented

	ingestController ingestioncontrollerpb.IngestControllerServiceClient
	activeStreamRepo *repository.ActiveStreamRepository
	refreshTokenAuth *auth.RefreshTokenAuthenticator
	streamStatus     *streamsvc.StreamStatus
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

	response, err := s.ingestController.StartServer(
		r.Context(),
		&ingestioncontrollerpb.StartServerRequest{
			IngestTemplate: "golang-ingest-template",
			Deployment:     token.Sub,
			Namespace:      "default",
			Meta: &ingestioncontrollerpb.BroadcasterMeta{
				BroadcasterId: token.UserID.String(),
				Username:      token.Sub,
			},
		},
	)

	if err != nil {
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.Unavailable:
				httputils.WriteErrorResponse(w, http.StatusServiceUnavailable, "Ingest operator is not available.", e.Message())

			case codes.Aborted:
				httputils.WriteErrorResponse(w, http.StatusConflict, "Ingest server already running or something went wrong on ingest operator.", e.Message())

			default:
				httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Something went wrong on ingest operator", e.Message())

			}
			return
		}

		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Ingest internal error.", err.Error())
		return
	}

	_, err = s.activeStreamRepo.InsertActiveStream(
		token.UserID,
		token.Sub,
		response.Namespace,
		response.Deployment,
	)

	if err != nil {
		log.Printf("Found error when store active stream record. Err: %s", err)

		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			httputils.WriteErrorResponse(w, http.StatusAccepted, "Already running. Stream restarting... Err:", err.Error())
			return
		}

		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable store active stream record", err.Error())
		return
	}
}

func (s *StreamingService) StreamingServiceStreamStop(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user_id from token

	token, err := auth.WithTokenPayload(r.Context())
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionFailed, "Not found user token payload", err.Error())
		return
	}
	log.Println(token)

	stream, err := s.activeStreamRepo.GetActiveStreamByBroadcasterId(token.UserID)

	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusNotFound, "Not found active stream record.", err.Error())
		return
	}

	if _, err = s.ingestController.StopServer(r.Context(), &ingestioncontrollerpb.StopServerRequest{
		Namespace:  stream.Namespace,
		Deployment: stream.Deployment,
	}); err != nil {
		httputils.WriteErrorResponse(w, http.StatusNotFound, "Unable stop ingest or not found it.", err.Error())
	}

	// if err := s.activeStreamRepo.DeleteActiveStreamByBroadcasterId(stream.BroadcasterID); err != nil {
	// 	log.Println("unable to delete old stream")
	// }
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
}

func NewStreaminServiceHandler(params StreamingServiceParams) *StreamingService {
	service := &StreamingService{
		ingestController: params.IngestController,
		activeStreamRepo: params.ActiveStreamRepo,
		refreshTokenAuth: params.RefreshTokenAuth,
		streamStatus:     params.StreamStatus,
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
