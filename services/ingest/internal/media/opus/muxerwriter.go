package opus

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
)

var (
	EmptySampleError = errors.New("Empty Opus sample")
)

type RtpToWebmOpusMuxWriter struct {
	reader      io.Reader
	opusBuilder *samplebuilder.SampleBuilder
	webmBuilder webm.BlockWriteCloser
	timestamp   time.Duration

	mx sync.Mutex
}

var _ media.MuxerWriter = (*RtpToWebmOpusMuxWriter)(nil)

func (w *RtpToWebmOpusMuxWriter) Write(p []byte) (n int, err error) {
	w.mx.Lock()
	defer w.mx.Unlock()

	var rtp rtp.Packet

	if err := rtp.Unmarshal(p); err != nil {
		return 0, err
	}

	w.opusBuilder.Push(&rtp)

	sample := w.opusBuilder.Pop()
	if sample == nil {
		return 0, EmptySampleError
	}
	w.timestamp += sample.Duration

	return w.webmBuilder.Write(true, int64(w.timestamp/time.Millisecond), sample.Data)
}

func (w *RtpToWebmOpusMuxWriter) GetReader() io.Reader {
	return w.reader
}

func NewRtpToWebmOpusWriter() *RtpToWebmOpusMuxWriter {
	rtpToWebmOpusWriter := &RtpToWebmOpusMuxWriter{}
	rtpToWebmOpusWriter.opusBuilder = samplebuilder.New(10, &codecs.OpusPacket{}, 48_000)

	r, w := io.Pipe()
	buff := media.NewBufioWriterCloser(w)

	ws, _ := webm.NewSimpleBlockWriter(buff, []webm.TrackEntry{
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
	rtpToWebmOpusWriter.webmBuilder = ws[0]
	rtpToWebmOpusWriter.reader = r
	return rtpToWebmOpusWriter
}
