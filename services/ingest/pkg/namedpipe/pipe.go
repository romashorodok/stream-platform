package namedpipe

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
)

type NamedPipe struct {
	Path     string
	PipeFile *os.File
}

func NewNamedPipe() (*NamedPipe, error) {
	tempDir := os.TempDir()

	pipePath := filepath.Join(tempDir, fmt.Sprintf("pipe-%s.fifo", uuid.New()))

	if err := syscall.Mkfifo(pipePath, 0666); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("cannot create pipe. Err: %s", err)
	}

	return &NamedPipe{Path: pipePath}, nil
}

func (pipe NamedPipe) OpenAsWriteOnly() (pipeFile *os.File, err error) {
	pipeFile, err = os.OpenFile(pipe.Path, os.O_RDWR, os.ModeNamedPipe)

	if err != nil {
		return nil, err
	}

	pipe.PipeFile = pipeFile

	return
}

func (pipe NamedPipe) Close() {

}
