package webrtcstatefulstream

import (
	"context"
	"io"
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/h264"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/opus"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/rtp"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/vp8"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
	"go.uber.org/fx"
)

type WebrtcStatefulStream struct {
	Audio           *webrtc.TrackLocalStaticRTP
	Video           *webrtc.TrackLocalStaticRTP
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

	h264 := media.NewMuxerBuilder(rtp.NewRtpToRtpMuxerWriter(),
		h264.NewRtpToH264MediaWriter(s.videoPipeWriter),
	)

	rtp := media.NewDemuxerBuilder(rtp.NewRtpTrackDemuxerReader(track),
		rtp.NewRtpTrackWriter(s.Video),
		media.NewTargetMediaWriter(h264),
	)

	go h264.Mux()
	go rtp.Demux()

	select {
	case <-ctx.Done():
	}
}

func (s *WebrtcStatefulStream) PipeVP8RemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	defer log.Println("[PipeVP8RemoteTrack] canceled")

	vp8 := media.NewMuxerBuilder(vp8.NewRtpToWebmVP8Writer(),
		media.NewTargetMediaWriter(s.videoPipeWriter),
	)

	rtp := media.NewDemuxerBuilder(rtp.NewRtpTrackDemuxerReader(track),
		rtp.NewRtpTrackWriter(s.Video),
		media.NewTargetMediaWriter(vp8),
	)

	go vp8.Mux()
	go rtp.Demux()

	select {
	case <-ctx.Done():
	}
}

func (s *WebrtcStatefulStream) PipeOpusRemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	defer log.Println("[PipeOpusRemoteTrack] canceled")

	opus := media.NewMuxerBuilder(opus.NewRtpToWebmOpusWriter(),
		media.NewTargetMediaWriter(s.audioPipeWriter),
	)

	rtp := media.NewDemuxerBuilder(rtp.NewRtpTrackDemuxerReader(track),
		rtp.NewRtpTrackWriter(s.Audio),
		media.NewTargetMediaWriter(opus),
	)

	go opus.Mux()
	go rtp.Demux()

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

func (s *WebrtcStatefulStream) SetVideoTrack(track *webrtc.TrackLocalStaticRTP) {
	s.Video = track
}

type WebrtcAllocatorFunc func() (*WebrtcStatefulStream, error)

type WebrtcAllocatorFuncParams struct {
	fx.In

	HLSMediaProcessor mediaprocessor.MediaProcessor `name:"mediaprocessor.hls.default"`
}

func NewWebrtcAllocatorFunc(params WebrtcAllocatorFuncParams) WebrtcAllocatorFunc {
	return func() (*WebrtcStatefulStream, error) {
		audioPipeReader, audioPipeWriter := io.Pipe()
		videoPipeReader, videoPipeWriter := io.Pipe()

		return &WebrtcStatefulStream{
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
