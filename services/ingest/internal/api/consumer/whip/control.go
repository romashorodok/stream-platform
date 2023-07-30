package whip

import (
	"io"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/orchestrator"
)

type WhipControl struct {
	webrtcAPI       *webrtc.API
	peerConnection  *webrtc.PeerConnection
	mediaProcessors []orchestrator.MediaProcessor
}

func (ctrl *WhipControl) StartStream(stream *orchestrator.Stream) error {

	ctrl.peerConnection.OnTrack(ctrl.onTrackHandler(stream))

	return nil
}

func (c *WhipControl) GetMediaProcessors() []orchestrator.MediaProcessor {
	return c.mediaProcessors
}

type OnTrackClosure func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)

func (ctrl *WhipControl) onTrackHandler(stream *orchestrator.Stream) OnTrackClosure {
	return func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Println("Establishing connection ", track.Codec())

		switch track.Codec().RTPCodecCapability.MimeType {

		case webrtc.MimeTypeOpus, "audio/OPUS":
			opusTrackWriter(track, ctrl.peerConnection, stream.Audio.PipeWriter)

		case webrtc.MimeTypeH264:
			h264TrackWriter(track, ctrl.peerConnection, stream.Video.PipeWriter)

		}
	}
}

// TODO: How does it work? As i found there is complicated establishing connection logic between peers. But lib has good abstraction logic
func (ctrl *WhipControl) handleOffer(r *http.Request) (whipSDPAnswer string, err error) {
	offer, err := io.ReadAll(r.Body)

	if err != nil {
		log.Println("Failed parse request offer or SDP offer is invalid. Err:", err)
		return "", err
	}

	if err := ctrl.peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  string(offer),
		Type: webrtc.SDPTypeOffer,
	}); err != nil {
		log.Println("Cannot set remote description in local state. Err:", err)
		return "", err
	}

	gatheringSuccess := webrtc.GatheringCompletePromise(ctrl.peerConnection)

	session, err := ctrl.peerConnection.CreateAnswer(nil)

	if err != nil {
		log.Println("Cannot start connection to send the answer to peer. Err:", err)
		return "", err
	}

	if err = ctrl.peerConnection.SetLocalDescription(session); err != nil {
		log.Println("Cannot set local description. Err:", err)
		return "", err
	}

	<-gatheringSuccess

	return ctrl.peerConnection.LocalDescription().SDP, err
}
