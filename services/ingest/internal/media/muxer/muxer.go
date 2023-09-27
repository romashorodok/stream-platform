package muxer

import (
	"io"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/muxer/h264muxer"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media/muxer/rtpmuxer"
)

type MuxPionWebrtcLocalTrack interface {
	Bind(track *webrtc.TrackLocalStaticRTP)
}

type MuxPionWebrtcRemoteTrack interface {
	BindLocal(track *webrtc.TrackRemote)
}

// Muxer take resource sample and pack it into specific container type
type Muxer interface {
	io.Reader
	io.Writer
}

var _ Muxer = rtpmuxer.NewRtpMuxer()
var _ Muxer = h264muxer.NewH264Muxer()

var NewRtpMuxer = rtpmuxer.NewRtpMuxer
var NewH264Muxer = h264muxer.NewH264Muxer
