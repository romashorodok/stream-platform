package h264muxer

import (
	"io"

	"github.com/pion/rtp/codecs"
)

type h264Muxer struct {
	h264codec codecs.H264Packet
	reader    *io.PipeReader
	writer    *io.PipeWriter
}

func (h *h264Muxer) Read(p []byte) (n int, err error) {
	return h.reader.Read(p)
}

func (m *h264Muxer) Write(p []byte) (n int, err error) {
	h264, err := m.h264codec.Unmarshal(p)
	if err != nil {
		return 0, err
	}
	return m.writer.Write(h264)
}

func NewH264Muxer() *h264Muxer {
	muxer := h264Muxer{}
	muxer.reader, muxer.writer = io.Pipe()
	return &muxer
}
