package whip

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream"
	"github.com/romashorodok/stream-platform/services/ingest/internal/wrtc"
	"github.com/romashorodok/stream-platform/services/ingest/pkg/service"
	"go.uber.org/fx"
)

type handler struct {
	webrtcAPI            *webrtc.API
	ingestSystemConfig   *service.IngestSystemConfig
	statefulStreamGlobal *statefulstream.StatefulStreamGlobal
}

var _ httputils.HttpHandler = (*handler)(nil)

func (h *handler) Handler(w http.ResponseWriter, r *http.Request) {
	connConfig := webrtc.Configuration{}

	peerConnection, err := h.webrtcAPI.NewPeerConnection(connConfig)
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable create peer connection. Err:", err.Error())
		return
	}

	if _, err := peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable set resiving only mode for video. Err:", err.Error())
		return
	}

	if _, err := peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable set resiving only mode for audio. Err:", err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.TODO())

	peerConnection.OnICEConnectionStateChange(
		func(state webrtc.ICEConnectionState) {
			if state == webrtc.ICEConnectionStateDisconnected {
				peerConnection.Close()
				cancel()
			}
		},
	)

	offer, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable read offer. Err:", err.Error())
		return
	}

	answer, err := wrtc.Answer(peerConnection, string(offer))
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable handle and generate answer. Err:", err.Error())
		return
	}

	wrtcHandler, err := h.statefulStreamGlobal.HandleWebrtc(ctx)
	if err != nil {
		switch err {
		case statefulstream.NewWebrtcStatefulStreamError:
			httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable handle webrtc. Err:", err.Error())
		default:
			httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable handle webrtc. Err:", err.Error())

		}
		return
	}

	peerConnection.OnTrack(wrtcHandler)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, answer)
}

const whipHandler = "/api/ingress/whip"

func (h *handler) GetOption() httputils.HttpHandlerOption {
	return func(hand http.Handler) {
		switch hand.(type) {
		case *mux.Router:
			mux := hand.(*mux.Router)
			mux.HandleFunc(whipHandler, h.Handler)
		default:
			panic("unsupported whip handler")
		}
	}
}

type WhipHandlerParams struct {
	fx.In

	WebrtcAPI            *webrtc.API
	IngestSystemConfig   *service.IngestSystemConfig
	StatefulStreamGlobal *statefulstream.StatefulStreamGlobal
}

func NewWhipHandler(params WhipHandlerParams) *handler {
	return &handler{
		webrtcAPI:            params.WebrtcAPI,
		ingestSystemConfig:   params.IngestSystemConfig,
		statefulStreamGlobal: params.StatefulStreamGlobal,
	}
}
