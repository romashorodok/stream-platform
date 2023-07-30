package orchestrator

import (
	"errors"
	"io"
	"log"
	"sync"

	"github.com/gorilla/mux"
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
	StartStream(stream *Stream) error
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
}

func NewOrchestrator(router *mux.Router, shutdown *shutdown.Shutdown) *Orchestrator {
	videoReader, videoWriter := io.Pipe()
	audioReader, audioWriter := io.Pipe()

	o := &Orchestrator{
		stream: &Stream{
			Video: &VideoStream{PipeReader: videoReader, PipeWriter: videoWriter},
			Audio: &AudioStream{PipeReader: audioReader, PipeWriter: audioWriter},
		},
		control: nil,
	}
	o.shutdown = shutdown
	o.hlsRouter = hlsrouter.NewHLSRouter(router)

	return o
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
	if o.control == nil {
		return errors.New("not found control name.")
	}

	for _, processor := range o.control.GetMediaProcessors() {
		o.shutdown.AddTask(processor.Destroy)
	}

	if err := o.control.StartStream(o.stream); err != nil {
		log.Println("Start stream error", err)
	}

	go o.StartMediaProcessors()

	return nil
}

func (o *Orchestrator) Stop() {
	// NOTE: Should it fail fast ? The main get the signal and stop the server
	o.shutdown.Trigger()
}
