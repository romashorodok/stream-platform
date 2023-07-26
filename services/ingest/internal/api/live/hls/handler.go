package hls

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/romashorodok/stream-platform/pkg/request"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
)

type HandlerImpl struct {
	mediaProcessor *mediaprocessor.HLSMediaProcessor
}

func NewHSLHandler(processor *mediaprocessor.HLSMediaProcessor) HandlerImpl {
	return HandlerImpl{mediaProcessor: processor}
}

func (h *HandlerImpl) Manifest(res http.ResponseWriter, r *http.Request) {
	log.Println("Manifest router active")
	log.Println(h.mediaProcessor)
}

type SegmentRequest struct {
	Segment string `json:"segment"`
}


func (*HandlerImpl) Segment(res http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	request, _ := request.UnmarshalRequest[SegmentRequest](vars)

	log.Println("Segment router active")
	log.Println(request)
}
