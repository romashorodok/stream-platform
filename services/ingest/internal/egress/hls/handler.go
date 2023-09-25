package hls

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/request"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor/hls"
	"go.uber.org/fx"
)

func MediaResourceResponseStream(res http.ResponseWriter, file string) error {
	media, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("media file not exists. Error: %s", err)
	}
	defer media.Close()

	if _, err := io.Copy(res, media); err != nil {
		return err
	}

	if flusher, ok := res.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

func Cors(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, private")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
}

type handler struct {
	mediaProcessor mediaprocessor.MediaProcessor
}

var _ httputils.HttpHandler = (*handler)(nil)

func (h *handler) GetHlsMediaProcessor() (*hls.FFmpegHLSMediaProcessor, error) {
	return mediaprocessor.CastMediaProcessor[hls.FFmpegHLSMediaProcessor](h.mediaProcessor)
}

func (h *handler) Manifest(w http.ResponseWriter, r *http.Request) {
	Cors(w)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

	processor, err := h.GetHlsMediaProcessor()
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError)
		return
	}

	if processor.ManifestFile == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := MediaResourceResponseStream(w, processor.ManifestFile); err != nil {
		log.Printf("[HLS Manifest Handler] %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type SegmentRequest struct {
	Segment string `json:"segment"`
}

func (h *handler) Segment(w http.ResponseWriter, r *http.Request) {
	Cors(w)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")

	processor, err := h.GetHlsMediaProcessor()
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	request, _ := request.UnmarshalRequest[SegmentRequest](vars)

	if request.Segment == "" {
		log.Println("[HLS Segment Handler] Segment file does not exist")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	segment := fmt.Sprintf("%s/%s", processor.SourceDirectory, request.Segment)

	if err := MediaResourceResponseStream(w, segment); err != nil {
		log.Printf("[HLS Segment Handler] %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

const hlsManifestHandler = "/api/egress/hls"
const hlsSegmentHandler = "/api/egress/hls/{segment}"

func (h *handler) GetOption() httputils.HttpHandlerOption {
	return func(hand http.Handler) {
		switch hand.(type) {
		case *mux.Router:
			mux := hand.(*mux.Router)
			mux.HandleFunc(hlsManifestHandler, h.Manifest)
			mux.HandleFunc(hlsSegmentHandler, h.Segment)
		default:
			panic("unsupported hls handler")
		}
	}
}

type HLSHandlerParams struct {
	fx.In

	MediaProcessor mediaprocessor.MediaProcessor `name:"mediaprocessor.hls.default"`
}

func NewHLSHandler(params HLSHandlerParams) *handler {
	return &handler{
		mediaProcessor: params.MediaProcessor,
	}
}
