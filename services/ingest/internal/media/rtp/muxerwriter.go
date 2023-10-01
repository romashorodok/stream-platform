package rtp

import (
	"errors"
	"io"
	"sync"

	"github.com/pion/rtp"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
)

var (
	NotRTPPackerError = errors.New("Invalid written packet payload unable serialize that.")
)

type RtpToRtpMuxerWriter struct {
	reader io.Reader
	writer io.Writer

	mx sync.Mutex
}

// GetReader implements media.MuxerWriter.
func (w *RtpToRtpMuxerWriter) GetReader() io.Reader {
	return w.reader
}

// Write implements media.MuxerWriter.
func (w *RtpToRtpMuxerWriter) Write(p []byte) (n int, err error) {
	w.mx.Lock()
	defer w.mx.Unlock()

	var rtp rtp.Packet

	if err := rtp.Unmarshal(p); err != nil {
		return 0, err
	}

	b, err := rtp.Marshal()
	if err != nil {
		return 0, NotRTPPackerError
	}
	return w.writer.Write(b)
}

var _ media.MuxerWriter = (*RtpToRtpMuxerWriter)(nil)

func NewRtpToRtpMuxerWriter() *RtpToRtpMuxerWriter {
	r, w := io.Pipe()
	return &RtpToRtpMuxerWriter{reader: r, writer: w}
}
