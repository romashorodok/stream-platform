package streamchannelssvc

import (
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

type StreamChannelsService struct {
	activeStreamRepository *repository.ActiveStreamRepository
}

func (s *StreamChannelsService) GetActiveStreams() {
	s.activeStreamRepository.GetAllActiveStreams()
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
