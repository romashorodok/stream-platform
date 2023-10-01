package media

import "io"

type DemuxerBuilder struct {
	demuxerReader DemuxerReader

	writer io.Writer
}

func (d *DemuxerBuilder) Demux() {
	for {
		sample, err := d.demuxerReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		_, _ = d.Write(sample)
	}
}

func (d *DemuxerBuilder) Write(p []byte) (n int, err error) {
	return d.writer.Write(p)
}

var _ Demuxer = (*DemuxerBuilder)(nil)

func NewDemuxerBuilder(demuxerReader DemuxerReader, mediaWriters ...MediaWriter) *DemuxerBuilder {
	writers := make([]io.Writer, len(mediaWriters))
	for i, mw := range mediaWriters {
		writers[i] = io.Writer(mw)
	}

	return &DemuxerBuilder{
		demuxerReader: demuxerReader,
		writer:        io.MultiWriter(writers...),
	}
}
