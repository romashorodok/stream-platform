package whep

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream/webrtcstatefulstream"
	"github.com/romashorodok/stream-platform/services/ingest/internal/wrtc"
	"go.uber.org/fx"
)

func Cors(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, private")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
}

type handler struct {
	webrtcAPI            *webrtc.API
	statefulStreamGlobal *statefulstream.StatefulStreamGlobal
}

var _ httputils.HttpHandler = (*handler)(nil)

func ensureWbertcStatefulStream(s statefulstream.StatefulStream) (*webrtcstatefulstream.WebrtcStatefulStream, error) {
	if stream, ok := s.(*webrtcstatefulstream.WebrtcStatefulStream); ok {
		return stream, nil
	}
	return nil, errors.New("Support only webrtc stream type")
}

func (h *handler) Whep(w http.ResponseWriter, r *http.Request) {
	Cors(w)

	if r.Method == "OPTIONS" {
		return
	}

	connConfig := webrtc.Configuration{}

	peerConnection, err := h.webrtcAPI.NewPeerConnection(connConfig)
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable create peer connection. Err:", err.Error())
		return
	}

	stream, err := ensureWbertcStatefulStream(h.statefulStreamGlobal.GetStatefulStream())
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusConflict, "invalid stream type start . The stream with webrtc", err.Error())
		return
	}

	offer, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable read offer. Err:", err.Error())
		return
	}

	_, _ = peerConnection.AddTrack(stream.Audio)
	_, _ = peerConnection.AddTrack(stream.Video)

	answer, err := wrtc.Answer(peerConnection, string(offer))
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "unable handle and generate answer. Err:", err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, answer)
}

const whepHandler = "/api/egress/whep"

func (h *handler) GetOption() httputils.HttpHandlerOption {
	return func(hand http.Handler) {
		switch hand.(type) {
		case *mux.Router:
			mux := hand.(*mux.Router)
			mux.HandleFunc(whepHandler, h.Whep)
		default:
			panic("unsupported hls handler")
		}
	}
}

type WhepHandlerParams struct {
	fx.In

	WebrtcAPI            *webrtc.API
	StatefulStreamGlobal *statefulstream.StatefulStreamGlobal
}

func NewWhepHandler(params WhepHandlerParams) *handler {
	return &handler{
		webrtcAPI:            params.WebrtcAPI,
		statefulStreamGlobal: params.StatefulStreamGlobal,
	}
}
