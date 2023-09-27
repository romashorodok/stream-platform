package opusmuxer

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

var (
	EmptySampleError  = errors.New("Empty Opus sample")
	NotSupportedError = errors.New("Not supported action")
)

type opusMuxer struct {
	mx     sync.Mutex
	reader *io.PipeReader
	writer *io.PipeWriter

	rtp         *rtp.Packet
	timestamp   time.Duration
	opusSample  *samplebuilder.SampleBuilder
	webmBuilder webm.BlockWriteCloser
}

func (m *opusMuxer) Bind(track *webrtc.TrackRemote) {
	var mx sync.Mutex

	for {
		mx.Lock()
		n, _, err := track.ReadRTP()

		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		m.rtp = n

		_, _ = m.Write([]byte{})
		mx.Unlock()
	}
}

func (m *opusMuxer) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *opusMuxer) Write(p []byte) (n int, err error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	if m.rtp != nil {
		m.opusSample.Push(m.rtp)
	} else if len(p) > 0 {
		return 0, NotSupportedError
	}

	sample := m.opusSample.Pop()

	if sample == nil {
		return 0, EmptySampleError
	}

	m.timestamp += sample.Duration

	return m.webmBuilder.Write(true, int64(m.timestamp/time.Millisecond), sample.Data)
}

func NewOpusMuxer() *opusMuxer {
	muxer := opusMuxer{}
	r, w := io.Pipe()
	muxer.opusSample = samplebuilder.New(10, &codecs.OpusPacket{}, 48_000)
	muxer.reader = r
	muxer.writer = w

	bufW := newCustomWriterCloser(w)

	ws, _ := webm.NewSimpleBlockWriter(bufW, []webm.TrackEntry{
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
	muxer.webmBuilder = ws[0]

	return &muxer
}

type customWriterCloser struct {
	writer *bufio.Writer
	closer io.WriteCloser
}

func newCustomWriterCloser(w io.WriteCloser) *customWriterCloser {
	return &customWriterCloser{
		writer: bufio.NewWriter(w),
		closer: w,
	}
}

func (c *customWriterCloser) Write(p []byte) (n int, err error) {
	return c.writer.Write(p)
}

func (c *customWriterCloser) Close() error {
	err := c.writer.Flush()
	if err != nil {
		return err
	}
	return c.closer.Close()
}
