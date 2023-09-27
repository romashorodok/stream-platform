package webrtcstatefulstream

import (
	"context"
	"io"
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/muxer"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/muxer/opusmuxer"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
	"go.uber.org/fx"
)

type WebrtcStatefulStream struct {
	audio           *webrtc.TrackLocalStaticRTP
	video           *webrtc.TrackLocalStaticRTP
	audioPipeReader *io.PipeReader
	audioPipeWriter *io.PipeWriter
	videoPipeReader *io.PipeReader
	videoPipeWriter *io.PipeWriter

	mediaProcessors []mediaprocessor.MediaProcessor
}

func (s *WebrtcStatefulStream) Ingest(ctx context.Context) error {
	defer log.Println("[StatefulStream] Ingestion process stopped")

	ingestionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, processor := range s.mediaProcessors {
		go func(processor mediaprocessor.MediaProcessor, video, audio *io.PipeReader) {
			defer cancel()
			_ = processor.Transcode(ingestionCtx, video, audio)
		}(processor, s.videoPipeReader, s.audioPipeReader)
	}

	select {
	case <-ctx.Done():
		cancel()
	case <-ingestionCtx.Done():
	}

	return nil
}

func (s *WebrtcStatefulStream) PipeH264RemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	defer log.Println("[PipeH264RemoteTrack] canceled")

	rtpMuxer := muxer.NewRtpMuxer()
	h264Muxer := muxer.NewH264Muxer()

	go rtpMuxer.Bind(track)
	go rtpMuxer.BindLocal(s.video)

	go func() {
		for {
			_, _ = io.Copy(h264Muxer, rtpMuxer)
		}
	}()

	go func() {
		for {
			_, _ = io.Copy(s.videoPipeWriter, h264Muxer)
		}
	}()

	select {
	case <-ctx.Done():
	}
}

func (s *WebrtcStatefulStream) PipeOpusRemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	defer log.Println("[PipeOpusRemoteTrack] canceled")

	opusMuxer := opusmuxer.NewOpusMuxer()

	go opusMuxer.Bind(track)

	go func() {
		for {
			_, _ = io.Copy(s.audioPipeWriter, opusMuxer)
		}
	}()

	select {
	case <-ctx.Done():
	}
}

func (s *WebrtcStatefulStream) Destroy() error {
	for _, processor := range s.mediaProcessors {
		processor.Destroy()
	}
	return nil
}

type WebrtcAllocatorFunc func() (*WebrtcStatefulStream, error)

type WebrtcAllocatorFuncParams struct {
	fx.In

	HLSMediaProcessor mediaprocessor.MediaProcessor `name:"mediaprocessor.hls.default"`
}

func NewWebrtcAllocatorFunc(params WebrtcAllocatorFuncParams) WebrtcAllocatorFunc {
	return func() (*WebrtcStatefulStream, error) {
		audio, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
		if err != nil {
			return nil, err
		}

		video, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
		if err != nil {
			return nil, err
		}

		audioPipeReader, audioPipeWriter := io.Pipe()
		videoPipeReader, videoPipeWriter := io.Pipe()

		return &WebrtcStatefulStream{
			audio:           audio,
			video:           video,
			audioPipeReader: audioPipeReader,
			audioPipeWriter: audioPipeWriter,
			videoPipeReader: videoPipeReader,
			videoPipeWriter: videoPipeWriter,
			mediaProcessors: []mediaprocessor.MediaProcessor{
				params.HLSMediaProcessor,
			},
		}, nil
	}
}
