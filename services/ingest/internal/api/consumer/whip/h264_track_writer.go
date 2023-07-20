package whip

import (
	"io"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
)

func h264TrackWriter(remoteTrack *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, pipeWriter *io.PipeWriter) {
	var writerMutex sync.RWMutex

	writer := h264writer.NewWith(pipeWriter)

	for {
		rtp, _, _ := remoteTrack.ReadRTP()

		writerMutex.RLock()
		writer.WriteRTP(rtp)
		writerMutex.RUnlock()
	}
}
