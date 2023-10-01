package rtp

import (
	"io"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
)

type PionWebrtcLocalTrack = webrtc.TrackLocalStaticRTP

type RtpTrackMediaWriter struct {
	track *PionWebrtcLocalTrack
}

func (w *RtpTrackMediaWriter) Write(packet []byte) (n int, err error) {
	return w.track.Write(packet)
}

var _ media.MediaWriter = (*RtpTrackMediaWriter)(nil)

func NewRtpTrackWriter(track *PionWebrtcLocalTrack) *RtpTrackMediaWriter {
	return &RtpTrackMediaWriter{track}
}

// Take rtp packet and write only track sample. It's discard all rtp metadata
type RtpToSampleMediaWriter struct {
	target io.Writer
	rtp    rtp.Packet
	mx     sync.Mutex
}

func (w *RtpToSampleMediaWriter) Write(p []byte) (n int, err error) {
	w.mx.Lock()
	defer w.mx.Unlock()
	if err := w.rtp.Unmarshal(p); err != nil {
		return 0, err
	}
	return w.target.Write(w.rtp.Payload)
}

var _ media.MediaWriter = (*RtpToSampleMediaWriter)(nil)

func NewRtpToSampleMediaWriter(target io.Writer) *RtpToSampleMediaWriter {
	return &RtpToSampleMediaWriter{target: target}
}
