package router

import (
	"github.com/gorilla/mux"
	"github.com/romashorodok/stream-platform/services/ingest/internal/api/live/hls"
)

var (
	unimplemented = hls.HandlerUnimplemented{}
)

type HLSRouter struct {
	Active        bool
	manifestRoute *mux.Route
	segmentRoute  *mux.Route
}

func NewHLSRouter(router *mux.Router) *HLSRouter {
	return &HLSRouter{
		manifestRoute: router.Path("/api/live/hls"),
		segmentRoute:  router.Path("/api/live/hls/{segment}"),
	}
}

func (r *HLSRouter) RegisterRoutes(handler hls.HandlerImpl) {
	r.manifestRoute.HandlerFunc(handler.Manifest)
	r.segmentRoute.HandlerFunc(handler.Segment)
	r.Active = true
}

func (r *HLSRouter) RemoveRoutes() {
	r.manifestRoute.HandlerFunc(unimplemented.Manifest)
	r.segmentRoute.HandlerFunc(unimplemented.Segment)
	r.Active = false
}
