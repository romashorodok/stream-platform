package streamchannelssvc

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/google/uuid"
	streamingpb "github.com/romashorodok/stream-platform/gen/golang/streaming/v1alpha"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"go.uber.org/fx"
)

var (
	UnableGetActiveStreamsListError = errors.New("Unable get active streams.")
	UnableGetActiveStreamError      = errors.New("Unable get active stream.")
	NotFoundGetActiveStreamError    = errors.New("Not found active stream.")
	StandaloneOnlyOperationError    = errors.New("Operation support only in standalone mode")
)

type StreamChannelsService struct {
	activeStreamRepository *repository.ActiveStreamRepository
	streamSystemConfig     *service.StreamSystemConfig
}

type egressDefault struct {
	repository.RunningActiveStreamEgress `json:"egress"`
}

type activeStreamDefault struct {
	ID       uuid.UUID `json:"active_stream_id"`
	Username string    `json:"username"`

	Egresses []egressDefault `json:"egresses"`
}

type getActiveStreamsList struct {
	Channels []activeStreamDefault `json:"channels"`
}

func (s *StreamChannelsService) GetActiveStreamsList(ctx context.Context) (*getActiveStreamsList, error) {
	channels, err := s.activeStreamRepository.GetAllRunningActiveStreams(ctx)
	if err != nil {
		return nil, UnableGetActiveStreamsListError
	}

	var result getActiveStreamsList

	for _, channel := range channels {
		model := activeStreamDefault{
			ID:       channel.ID,
			Username: channel.Username,
		}

		for _, egress := range channel.Egresses {
			model.Egresses = append(model.Egresses, egressDefault{
				RunningActiveStreamEgress: egress,
			})
		}

		result.Channels = append(result.Channels, model)

	}

	return &result, err
}

type egressWithRoute struct {
	repository.RunningActiveStreamEgress `json:"egress"`

	Route string `json:"route"`
}

func (s *StreamChannelsService) getEgressesRoutes(channel *repository.RunningActiveStream) ([]egressWithRoute, error) {
	if s.streamSystemConfig.Standalone {
		var result []egressWithRoute
		for _, egress := range channel.Egresses {
			model := egressWithRoute{
				RunningActiveStreamEgress: egress,
			}

			var egress streamingpb.StreamEgressType = streamingpb.StreamEgressType(streamingpb.StreamEgressType_value[egress.Type])

			switch egress {
			case streamingpb.StreamEgressType_STREAM_TYPE_HLS:
				model.Route = s.streamSystemConfig.IngestStandalone.IngestUri + s.streamSystemConfig.IngestStandalone.IngestHLSRoute
			case streamingpb.StreamEgressType_STREAM_TYPE_WEBRTC:
				model.Route = s.streamSystemConfig.IngestStandalone.IngestUri + s.streamSystemConfig.IngestStandalone.IngestWebrtcRoute
			}

			result = append(result, model)
		}

		return result, nil
	}

	return nil, StandaloneOnlyOperationError
}

type activeStreamWithEgressRoutes struct {
	ID       uuid.UUID `json:"active_stream_id"`
	Username string    `json:"username"`

	Egresses []egressWithRoute `json:"egresses"`
}

type getActiveStream struct {
	Channel activeStreamWithEgressRoutes `json:"channel"`
}

func (s *StreamChannelsService) GetActiveStream(ctx context.Context, username string) (*getActiveStream, error) {
	channel, err := s.activeStreamRepository.GetActiveStreamByUsername(ctx, username)
	if err != nil {
		log.Println("Unable get active stream record. Err:", err)

		if strings.HasPrefix(err.Error(), "qrm: no rows in result set") {
			return nil, NotFoundGetActiveStreamError
		}

		return nil, UnableGetActiveStreamError
	}

	egressesWithRoutes, err := s.getEgressesRoutes(channel)
	if err != nil {
		log.Println("Unable get active stream routes. Err:", err)
		return nil, err
	}

	return &getActiveStream{
		Channel: activeStreamWithEgressRoutes{
			ID:       channel.ID,
			Username: channel.Username,
			Egresses: egressesWithRoutes,
		},
	}, nil
}

type StreamChannelsServiceParams struct {
	fx.In

	ActiveStreamRepository *repository.ActiveStreamRepository
	StreamSystemConfig     *service.StreamSystemConfig
}

func NewStreamChannelsService(params StreamChannelsServiceParams) *StreamChannelsService {
	return &StreamChannelsService{
		activeStreamRepository: params.ActiveStreamRepository,
		streamSystemConfig:     params.StreamSystemConfig,
	}
}
