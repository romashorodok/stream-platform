package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Task func()

type Shutdown struct {
	shutdownSignal <-chan os.Signal
	tasks          []Task
	mutex          sync.Mutex
	context        context.Context
	Trigger        context.CancelFunc
}

func NewShutdown() *Shutdown {
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
		syscall.SIGHUP,
	)
	ctx, trigger := context.WithCancel(context.Background())

	return &Shutdown{
		tasks:          make([]Task, 0),
		shutdownSignal: shutdownSignal,
		context:        ctx,
		Trigger:        trigger,
	}
}

func (s *Shutdown) AddTask(task func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.tasks = append(s.tasks, task)
}

func (s *Shutdown) Gracefully() {
	select {
	case <-s.shutdownSignal:
	case <-s.context.Done():
	}

	var wg sync.WaitGroup

	wg.Add(len(s.tasks))

	for _, task := range s.tasks {
		go func(eval func()) {
			defer wg.Done()
			eval()
		}(task)
	}

	wg.Wait()
}
