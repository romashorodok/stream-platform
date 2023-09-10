package hls

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/romashorodok/stream-platform/pkg/request"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
)

func MediaResponseStream(res http.ResponseWriter, buffer []byte, file string) error {
	media, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("media file not exists. Error: %s", err)
	}
	defer media.Close()

	for {
		nRead, err := media.Read(buffer)

		if err != nil {
			if err == io.EOF {
				break
			}

			return errors.New("read media error")
		}

		if nRead == 0 {
			break
		}

		_, err = res.Write(buffer[:nRead])
		if err != nil {
			break
		}

		if flusher, ok := res.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	return nil
}

type HandlerImpl struct {
	mediaProcessor *mediaprocessor.HLSMediaProcessor
}

func NewHSLHandler(processor *mediaprocessor.HLSMediaProcessor) HandlerImpl {
	return HandlerImpl{mediaProcessor: processor}
}

func (h *HandlerImpl) Manifest(res http.ResponseWriter, r *http.Request) {
	res.Header().Set("Cache-Control", "no-cache, no-store, private")

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "*")
	res.Header().Set("Access-Control-Allow-Methods", "GET")

	res.Header().Set("Content-Type", "application/vnd.apple.mpegurl")

	if h.mediaProcessor.ManifestFile == "" {
		log.Println("[HLS Manifest Handler] Manifest file does not exist")
		res.WriteHeader(http.StatusNotFound)
		return
	}

	if err := MediaResponseStream(res, make([]byte, 2048), h.mediaProcessor.ManifestFile); err != nil {
		log.Printf("[HLS Manifest Handler] %s\n", err)
		res.WriteHeader(http.StatusNotFound)
		return
	}

}

type SegmentRequest struct {
	Segment string `json:"segment"`
}

func (h *HandlerImpl) Segment(res http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	request, _ := request.UnmarshalRequest[SegmentRequest](vars)

	if request.Segment == "" {
		log.Println("[HLS Segment Handler] Segment file does not exist")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	res.Header().Set("Content-Type", "application/octet-stream")
	res.Header().Set("Transfer-Encoding", "chunked")

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "*")
	res.Header().Set("Access-Control-Allow-Methods", "GET")

	segment := fmt.Sprintf("%s/%s", h.mediaProcessor.SourceDirectory, request.Segment)

	if err := MediaResponseStream(res, make([]byte, 2048), segment); err != nil {
		log.Printf("[HLS Segment Handler] %s\n", err)
		res.WriteHeader(http.StatusNotFound)
		return
	}
}
