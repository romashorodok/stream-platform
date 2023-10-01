package statefulstream

import (
	"context"
	"errors"
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/shutdown"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream/webrtcstatefulstream"
	"go.uber.org/fx"
)

type StatefulStream interface {
	Ingest(context.Context) error
	Destroy() error
}

var EmptyStatefulStream = (StatefulStream)(nil)

var (
	NewWebrtcStatefulStreamError = errors.New("failed create stateful stream")
)

type StatefulStreamGlobal struct {
	shutdown        *shutdown.Shutdown
	webrtcAllocator webrtcstatefulstream.WebrtcAllocatorFunc
	statefulStream  StatefulStream
}

type WebrtcTrackHandler func(*webrtc.TrackRemote, *webrtc.RTPReceiver)

func (s *StatefulStreamGlobal) HandleWebrtc(ctx context.Context) (WebrtcTrackHandler, error) {
	if s.statefulStream != EmptyStatefulStream {
		err := s.statefulStream.Destroy()
		if err != nil {
			log.Printf("[HandleWebrc]: stream destory error. Err: %s", err)
		}
		s.statefulStream = nil
	}

	stream, err := s.webrtcAllocator()
	if err != nil {
		return nil, NewWebrtcStatefulStreamError
	}

	s.statefulStream = stream

	s.shutdown.AddTask(func() {
		stream.Destroy()
	})

	go func() {
		ctx, cancel := context.WithCancel(ctx)

		s.shutdown.AddTask(func() {
			cancel()
		})

		s.statefulStream.Ingest(ctx)

		select {
		case <-ctx.Done():
			_ = s.statefulStream.Destroy()
			cancel()
		}
	}()

	return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("Received track ", track.Codec())

		switch track.Codec().RTPCodecCapability.MimeType {
		case webrtc.MimeTypeOpus, "audio/OPUS":
			stream.PipeOpusRemoteTrack(ctx, track)
		case webrtc.MimeTypeH264:
			stream.PipeH264RemoteTrack(ctx, track)
		}
	}, nil
}

func (s *StatefulStreamGlobal) GetStatefulStream() StatefulStream {
	return s.statefulStream
}

type StatefulStreamGlobalParams struct {
	fx.In

	Shutdown            *shutdown.Shutdown
	WebrtcAllocatorFunc webrtcstatefulstream.WebrtcAllocatorFunc
}

func NewStatefulStreamGlobal(params StatefulStreamGlobalParams) *StatefulStreamGlobal {
	return &StatefulStreamGlobal{
		statefulStream:  EmptyStatefulStream,
		webrtcAllocator: params.WebrtcAllocatorFunc,
		shutdown:        params.Shutdown,
	}
}
