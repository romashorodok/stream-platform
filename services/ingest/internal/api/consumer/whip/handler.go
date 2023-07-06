package whip

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
)

// Each broadcaster sends sdp offer with `m=` field which describe stream as example:
// m=audio 54208 UDP/TLS/RTP/SAVPF 111
// a=rtpmap:111 OPUS/48000/2
//
// 54208 - is suggested port number. The port for assigning determine the ICE (Interactive Connectivity Establishment) server for the peers.
// 111 - WebRTC payload type. Where do i find this number ?
// 48,000 Hz for codec
// 2 audio channels

// m=video 54208 UDP/TLS/RTP/SAVPF 96
// a=rtpmap:96 H264/90000

type whipHandler interface {
	Handler(res http.ResponseWriter, r *http.Request)
}

type handler struct {
	whipHandler

	streamMutex *sync.Mutex
}

func NewHandler() *handler {
	return &handler{}
}

func (h *handler) Handler(res http.ResponseWriter, r *http.Request) {
	// h.streamMutex.Lock()
	// defer h.streamMutex.Unlock()

	streamKey := r.Header.Get("Authorization")

	if streamKey == "" {
		log.Println("Authorization header not set")
		return
	}

	offer, err := io.ReadAll(r.Body)

	if err != nil {
		log.Println("SDP offer is empty")
		return
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})

	peerConnection.OnTrack(
		func(track *webrtc.TrackRemote, rtp *webrtc.RTPReceiver) {
			log.Println(rtp.GetParameters().Codecs)
		},
	)

	if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  string(offer),
		Type: webrtc.SDPTypeOffer,
	}); err != nil {
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	answer, err := peerConnection.CreateAnswer(nil)

	if err != nil {
		log.Println(err)
		return
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		log.Println(err)
		return
	}

	<-gatherComplete

	res.WriteHeader(http.StatusCreated)
	fmt.Fprint(res, peerConnection.LocalDescription().SDP)
}
