package webrtcstatefulstream

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
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

	var rwmx sync.RWMutex

	writer := h264writer.NewWith(s.videoPipeWriter)

	// go wrtc.PipeRemoteTrack(ctx, track, s.video)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			rtp, _, _ := track.ReadRTP()

			if rtp == nil {
				continue
			}

			rwmx.RLock()
			if err := writer.WriteRTP(rtp); err != nil {
				// log.Printf("[PipeH264RemoteTrack]: h264 writer. %s", err)
			}
			rwmx.RUnlock()
		}
	}
}

func (s *WebrtcStatefulStream) PipeOpusRemoteTrack(ctx context.Context, track *webrtc.TrackRemote) {
	defer log.Println("[PipeOpusRemoteTrack] canceled")

	var rwmx sync.RWMutex

	// TODO: Start SFU

	// go wrtc.PipeRemoteTrack(ctx, track, s.audio)

	opusBuilder := samplebuilder.New(10, &codecs.OpusPacket{}, 48_000)

	// NOTE: Don't mux twice
	ws, _ := webm.NewSimpleBlockWriter(s.audioPipeWriter, []webm.TrackEntry{
		{
			Name:            "Audio",
			TrackNumber:     1,
			TrackUID:        12345,
			CodecID:         "A_OPUS",
			TrackType:       2,
			DefaultDuration: 20000000,
			Audio: &webm.Audio{
				SamplingFrequency: 48_000.0,
				Channels:          2,
			},
		},
	})
	webmOpusBlockWriter := ws[0]

	var timestamp time.Duration

	for {
		select {
		case <-ctx.Done():
			return
		default:
			rtp, _, _ := track.ReadRTP()

			if rtp == nil {
				continue
			}

			opusBuilder.Push(rtp)

			opusSample := opusBuilder.Pop()
			if opusSample == nil {
				continue
			}

			rwmx.RLock()
			timestamp += opusSample.Duration

			if _, err := webmOpusBlockWriter.Write(true, int64(timestamp/time.Millisecond), opusSample.Data); err != nil {
				// log.Printf("[PipeOpusRemoteTrack]: Webm opus block writer. %s", err)
			}
			rwmx.RUnlock()
		}
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
