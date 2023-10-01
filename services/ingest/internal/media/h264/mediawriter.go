package h264

import (
	"errors"
	"io"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
)

var (
	InvalidRTPData = errors.New("Invalid rtp packet")
)

type RtpToH264MediaWriter struct {
	writer *h264writer.H264Writer
}

func (w *RtpToH264MediaWriter) Write(p []byte) (n int, err error) {

	var rtp rtp.Packet

	if err := rtp.Unmarshal(p); err != nil {
		return 0, InvalidRTPData
	}

	if err := w.writer.WriteRTP(&rtp); err != nil {
		return 0, err
	}

	return len(p), nil
}

var _ media.MediaWriter = (*RtpToH264MediaWriter)(nil)

func NewRtpToH264MediaWriter(target io.Writer) *RtpToH264MediaWriter {
	writer := h264writer.NewWith(target)

	return &RtpToH264MediaWriter{writer: writer}
}
