package media

import "io"

var receiveMTU = 1460

type DemuxerReader interface {
	Read() ([]byte, error)
}

// Demuxer takes container type like mp4 and return raw sample data
type Demuxer interface {
	io.Writer

	Demux()
}

// Muxer take resource sample and pack it into specific container type
type Muxer interface {
	io.Writer

	Mux()
}

type MuxerWriter interface {
	io.Writer

	GetReader() io.Reader
}

type MediaWriter interface {
	io.Writer
}
