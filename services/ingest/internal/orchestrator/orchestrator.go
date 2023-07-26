package orchestrator

import (
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/mux"
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
	Name      string
	IsRunning bool

	router    *mux.Router
	hlsRouter *hlsrouter.HLSRouter

	stream            *Stream
	controls          map[string]Control
	orchestratorMutex sync.Mutex
}

func NewOrchestrator(router *mux.Router) *Orchestrator {
	videoReader, videoWriter := io.Pipe()
	audioReader, audioWriter := io.Pipe()

	o := &Orchestrator{
		stream: &Stream{
			Video: &VideoStream{PipeReader: videoReader, PipeWriter: videoWriter},
			Audio: &AudioStream{PipeReader: audioReader, PipeWriter: audioWriter},
		},
		controls: make(map[string]Control),
	}
	o.controls = make(map[string]Control)

	o.hlsRouter = hlsrouter.NewHLSRouter(router)

	return o
}

func (o *Orchestrator) RegisterControl(impl Control) error {
	o.orchestratorMutex.Lock()
	defer o.orchestratorMutex.Unlock()

	if o.controls[o.Name] != nil {
		return errors.New("control already assigned")
	}

	if len(o.controls) > 1 {
		return errors.New("system supports only one stream source")
	}

	o.controls[o.Name] = impl

	return nil
}

func (o *Orchestrator) StartMediaProcessors() {
	shutdown := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	control := o.controls[o.Name]

	go func() {
		var wg sync.WaitGroup
		for _, processor := range control.GetMediaProcessors() {
			wg.Add(1)

			go func(processor MediaProcessor) {
				defer func() {
					wg.Done()
				}()

				switch concreteProcessor := processor.(type) {
				case *mediaprocessor.HLSMediaProcessor:
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
		done <- struct{}{}
		log.Println("Done")
	}()

	select {
	case <-shutdown:
		o.Stop()
	case <-done:
		o.Stop()
	}
}

func (o *Orchestrator) Start() error {
	if len(o.controls) > 1 {
		return errors.New("system supports only one stream source")
	}

	control := o.controls[o.Name]

	if control == nil {
		return errors.New("not found control name.")
	}

	if err := control.StartStream(o.stream); err != nil {
		log.Println("Start stream error", err)
	}

	go o.StartMediaProcessors()

	o.IsRunning = true

	return nil
}

func (o *Orchestrator) Stop() error {
	control := o.controls[o.Name]

	log.Println("Destroying media processors")

	for _, processor := range control.GetMediaProcessors() {
		processor.Destroy()
	}

	o.hlsRouter.RemoveRoutes()

	return nil
}
