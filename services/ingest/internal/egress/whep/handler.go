package whep

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/romashorodok/stream-platform/pkg/httputils"
)

type handler struct {
}

var _ httputils.HttpHandler = (*handler)(nil)

func (*handler) Whep(w http.ResponseWriter, r *http.Request) {}

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

func NewWhepHandler() *handler {
	return &handler{}
}
