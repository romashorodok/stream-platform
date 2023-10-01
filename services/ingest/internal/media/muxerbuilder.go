package media

import (
	"io"
	"sync"
)

type MuxerBuilder struct {
	muxerReader io.Reader
	muxerWriter MuxerWriter
	writer      io.Writer

	mx sync.Mutex
}

// muxerWriter determine output type of muxer and mediaWriters must handle that type.
// if muxerWriter return webm opus all mediaWriters should be able handle webm
func (m *MuxerBuilder) Write(p []byte) (int, error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	n, err := m.muxerWriter.Write(p)
	if err != nil {
		return n, err
	}

	return n, err
}

// TODO: Possible directly write in writer function but i need read from m.muxerReader which is io.Pipe and this will block in writer.
// NOTE: Possible read by `io.ReadAtLeast' to escape blocking from pipe
func (m *MuxerBuilder) Mux() {
	for {
		io.Copy(m.writer, m.muxerReader)
	}
}

var _ Muxer = (*MuxerBuilder)(nil)

func NewMuxerBuilder(muxerWriter MuxerWriter, mediaWriters ...MediaWriter) *MuxerBuilder {
	writers := make([]io.Writer, len(mediaWriters))
	for i, mw := range mediaWriters {
		writers[i] = io.Writer(mw)
	}

	return &MuxerBuilder{
		muxerReader: muxerWriter.GetReader(),
		muxerWriter: muxerWriter,
		writer:      io.MultiWriter(writers...),
	}
}
