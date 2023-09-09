package streamsvc

import (
	"log"

	streamingpb "github.com/romashorodok/stream-platform/gen/golang/streaming/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/services/stream/internal/storage/postgress/repository"
	"go.uber.org/fx"
)

type StreamStatus struct {
	stream *repository.ActiveStreamRepository
}

func (s *StreamStatus) IsRunning(auth *auth.TokenPayload) *streamingpb.StreamStatus {
	activeStream, err := s.stream.GetActiveStreamByBroadcasterId(auth.UserID)
	if err != nil {
		log.Printf("[%s] Not found active stream. Err: %s", auth.Sub, err)
		return &streamingpb.StreamStatus{Running: false, Deployed: false}
	}

	return &streamingpb.StreamStatus{Running: activeStream.Running, Deployed: activeStream.Deployed}
}

type NewStreamStatusParams struct {
	fx.In

	Stream *repository.ActiveStreamRepository
}

func NewStreamStatus(params NewStreamStatusParams) *StreamStatus {
	return &StreamStatus{
		stream: params.Stream,
	}
}
