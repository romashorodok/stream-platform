package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest --config=handler.cfg.yaml ../../../../../gen/openapiv3/streaming/v1alpha/service.openapi.yaml

type StreamingService struct {
	Unimplemented

	ingestController ingestioncontrollerpb.IngestControllerServiceClient
	activeStreamRepo *repository.ActiveStreamRepository
}

var _ ServerInterface = (*StreamingService)(nil)

var (
	username        = "testusername"
	namespace       = "default"
	deployment_name = username
	user_id         = 1
)

func (s *StreamingService) StreamingServiceStreamStart(w http.ResponseWriter, r *http.Request) {
	var request StreamStartRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Unable deserialize request body. Error: %s",
				err.Error(),
			),
		})
		return
	}

	response, err := s.ingestController.StartServer(
		r.Context(),
		&ingestioncontrollerpb.StartServerRequest{
			IngestTemplate: request.IngestTemplate,
			Deployment:     username,
		},
	)

	// TODO: already running case

	if err != nil {
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.Unavailable:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)

				json.NewEncoder(w).Encode(ErrorResponse{
					Message: fmt.Sprintf(
						"Ingest operator is not available. Error: %s",
						e.Message(),
					),
				})

			}

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Cannot start ingest server. Error: %s",
				err.Error(),
			),
		})
		return
	}

	// TODO: get from here dns or something
	_ = response

	model, err := s.activeStreamRepo.InsertActiveStream(
		uint32(user_id),
		namespace,
		deployment_name,
	)

	if err != nil {
		_, _ = s.ingestController.StopServer(r.Context(), &ingestioncontrollerpb.StopServerRequest{
			Namespace:  namespace,
			Deployment: username,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Unable store active stream record. Error: %s",
				err.Error(),
			),
		})
		return
	}

	_ = model
}

func (s *StreamingService) StreamingServiceStreamStop(w http.ResponseWriter, r *http.Request) {
	// TODO: Get user_id from token

	stream, err := s.activeStreamRepo.GetActiveStreamByBroadcasterId(uint32(user_id))

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Not found active stream record. Error: %s",
				err.Error(),
			),
		})
		return
	}

	if _, err = s.ingestController.StopServer(r.Context(), &ingestioncontrollerpb.StopServerRequest{
		Namespace:  stream.Namespace,
		Deployment: stream.Deployment,
	}); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Unable to stop ingest or not found it. Error: %s",
				err.Error(),
			),
		})
		return
	}

	if err = s.activeStreamRepo.DeleteActiveStreamByBroadcasterId(uint32(stream.BroadcasterID)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		json.NewEncoder(w).Encode(ErrorResponse{
			Message: fmt.Sprintf(
				"Cannot delete active stream record. Error: %s",
				err.Error(),
			),
		})
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
}

func NewStreaminServiceHandler(params StreamingServiceParams) *StreamingService {
	service := &StreamingService{
		ingestController: params.IngestController,
		activeStreamRepo: params.ActiveStreamRepo,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			spec, err := GetSwagger()
			// NOTE: If don't do this validation will not work.
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
