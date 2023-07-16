package whip

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
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
				mediaprocessor.HSLMediaProcessor{},
			},
		},
	}
}

func audioWriter(remoteTrack *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, pipeWriter *io.PipeWriter) {
	var writerMutex sync.RWMutex

	audioBuilder := samplebuilder.New(10, &codecs.OpusPacket{}, 48000)

	// NOTE: It's packing opus into webm(EBML) container, but how to pass it like plain bytes.
	ws, _ := webm.NewSimpleBlockWriter(pipeWriter, []webm.TrackEntry{
		{
			Name:            "Audio",
			TrackNumber:     1,
			TrackUID:        12345,
			CodecID:         "A_OPUS",
			TrackType:       2,
			DefaultDuration: 20000000,
			Audio: &webm.Audio{
				SamplingFrequency: 48000.0,
				Channels:          2,
			},
		},
	})

	audioWEBMWriter := ws[0]
	var audioTimestamp time.Duration

	for {
		rtp, _, _ := remoteTrack.ReadRTP()

		audioBuilder.Push(rtp)

		sample := audioBuilder.Pop()

		if sample == nil {
			continue
		}

		writerMutex.RLock()

		audioTimestamp += sample.Duration

		_, err := audioWEBMWriter.Write(true, int64(audioTimestamp/time.Millisecond), sample.Data)

		if err != nil {
			log.Println("Unable write the audio into pipe. Err:", err)
		}

		writerMutex.RUnlock()
	}
}

func videoWriter(remoteTrack *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, pipeWriter *io.PipeWriter) {
	var writerMutex sync.RWMutex

	writer := h264writer.NewWith(pipeWriter)

	for {
		rtp, _, _ := remoteTrack.ReadRTP()

		writerMutex.RLock()
		writer.WriteRTP(rtp)
		writerMutex.RUnlock()
	}
}

func (h *handler) Handler(res http.ResponseWriter, r *http.Request) {

	err := h.orchestrator.RegisterControl(h.control)

	if err != nil {
		log.Println(err)
	}

	peerConnection, err := h.webrtcAPI.NewPeerConnection(webrtc.Configuration{})

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
