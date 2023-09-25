package orchestrator

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/gorilla/mux"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/shutdown"
	"github.com/romashorodok/stream-platform/services/ingest/internal/api/live/hls"
	hlsrouter "github.com/romashorodok/stream-platform/services/ingest/internal/api/live/hls/router"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
)

type MediaProcessor interface {
	Transcode(videoSourcePipe *io.PipeReader, audioSourcePipe *io.PipeReader) error
	Destroy()
}

type Control interface {
	StartStream(stream *Stream, webrtcSream *WebrtcStream) error
	GetMediaProcessors() []MediaProcessor
}

type VideoStream struct {
	PipeReader *io.PipeReader
	PipeWriter *io.PipeWriter
}

type AudioStream struct {
	PipeReader *io.PipeReader
	PipeWriter *io.PipeWriter
}

type Stream struct {
	Video *VideoStream
	Audio *AudioStream
}

type Orchestrator struct {
	Name string

	shutdown  *shutdown.Shutdown
	router    *mux.Router
	hlsRouter *hlsrouter.HLSRouter

	stream            *Stream
	control           Control
	orchestratorMutex sync.Mutex

	WebrtcStream *WebrtcStream
}

func NewOrchestrator(router *mux.Router, shutdown *shutdown.Shutdown) *Orchestrator {
	videoReader, videoWriter := io.Pipe()
	audioReader, audioWriter := io.Pipe()

	o := &Orchestrator{
		stream: &Stream{
			Video: &VideoStream{PipeReader: videoReader, PipeWriter: videoWriter},
			Audio: &AudioStream{PipeReader: audioReader, PipeWriter: audioWriter},
		},
		WebrtcStream: &WebrtcStream{PliChan: make(chan any, 50)},
		control:      nil,
	}
	o.shutdown = shutdown
	o.hlsRouter = hlsrouter.NewHLSRouter(router)

	return o
}

type WebrtcStream struct {
	Video *webrtc.TrackLocalStaticRTP
	Audio *webrtc.TrackLocalStaticRTP

	PliChan chan any
}

func (s *WebrtcStream) Start() error {
	audio, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if err != nil {
		return fmt.Errorf("unable create audio track. Err: %s", err)
	}

	video, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		return fmt.Errorf("unable create video track. Err: %s", err)
	}

	s.Audio = audio
	s.Video = video

	return nil
}

func (o *Orchestrator) RegisterControl(impl Control) error {
	o.orchestratorMutex.Lock()
	defer o.orchestratorMutex.Unlock()

	if o.control != nil {
		return errors.New("control already assigned")
	}

	o.control = impl

	return nil
}

func (o *Orchestrator) StartMediaProcessors() {
	var wg sync.WaitGroup

	for _, processor := range o.control.GetMediaProcessors() {
		wg.Add(1)

		go func(processor MediaProcessor) {
			defer func() {
				wg.Done()
			}()

			switch concreteProcessor := processor.(type) {
			case *mediaprocessor.HLSMediaProcessor:
				o.shutdown.AddTask(o.hlsRouter.RemoveRoutes)
				o.hlsRouter.RegisterRoutes(
					hls.NewHSLHandler(concreteProcessor),
				)
			}

			err := processor.Transcode(o.stream.Video.PipeReader, o.stream.Audio.PipeReader)
			if err != nil {
				log.Println("Error was caught in media processor. Err", err)
			}

		}(processor)
	}

	wg.Wait()
}

func (o *Orchestrator) Start() error {

	_ = o.WebrtcStream.Start()

	if o.control == nil {
		return errors.New("not found control name.")
	}

	if err := o.control.StartStream(o.stream, o.WebrtcStream); err != nil {
		log.Println("Start stream error", err)
	}

	for _, processor := range o.control.GetMediaProcessors() {
		o.shutdown.AddTask(processor.Destroy)
	}

	go o.StartMediaProcessors()

	return nil
}

func (o *Orchestrator) Stop() {
	// NOTE: Should it fail fast ? The main get the signal and stop the server
	o.shutdown.Trigger()
}
