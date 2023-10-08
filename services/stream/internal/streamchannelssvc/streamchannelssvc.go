package streamchannelssvc

import (
	"context"
	"errors"

	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

var (
	UnableGetActiveStreamsListError = errors.New("Unable get active streams.")
)

type StreamChannelsService struct {
	activeStreamRepository *repository.ActiveStreamRepository
}

func (s *StreamChannelsService) GetActiveStreamsList(ctx context.Context) ([]repository.RunningActiveStreams, error) {
	result, err := s.activeStreamRepository.GetAllRunningActiveStreams(ctx)
	if err != nil {
		return nil, UnableGetActiveStreamsListError
	}

	return result, err
}

type StreamChannelsServiceParams struct {
	fx.In

	ActiveStreamRepository *repository.ActiveStreamRepository
}

func NewStreamChannelsService(params StreamChannelsServiceParams) *StreamChannelsService {
	return &StreamChannelsService{
		activeStreamRepository: params.ActiveStreamRepository,
	}
}
