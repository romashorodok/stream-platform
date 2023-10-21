package vp8

import (
	"errors"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"

	"io"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

var (
	EmptySampleError      = errors.New("Empty Opus sample")
	UnableInitWebmBuilder = errors.New("Unable init webm builder")
)

type RtpToWebmVP8MuxWriter struct {
	reader      io.Reader
	vp8builder  *samplebuilder.SampleBuilder
	webmBuilder webm.BlockWriteCloser
	timestamp   time.Duration
	buff        *media.BufioWriterCloser

	mx sync.Mutex
}

func (w *RtpToWebmVP8MuxWriter) GetReader() io.Reader {
	return w.reader
}

func (w *RtpToWebmVP8MuxWriter) Write(p []byte) (n int, err error) {
	w.mx.Lock()
	defer w.mx.Unlock()

	var rtp rtp.Packet

	if err := rtp.Unmarshal(p); err != nil {
		return 0, err
	}

	w.vp8builder.Push(&rtp)

	sample := w.vp8builder.Pop()
	if sample == nil {
		return 0, errors.Join(EmptySampleError, err)
	}

	videoKeyframe := (sample.Data[0]&0x1 == 0)
	if w.webmBuilder == nil {
		if !videoKeyframe {
			return 0, UnableInitWebmBuilder
		}

		raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24

		width := int(raw & 0x3FFF)
		height := int((raw >> 16) & 0x3FFF)

		ws, _ := webm.NewSimpleBlockWriter(w.buff, []webm.TrackEntry{
			{
				Name:            "Video",
				TrackNumber:     1,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 20000000,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		})
		w.webmBuilder = ws[0]

	}
	w.timestamp += sample.Duration

	return w.webmBuilder.Write(videoKeyframe, int64(w.timestamp/time.Millisecond), sample.Data)
}

var _ media.MuxerWriter = (*RtpToWebmVP8MuxWriter)(nil)

func NewRtpToWebmVP8Writer() *RtpToWebmVP8MuxWriter {
	rtpToWebmVP8writer := &RtpToWebmVP8MuxWriter{}
	rtpToWebmVP8writer.vp8builder = samplebuilder.New(10, &codecs.VP8Packet{}, 90000)

	r, w := io.Pipe()
	rtpToWebmVP8writer.buff = media.NewBufioWriterCloser(w)

	rtpToWebmVP8writer.reader = r
	return rtpToWebmVP8writer
}
