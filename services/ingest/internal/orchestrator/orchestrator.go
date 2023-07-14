package orchestrator

import (
	"errors"
	"io"
	"log"
	"sync"
)

type MediaProcessor interface {
	Transcode(mediaSourcePipe *io.PipeReader) error
}

type Control interface {
	StartStream(stream *Stream) error
	GetMediaProcessors() []MediaProcessor
}

type Stream struct {
	PipeReader *io.PipeReader
	PipeWriter *io.PipeWriter
}

type Orchestrator struct {
	Name      string
	IsRunning bool

	Stream            *Stream
	controls          map[string]Control
	orchestratorMutex sync.Mutex
}

func NewOrchestrator() *Orchestrator {
	o := &Orchestrator{}
	o.controls = make(map[string]Control)

	pipeReader, pipeWriter := io.Pipe()
	o.Stream = &Stream{
		PipeReader: pipeReader,
		PipeWriter: pipeWriter,
	}

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
	var wg sync.WaitGroup

	control := o.controls[o.Name]

	for _, processor := range control.GetMediaProcessors() {
		wg.Add(1)
		go func(processor MediaProcessor) {
			defer func() {
				wg.Done()
			}()

			err := processor.Transcode(o.Stream.PipeReader)

			if err != nil {
				log.Println("Error was caught in media processor. Err", err)
			}
		}(processor)
	}

	wg.Wait()
}

func (o *Orchestrator) Start() error {
	if len(o.controls) > 1 {
		return errors.New("system supports only one stream source")
	}

	control := o.controls[o.Name]

	if control == nil {
		return errors.New("not found control name.")
	}

	if err := control.StartStream(o.Stream); err != nil {
		log.Println("Start stream error", err)
	}

	go o.StartMediaProcessors()

	o.IsRunning = true

	return nil
}
