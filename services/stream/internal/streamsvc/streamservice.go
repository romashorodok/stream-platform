package streamsvc

import (
	"context"
	"errors"
	"log"
	"strings"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"go.uber.org/fx"
)

var (
	NotFoundActiveStream = errors.New("Not found active stream record.")
	UnableInsertStream   = errors.New("Unable insert stream.")
	UnableStopStream     = errors.New("Unable stop stream.")
	UnableDeleteStream   = errors.New("Unable delete stream.")
	StreamAlredyExists   = errors.New("Stream already exists.")
)

type StreamService struct {
	ingestController       ingestioncontrollerpb.IngestControllerServiceClient
	config                 *service.StreamSystemConfig
	activeStreamRepository *repository.ActiveStreamRepository
}

func (s *StreamService) StartIngestServer(ctx context.Context, token *auth.TokenPayload) error {
	response, err := s.ingestController.StartServer(ctx, &ingestioncontrollerpb.StartServerRequest{
		IngestTemplate: s.config.IngestTemplate,
		Deployment:     token.Sub,
		Namespace:      s.config.Namespace,
		Meta: &ingestioncontrollerpb.BroadcasterMeta{
			BroadcasterId: token.UserID.String(),
			Username:      token.Sub,
		},
	})
	if err != nil {
		log.Printf("[%s]: Unable start ingest server. Err: %s", token.Sub, err)
		_ = s.StopIngestServer(ctx, token)
		return err
	}

	if _, err := s.activeStreamRepository.InsertActiveStream(
		token.UserID,
		token.Sub,
		response.Namespace,
		response.Deployment,
	); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			log.Printf("[%s]: Stream already exist. Err: %s", token.Sub, err)
			return StreamAlredyExists
		}

		_, _ = s.ingestController.StopServer(ctx, &ingestioncontrollerpb.StopServerRequest{
			Namespace:  response.Namespace,
			Deployment: response.Deployment,
		})

		log.Printf("[%s]: Unable insert stream. Err: %s", token.Sub, err)
		return UnableInsertStream
	}

	return nil
}

func (s *StreamService) StopIngestServer(ctx context.Context, token *auth.TokenPayload) error {
	stream, err := s.activeStreamRepository.GetActiveStreamByBroadcasterId(token.UserID)
	if err != nil {
		log.Printf("[%s]: Not found active stream. Err: %s", token.Sub, err)
		return NotFoundActiveStream
	}

	if err := s.activeStreamRepository.DeleteActiveStreamByBroadcasterId(stream.BroadcasterID); err != nil {
		log.Printf("[%s]: Unable delete stream. Err: %s", token.Sub, err)
		return UnableDeleteStream
	}

	if _, err := s.ingestController.StopServer(ctx, &ingestioncontrollerpb.StopServerRequest{
		Namespace:  stream.Namespace,
		Deployment: stream.Deployment,
	}); err != nil {
		log.Printf("[%s]: Unable stop stream. Err: %s", token.Sub, err)
		return UnableStopStream
	}

	return nil
}

type StreamServiceParams struct {
	fx.In

	IngestController       ingestioncontrollerpb.IngestControllerServiceClient
	Config                 *service.StreamSystemConfig
	ActiveStreamRepository *repository.ActiveStreamRepository
}

func NewStreamService(params StreamServiceParams) *StreamService {
	return &StreamService{
		ingestController:       params.IngestController,
		config:                 params.Config,
		activeStreamRepository: params.ActiveStreamRepository,
	}
}
