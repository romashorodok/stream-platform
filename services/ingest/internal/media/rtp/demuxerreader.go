package rtp

import (
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/media"
)

type PionWebrtcRemoteTrack = webrtc.TrackRemote

// Read webrtc rtp track and return rtp packet as []byte.
// This packet may be restored with all metadata if needed
type RtpTrackDemuxerReader struct {
	track *PionWebrtcRemoteTrack
	mx    sync.Mutex
}

func (r *RtpTrackDemuxerReader) Read() ([]byte, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	pkt, _, err := r.track.ReadRTP()
	if err != nil {
		return nil, err
	}
	return pkt.Marshal()
}

var _ media.DemuxerReader = (*RtpTrackDemuxerReader)(nil)

func NewRtpTrackDemuxerReader(track *PionWebrtcRemoteTrack) *RtpTrackDemuxerReader {
	return &RtpTrackDemuxerReader{track: track}
}
