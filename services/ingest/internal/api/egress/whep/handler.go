package whep

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/services/ingest/internal/config"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream/webrtcstatefulstream"
)

type Handler struct {
	// stream    *orchestrator.WebrtcStream
	stream    *webrtcstatefulstream.WebrtcStatefulStream
	webrtcAPI *webrtc.API
	config    *config.Config
}

func NewHandler(api *webrtc.API, stream *webrtcstatefulstream.WebrtcStatefulStream, config *config.Config) *Handler {
	return &Handler{
		webrtcAPI: api,
		stream:    stream,
		config:    config,
	}
}

func (h *Handler) Whep(offer string) (string, string, error) {
	whepSessionId := uuid.New().String()

	log.Println("WHEP config", h.config)

	connconf := webrtc.Configuration{}
	// if h.config.Turn.Enable {
	// 	connconf.ICEServers = []webrtc.ICEServer{
	// 		{
	// 			URLs:       []string{h.config.Turn.URL},
	// 			Username:   h.config.Turn.User,
	// 			Credential: h.config.Turn.Password,
	// 		},
	// 	}
	// 	connconf.ICETransportPolicy = webrtc.ICETransportPolicyRelay
	// }

	peerConnection, err := h.webrtcAPI.NewPeerConnection(connconf)
	if err != nil {
		return "", "", fmt.Errorf("unable create peer connection. Err: %s", err)
	}

	// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
	// 	return "", "", fmt.Errorf("unable set peer connection to send only. Err: %s", err)
	// }

	// if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
	// 	return "", "", fmt.Errorf("unable set peer connection to send only. Err: %s", err)
	// }

	_, _ = peerConnection.AddTrack(h.stream.Audio)
	_, err = peerConnection.AddTrack(h.stream.Video)
	if err != nil {
		log.Println("rtp sender error")
	}

	// go func() {
	// 	for {
	// 		rtcpPackets, _, rtcpErr := rtpSender.ReadRTCP()
	// 		if rtcpErr != nil {
	// 			return
	// 		}

	// 		for _, r := range rtcpPackets {
	// 			if _, isPLI := r.(*rtcp.PictureLossIndication); isPLI {
	// 				select {
	// 				case h.stream.PliChan <- true:
	// 				default:
	// 				}
	// 			}
	// 		}
	// 	}
	// }()

	if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  offer,
		Type: webrtc.SDPTypeOffer,
	}); err != nil {
		return "", "", err
	}

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("PeerConnection State has changed %s \n", state.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("PeerConnection ice candidate", candidate.String)
			log.Printf("PeerConnection ice candidate", candidate.Port)
			log.Printf("PeerConnection ice candidate", candidate.Protocol)
			log.Printf("PeerConnection ice candidate", candidate.Address)
		}
	})

	gatheringSuccess := webrtc.GatheringCompletePromise(peerConnection)

	session, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", "", err
	}

	if err = peerConnection.SetLocalDescription(session); err != nil {
		return "", "", err
	}

	<-gatheringSuccess

	return peerConnection.LocalDescription().SDP, whepSessionId, nil
}

func (hand *Handler) Handler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Cache-Control", "no-cache, no-store, private")

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "*")
	res.Header().Set("Access-Control-Allow-Methods", "*")

	offer, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		httputils.WriteErrorResponse(res, http.StatusInternalServerError, "unable read offer from body. Err:", err.Error())
	}

	sdpAnswer, whepSessionId, _ := hand.Whep(string(offer))

	_ = whepSessionId
	apiPath := req.Host + strings.TrimSuffix(req.URL.RequestURI(), "whep")
	_ = apiPath
	// res.Header().Add("Content-Type", "application/sdp")
	// res.Header().Add("Link", `<`+apiPath+"sse/"+whepSessionId+`>; rel="urn:ietf:params:whep:ext:core:server-sent-events"; events="layers"`)
	// res.Header().Add("Link", `<`+apiPath+"layer/"+whepSessionId+`>; rel="urn:ietf:params:whep:ext:core:layer"`)

	fmt.Fprint(res, sdpAnswer)
}
