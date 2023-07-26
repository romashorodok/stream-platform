package whip

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
	"github.com/romashorodok/stream-platform/services/ingest/internal/orchestrator"
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

	orchestrator *orchestrator.Orchestrator
	webrtcAPI    *webrtc.API
	control      *WhipControl
	streamMutex  sync.Mutex
}

func NewHandler(o *orchestrator.Orchestrator, webrtcAPI *webrtc.API) *handler {
	o.Name = "whip"

	return &handler{
		orchestrator: o,
		webrtcAPI:    webrtcAPI,
		control: &WhipControl{
			webrtcAPI: webrtcAPI,
			mediaProcessors: []orchestrator.MediaProcessor{
				&mediaprocessor.HLSMediaProcessor{},
			},
		},
	}
}

func (h *handler) Handler(res http.ResponseWriter, r *http.Request) {

	err := h.orchestrator.RegisterControl(h.control)

	if err != nil {
		log.Println(err)
	}

	peerConnection, err := h.webrtcAPI.NewPeerConnection(webrtc.Configuration{})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		if state == webrtc.ICEConnectionStateDisconnected {
			h.orchestrator.Stop()
		}
	})

	if err != nil {
		log.Println("Cannot create peer connection. Err:", err)
	}

	h.control.peerConnection = peerConnection

	whipSDPAnswer, err := h.control.handleOffer(r)

	if err != nil {
		log.Println("Cannot send answer to peer. Err:", err)
		return
	}

	go h.orchestrator.Start()

	res.WriteHeader(http.StatusCreated)
	fmt.Fprint(res, whipSDPAnswer)

}
